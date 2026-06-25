import { useState, FormEvent } from 'react';
import { createListener } from '../api/listeners';

interface Props {
  onClose: () => void;
  onCreated: () => void;
}

export default function ListenerForm({ onClose, onCreated }: Props) {
  const [name, setName] = useState('');
  const [protocol] = useState('tcp');
  const [host, setHost] = useState('0.0.0.0');
  const [port, setPort] = useState(4444);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await createListener({ name, protocol, host, port });
      onCreated();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setError(msg || 'Failed to create listener');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2 className="card-title">Create Listener</h2>
        {error && <div className="alert-error">{error}</div>}
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label className="form-label">Name</label>
            <input className="form-input" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="form-group">
            <label className="form-label">Protocol</label>
            <select className="form-select" value={protocol} disabled>
              <option value="tcp">TCP</option>
            </select>
          </div>
          <div className="form-group">
            <label className="form-label">Host</label>
            <input className="form-input" value={host} onChange={(e) => setHost(e.target.value)} required />
          </div>
          <div className="form-group">
            <label className="form-label">Port</label>
            <input className="form-input" type="number" min={1} max={65535} value={port} onChange={(e) => setPort(Number(e.target.value))} required />
          </div>
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
            <button className="btn btn-secondary" type="button" onClick={onClose}>Cancel</button>
            <button className="btn btn-primary" type="submit" disabled={loading}>
              {loading ? <span className="spinner" /> : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
