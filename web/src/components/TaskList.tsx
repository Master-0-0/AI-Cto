import type { Task } from '../types';

interface Props {
  tasks: Task[];
}

function decodeOutput(output?: string): string {
  if (!output) return '';
  try {
    return atob(output);
  } catch {
    return output;
  }
}

export default function TaskList({ tasks }: Props) {
  if (tasks.length === 0) {
    return <p className="text-muted">No tasks yet.</p>;
  }

  return (
    <div className="card">
      <div className="card-title">Tasks</div>
      <div style={{ maxHeight: 200, overflowY: 'auto' }}>
        <table>
          <thead style={{ position: 'sticky', top: 0, background: 'var(--bg-secondary)', zIndex: 1 }}>
            <tr>
              <th>Command</th>
              <th>Arguments</th>
              <th>Status</th>
              <th>Created</th>
              <th>Duration</th>
            </tr>
          </thead>
          <tbody>
            {tasks.map((t) => (
              <tr key={t.id}>
                <td>{t.command}</td>
                <td style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis' }}>{t.arguments || '-'}</td>
                <td><span className={`badge badge-${t.status}`}>{t.status}</span></td>
                <td className="text-muted">{new Date(t.created_at).toLocaleString()}</td>
                <td className="text-muted">{t.duration_ms ? `${t.duration_ms}ms` : '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export { type Props as TaskListProps, decodeOutput };
