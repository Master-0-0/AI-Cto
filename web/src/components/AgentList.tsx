import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getAgents, deleteAgent } from '../api/agents';
import ContextMenu from './ContextMenu';
import ScreenshotModal from './ScreenshotModal';
import type { Agent } from '../types';

export default function AgentList() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [contextMenu, setContextMenu] = useState<{
    agent: Agent; x: number; y: number;
  } | null>(null);
  const [screenshotAgent, setScreenshotAgent] = useState<Agent | null>(null);

  async function load() {
    try {
      const data = await getAgents();
      setAgents(data);
    } catch { /* ignore */ }
  }

  useEffect(() => {
    load();
    const interval = setInterval(load, 10000);
    return () => clearInterval(interval);
  }, []);

  async function handleDelete(id: string) {
    if (!confirm('Delete this agent?')) return;
    await deleteAgent(id);
    load();
  }

  const osIcon = (os: string) => os === 'windows' ? '🪟' : os === 'linux' ? '🐧' : '🍎';

  return (
    <div>
      <h1 className="page-title">Agents</h1>
      <div className="card">
        {agents.length === 0 ? (
          <p className="text-muted">No agents connected. Deploy an implant to get started.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Hostname</th>
                <th>OS</th>
                <th>Username</th>
                <th>Exe</th>
                <th>Remote IP</th>
                <th>Status</th>
                <th>Last Seen</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {agents.map((a) => (
                <tr
                  key={a.id}
                  onContextMenu={(e) => {
                    e.preventDefault();
                    setContextMenu({ agent: a, x: e.clientX, y: e.clientY });
                  }}
                  style={{ cursor: 'context-menu' }}
                >
                  <td>
                    <Link to={`/agents/${a.id}`}>{a.hostname}</Link>
                  </td>
                  <td>{osIcon(a.os)} {a.os}/{a.arch}</td>
                  <td>{a.username}</td>
                  <td style={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>{a.exe_name}</td>
                  <td style={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>{a.remote_ip}</td>
                  <td><span className={`badge badge-${a.status}`}>{a.status}</span></td>
                  <td className="text-muted">{new Date(a.last_seen).toLocaleString()}</td>
                  <td>
                    <Link to={`/agents/${a.id}`} className="btn btn-primary btn-xs">Interact</Link>
                    {' '}
                    <button className="btn btn-danger btn-xs" onClick={() => handleDelete(a.id)}>Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          items={[
            {
              label: 'Take Screenshot',
              icon: '📸',
              onClick: () => setScreenshotAgent(contextMenu.agent),
            },
          ]}
          onClose={() => setContextMenu(null)}
        />
      )}
      {screenshotAgent && (
        <ScreenshotModal
          agentId={screenshotAgent.id}
          hostname={screenshotAgent.hostname}
          onClose={() => setScreenshotAgent(null)}
        />
      )}
    </div>
  );
}
