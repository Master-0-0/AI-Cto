import { useEffect, useState } from 'react';
import StatCard from './StatCard';
import { getStats } from '../api/dashboard';
import { getAgents } from '../api/agents';
import type { DashboardStats, Agent } from '../types';

export default function Dashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [recentAgents, setRecentAgents] = useState<Agent[]>([]);

  useEffect(() => {
    async function load() {
      try {
        const [s, agents] = await Promise.all([getStats(), getAgents()]);
        setStats(s);
        setRecentAgents(agents.slice(0, 10));
      } catch {
        // ignore
      }
    }
    load();
    const interval = setInterval(load, 15000);
    return () => clearInterval(interval);
  }, []);

  if (!stats) {
    return <div className="spinner" />;
  }

  return (
    <div>
      <h1 className="page-title">Dashboard</h1>
      <div className="stat-grid">
        <StatCard title="Total Agents" value={stats.agent_count} />
        <StatCard title="Active Agents" value={stats.active_agents} color="#4caf50" />
        <StatCard title="Listeners Running" value={stats.running_listeners} color="#2196f3" />
        <StatCard title="Tasks Today" value={stats.tasks_today} color="#ff9800" />
      </div>

      <div className="card">
        <div className="card-title">Recent Agents</div>
        {recentAgents.length === 0 ? (
          <p className="text-muted">No agents connected yet.</p>
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
              </tr>
            </thead>
            <tbody>
              {recentAgents.map((a) => (
                <tr key={a.id}>
                  <td>{a.hostname}</td>
                  <td>{a.os}/{a.arch}</td>
                  <td>{a.username}</td>
                  <td style={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>{a.exe_name}</td>
                  <td style={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>{a.remote_ip}</td>
                  <td><span className={`badge badge-${a.status}`}>{a.status}</span></td>
                  <td className="text-muted">{new Date(a.last_seen).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
