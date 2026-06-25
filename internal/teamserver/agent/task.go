package agent

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"c2/internal/teamserver/database"
)

// TaskManager handles task queue operations.
type TaskManager struct {
	db             *sql.DB
	mu             sync.Mutex
	pendingSignals map[string]chan struct{} // agentID -> wake-up signal
}

// NewTaskManager creates a new TaskManager.
func NewTaskManager(db *sql.DB) *TaskManager {
	return &TaskManager{
		db:             db,
		pendingSignals: make(map[string]chan struct{}),
	}
}

// Enqueue creates a new task for an agent.
func (tm *TaskManager) Enqueue(task *database.Task) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task.ID = uuid.New().String()
	task.Status = "pending"
	task.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	_, err := tm.db.Exec(
		`INSERT INTO tasks (id, agent_id, operator_id, command, arguments, status, timeout_sec, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.AgentID, task.OperatorID, task.Command, task.Arguments,
		task.Status, task.TimeoutSec, task.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting task: %w", err)
	}

	// Signal any waiting poller
	tm.signalAgent(task.AgentID)

	return nil
}

// Dequeue retrieves pending tasks for an agent and marks them as sent.
func (tm *TaskManager) Dequeue(agentID string, timeout time.Duration) ([]*database.Task, error) {
	tm.mu.Lock()

	// Check for pending tasks
	tasks, err := tm.getPending(agentID)
	if err != nil {
		tm.mu.Unlock()
		return nil, err
	}

	if len(tasks) > 0 {
		// Mark as sent
		now := time.Now().UTC().Format(time.RFC3339)
		for _, t := range tasks {
			tm.db.Exec("UPDATE tasks SET status = 'sent', sent_at = ? WHERE id = ?", now, t.ID)
			t.Status = "sent"
		}
		tm.mu.Unlock()
		return tasks, nil
	}

	// No tasks available, wait for signal or timeout
	ch := tm.getOrCreateSignal(agentID)
	tm.mu.Unlock()

	select {
	case <-ch:
		// Re-check for tasks
		tm.mu.Lock()
		tasks, err = tm.getPending(agentID)
		if err != nil {
			tm.mu.Unlock()
			return nil, err
		}
		if len(tasks) > 0 {
			now := time.Now().UTC().Format(time.RFC3339)
			for _, t := range tasks {
				tm.db.Exec("UPDATE tasks SET status = 'sent', sent_at = ? WHERE id = ?", now, t.ID)
				t.Status = "sent"
			}
		}
		tm.mu.Unlock()
		return tasks, nil
	case <-time.After(timeout):
		return nil, nil // No tasks within timeout
	}
}

// Complete updates a task as completed.
func (tm *TaskManager) Complete(taskID, status, output, errorMsg string, durationMs int) error {
	now := time.Now().UTC().Format(time.RFC3339)
	encodedOutput := base64.StdEncoding.EncodeToString([]byte(output))

	_, err := tm.db.Exec(
		`UPDATE tasks SET status = ?, output = ?, error_message = ?, completed_at = ?, duration_ms = ?
		 WHERE id = ?`,
		status, encodedOutput, errorMsg, now, durationMs, taskID,
	)
	if err != nil {
		return fmt.Errorf("completing task %s: %w", taskID, err)
	}

	return nil
}

// ListByAgent returns tasks for a given agent, optionally filtered by status.
func (tm *TaskManager) ListByAgent(agentID, statusFilter string) ([]*database.Task, error) {
	var rows *sql.Rows
	var err error

	if statusFilter != "" {
		rows, err = tm.db.Query(
			"SELECT id, agent_id, operator_id, command, arguments, status, output, error_message, timeout_sec, created_at, sent_at, completed_at, duration_ms FROM tasks WHERE agent_id = ? AND status = ? ORDER BY created_at DESC LIMIT 100",
			agentID, statusFilter,
		)
	} else {
		rows, err = tm.db.Query(
			"SELECT id, agent_id, operator_id, command, arguments, status, output, error_message, timeout_sec, created_at, sent_at, completed_at, duration_ms FROM tasks WHERE agent_id = ? ORDER BY created_at DESC LIMIT 100",
			agentID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

// Get returns a single task by ID.
func (tm *TaskManager) Get(taskID string) (*database.Task, error) {
	var t database.Task
	err := tm.db.QueryRow(
		"SELECT id, agent_id, operator_id, command, arguments, status, output, error_message, timeout_sec, created_at, sent_at, completed_at, duration_ms FROM tasks WHERE id = ?",
		taskID,
	).Scan(&t.ID, &t.AgentID, &t.OperatorID, &t.Command, &t.Arguments, &t.Status, &t.Output, &t.ErrorMessage, &t.TimeoutSec, &t.CreatedAt, &t.SentAt, &t.CompletedAt, &t.DurationMs)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Delete removes a pending task.
func (tm *TaskManager) Delete(taskID string) error {
	_, err := tm.db.Exec("DELETE FROM tasks WHERE id = ? AND status = 'pending'", taskID)
	return err
}

// NotifyAgent signals that a task is available for the given agent.
func (tm *TaskManager) NotifyAgent(agentID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.signalAgent(agentID)
}

func (tm *TaskManager) getPending(agentID string) ([]*database.Task, error) {
	rows, err := tm.db.Query(
		"SELECT id, agent_id, operator_id, command, arguments, status, output, error_message, timeout_sec, created_at, sent_at, completed_at, duration_ms FROM tasks WHERE agent_id = ? AND status = 'pending' ORDER BY created_at ASC LIMIT 10",
		agentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (tm *TaskManager) getOrCreateSignal(agentID string) chan struct{} {
	ch, ok := tm.pendingSignals[agentID]
	if !ok {
		ch = make(chan struct{}, 1)
		tm.pendingSignals[agentID] = ch
	}
	return ch
}

func (tm *TaskManager) signalAgent(agentID string) {
	ch, ok := tm.pendingSignals[agentID]
	if ok {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func scanTasks(rows *sql.Rows) ([]*database.Task, error) {
	var tasks []*database.Task
	for rows.Next() {
		var t database.Task
		if err := rows.Scan(&t.ID, &t.AgentID, &t.OperatorID, &t.Command, &t.Arguments, &t.Status, &t.Output, &t.ErrorMessage, &t.TimeoutSec, &t.CreatedAt, &t.SentAt, &t.CompletedAt, &t.DurationMs); err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		tasks = append(tasks, &t)
	}
	return tasks, nil
}
