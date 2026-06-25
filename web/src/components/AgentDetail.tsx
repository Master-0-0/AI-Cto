import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { getAgent } from '../api/agents';
import { createTask, getTasks } from '../api/tasks';
import TaskForm from './TaskForm';
import TaskList from './TaskList';
import AgentTerminal from './AgentTerminal';
import FileManager from './FileManager';
import type { Agent, Task } from '../types';

function getDefaultPath(agent: Agent): string | undefined {
  try {
    if (agent.metadata_json) {
      const meta = JSON.parse(agent.metadata_json);
      return meta.exe_dir || undefined;
    }
  } catch { /* ignore */ }
  return undefined;
}

export default function AgentDetail() {
  const { id } = useParams<{ id: string }>();
  const [agent, setAgent] = useState<Agent | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [editingHeartbeat, setEditingHeartbeat] = useState(false);
  const [heartbeatValue, setHeartbeatValue] = useState(60);

  async function loadAgent() {
    if (!id) return;
    try {
      const a = await getAgent(id);
      setAgent(a);
    } catch { /* ignore */ }
  }

  async function loadTasks() {
    if (!id) return;
    try {
      const t = await getTasks(id);
      setTasks(t);
    } catch { /* ignore */ }
  }

  useEffect(() => {
    loadAgent();
    loadTasks();
    const interval = setInterval(() => { loadAgent(); loadTasks(); }, 5000);
    return () => clearInterval(interval);
  }, [id]);

  if (!agent) {
    return <div className="spinner" />;
  }

  return (
    <div>
      <div className="flex-between mb-16">
        <h1 className="page-title" style={{ marginBottom: 0 }}>
          <Link to="/agents">Agents</Link> / {agent.hostname}
        </h1>
        <span className={`badge badge-${agent.status}`} style={{ fontSize: '0.9rem' }}>{agent.status}</span>
      </div>

      <div className="card mb-16">
        <div className="card-title">System Information</div>
        <table>
          <tbody>
            <tr><td style={{ width: 160, color: 'var(--text-secondary)' }}>Agent ID</td><td style={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>{agent.id}</td></tr>
            <tr><td style={{ color: 'var(--text-secondary)' }}>Hostname</td><td>{agent.hostname}</td></tr>
            <tr><td style={{ color: 'var(--text-secondary)' }}>OS</td><td>{agent.os} / {agent.arch}</td></tr>
            <tr><td style={{ color: 'var(--text-secondary)' }}>Username</td><td>{agent.username}</td></tr>
            <tr><td style={{ color: 'var(--text-secondary)' }}>PID</td><td>{agent.pid}</td></tr>
            <tr><td style={{ color: 'var(--text-secondary)' }}>First Seen</td><td>{new Date(agent.first_seen).toLocaleString()}</td></tr>
            <tr><td style={{ color: 'var(--text-secondary)' }}>Last Seen</td><td>{new Date(agent.last_seen).toLocaleString()}</td></tr>
            <tr>
              <td style={{ color: 'var(--text-secondary)' }}>Heartbeat</td>
              <td>
                {editingHeartbeat ? (
                  <span style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                    <input
                      className="form-input"
                      type="number"
                      min={1}
                      max={86400}
                      value={heartbeatValue}
                      onChange={(e) => setHeartbeatValue(Number(e.target.value))}
                      style={{ width: 80, padding: '2px 6px', fontSize: '0.85rem' }}
                      autoFocus
                    />
                    <span>s</span>
                    <button
                      className="btn btn-primary btn-xs"
                      onClick={async () => {
                        await createTask(agent.id, { command: 'sleep', arguments: String(heartbeatValue) });
                        setEditingHeartbeat(false);
                        loadAgent();
                      }}
                    >✓</button>
                    <button className="btn btn-secondary btn-xs" onClick={() => setEditingHeartbeat(false)}>✕</button>
                  </span>
                ) : (
                  <span style={{ cursor: 'pointer' }} onClick={() => { setHeartbeatValue(agent.heartbeat_interval); setEditingHeartbeat(true); }}>
                    {agent.heartbeat_interval}s <span style={{ color: 'var(--accent)', marginLeft: 8, fontSize: '0.8rem' }}>✎ change</span>
                  </span>
                )}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <FileManager agentId={agent.id} defaultPath={getDefaultPath(agent)} />

      <TaskForm agentId={agent.id} onTaskCreated={loadTasks} />
      <div style={{ marginBottom: 16 }}><TaskList tasks={tasks} /></div>
      <AgentTerminal agentId={agent.id} />
    </div>
  );
}
