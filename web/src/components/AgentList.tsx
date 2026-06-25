import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getAgents, deleteAgent } from '../api/agents';
import type { Agent } from '../types';

export default function AgentList() {
  const [agents, setAgents] = useState<Agent[]>([]);

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
                <th>Status</th>
                <th>Last Seen</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {agents.map((a) => (
                <tr key={a.id}>
                  <td>
                    <Link to={`/agents/${a.id}`}>{a.hostname}</Link>
                  </td>
                  <td>{osIcon(a.os)} {a.os}/{a.arch}</td>
                  <td>{a.username}</td>
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
    </div>
  );
}
