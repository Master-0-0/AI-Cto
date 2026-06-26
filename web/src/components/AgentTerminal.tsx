import { useState, useRef, useEffect, useCallback } from 'react';
import { createTask, getTasks } from '../api/tasks';
import { useWebSocket } from '../hooks/useWebSocket';
import { decodeBase64 } from '../utils';
import type { Task } from '../types';

interface Props {
  agentId: string;
}

interface LogEntry {
  id: string;
  command: string;
  output: string;
  status: string;
  timestamp: Date;
}

function decodeOutput(output?: string): string {
  if (!output) return '';
  try {
    return decodeBase64(output);
  } catch {
    return output;
  }
}

export default function AgentTerminal({ agentId }: Props) {
  const [input, setInput] = useState('');
  const [log, setLog] = useState<LogEntry[]>([]);
  const outputRef = useRef<HTMLDivElement>(null);
  const seenIds = useRef<Set<string>>(new Set());

  // Load existing completed tasks on mount
  useEffect(() => {
    getTasks(agentId).then((tasks) => {
      const entries: LogEntry[] = [];
      for (const t of tasks) {
        if (t.status === 'completed' || t.status === 'error') {
          if (seenIds.current.has(t.id)) continue;
          seenIds.current.add(t.id);
          entries.push({
            id: t.id,
            command: `${t.command} ${t.arguments || ''}`.trim(),
            output: t.status === 'error' ? (t.error_message || 'Error') : decodeOutput(t.output),
            status: t.status,
            timestamp: new Date(t.completed_at || t.created_at),
          });
        }
      }
      if (entries.length > 0) {
        setLog((prev) => [...prev, ...entries].sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime()));
      }
    }).catch(() => {});
  }, [agentId]);

  // WebSocket for real-time task results
  const { lastMessage, connectionStatus } = useWebSocket(agentId, {
    onMessage: () => {}, // handled via lastMessage
  });

  // Process incoming WebSocket messages
  useEffect(() => {
    if (!lastMessage?.data) return;
    try {
      const msg = JSON.parse(lastMessage.data);
      if (msg.type === 'task_complete' && msg.data) {
        const result = msg.data as {
          TaskID?: string;
          task_id?: string;
          Command?: string;
          Arguments?: string;
          Status?: string;
          status?: string;
          Output?: string;
          output?: string;
          ErrorMsg?: string;
          error_msg?: string;
        };
        const taskId = result.TaskID || result.task_id || '';
        const status = result.Status || result.status || 'completed';
        const rawOutput = result.Output || result.output || '';
        const errMsg = result.ErrorMsg || result.error_msg || '';

        setLog((prev) => {
          // Skip if we already have this task
          if (prev.some((e) => e.id === taskId)) return prev;

          const output = status === 'error' ? errMsg || 'Error' : decodeOutput(rawOutput);
          const command = `${result.Command || 'shell'} ${result.Arguments || ''}`.trim();

          return [...prev, {
            id: taskId,
            command,
            output,
            status,
            timestamp: new Date(),
          }];
        });
      } else if (msg.type === 'agent_status' || msg.type === 'status_change') {
        setLog((prev) => [
          ...prev,
          {
            id: `status-${Date.now()}`,
            command: '[system]',
            output: `Agent status changed to: ${msg.data}`,
            status: 'completed',
            timestamp: new Date(),
          },
        ]);
      }
    } catch { /* ignore malformed messages */ }
  }, [lastMessage]);

  // Auto-scroll to bottom
  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [log]);

  async function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && input.trim()) {
      const cmd = input.trim();
      setInput('');

      const pendingId = `pending-${Date.now()}`;
      setLog((prev) => [
        ...prev,
        { id: pendingId, command: cmd, output: '...', status: 'pending', timestamp: new Date() },
      ]);

      try {
        // Create task via REST for reliable delivery (per design decision #6)
        await createTask(agentId, { command: 'shell', arguments: cmd });
      } catch {
        // Update pending entry on error
        setLog((prev) =>
          prev.map((e) =>
            e.id === pendingId ? { ...e, output: 'Failed to send command', status: 'error' } : e
          )
        );
      }
    }
  }

  return (
    <div className="card">
      <div className="flex-between">
        <div className="card-title">Terminal</div>
        <span className={`badge badge-${connectionStatus === 'connected' ? 'active' : 'inactive'}`} style={{ fontSize: '0.7rem' }}>
          {connectionStatus}
        </span>
      </div>
      <div className="terminal" ref={outputRef}>
        {log.map((entry) => (
          <div key={entry.id} style={{ marginBottom: 8 }}>
            <div style={{ color: '#888', fontSize: '0.75rem', marginBottom: 2 }}>
              [{entry.timestamp.toLocaleTimeString()}] $ {entry.command}
            </div>
            <div className="terminal-output">{entry.output}</div>
          </div>
        ))}
        <div className="terminal-input-line">
          <span className="terminal-prompt">$</span>
          <input
            className="terminal-input"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            autoFocus
            spellCheck={false}
          />
        </div>
      </div>
    </div>
  );
}
