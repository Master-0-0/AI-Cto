import { useState, FormEvent } from 'react';
import { createTask } from '../api/tasks';

interface Props {
  agentId: string;
  onTaskCreated: () => void;
}

export default function TaskForm({ agentId, onTaskCreated }: Props) {
  const [command, setCommand] = useState('shell');
  const [args, setArgs] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await createTask(agentId, { command, arguments: args });
      setArgs('');
      onTaskCreated();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setError(msg || 'Failed to create task');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="card" style={{ marginBottom: 16 }}>
      <div className="card-title">Send Command</div>
      {error && <div className="alert-error">{error}</div>}
      <form onSubmit={handleSubmit} style={{ display: 'flex', gap: 8 }}>
        <select className="form-select" value={command} onChange={(e) => setCommand(e.target.value)} style={{ width: 120 }}>
          <option value="shell">Shell</option>
          <option value="sleep">Sleep</option>
          <option value="exit">Exit</option>
        </select>
        <input
          className="form-input"
          value={args}
          onChange={(e) => setArgs(e.target.value)}
          placeholder={command === 'shell' ? 'e.g. whoami' : command === 'sleep' ? 'seconds' : ''}
          required={command !== 'exit'}
          style={{ flex: 1 }}
        />
        <button className="btn btn-primary" type="submit" disabled={loading}>
          {loading ? <span className="spinner" /> : 'Send'}
        </button>
      </form>
    </div>
  );
}
