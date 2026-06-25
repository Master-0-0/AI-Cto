import { useState, useEffect, FormEvent } from 'react';
import { buildImplant } from '../api/builder';
import { getListeners } from '../api/listeners';
import type { BuildResponse, Listener } from '../types';

export default function BuildPanel() {
  const [os, setOs] = useState('windows');
  const [arch, setArch] = useState('amd64');
  const [listenerId, setListenerId] = useState('');
  const [connectionType, setConnectionType] = useState('reverse');
  const [listeners, setListeners] = useState<Listener[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<BuildResponse | null>(null);

  useEffect(() => {
    getListeners()
      .then((data) => {
        setListeners(data);
        const running = data.filter((l) => l.status === 'running');
        if (running.length > 0) {
          setListenerId(running[0].id);
        } else if (data.length > 0) {
          setListenerId(data[0].id);
        }
      })
      .catch(() => {});
  }, []);

  async function handleBuild(e: FormEvent) {
    e.preventDefault();
    setError('');
    setResult(null);

    if (!listenerId) {
      setError('Please select a listener.');
      return;
    }

    setLoading(true);

    try {
      const res = await buildImplant({
        os,
        arch,
        listener_id: listenerId,
        connection_type: connectionType,
      });
      setResult(res);
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setError(msg || 'Build failed. Ensure Go toolchain is available on the server.');
    } finally {
      setLoading(false);
    }
  }

  const selectedListener = listeners.find((l) => l.id === listenerId);

  return (
    <div>
      <h1 className="page-title">Build Implant</h1>
      <div className="card" style={{ maxWidth: 540 }}>
        <div className="card-title">Generate Payload</div>

        {error && <div className="alert-error">{error}</div>}

        {result && (
          <div style={{
            background: 'rgba(76, 175, 80, 0.15)',
            color: 'var(--success)',
            padding: '10px 14px',
            borderRadius: 'var(--radius)',
            marginBottom: 16,
            fontSize: '0.85rem',
          }}>
            <div><strong>Build successful!</strong></div>
            <div>File: {result.filename}</div>
            <div>SHA256: <span style={{ fontFamily: 'monospace', fontSize: '0.7rem', wordBreak: 'break-all' }}>{result.checksum}</span></div>
            <div style={{ marginTop: 8 }}>
              <a href={result.download_url} className="btn btn-primary btn-sm" download>
                Download
              </a>
            </div>
          </div>
        )}

        <form onSubmit={handleBuild}>
          <div className="form-group">
            <label className="form-label">Listener</label>
            {listeners.length === 0 ? (
              <p className="text-muted" style={{ fontSize: '0.85rem' }}>
                No listeners available. Create and start a listener first.
              </p>
            ) : (
              <select
                className="form-select"
                value={listenerId}
                onChange={(e) => setListenerId(e.target.value)}
                required
              >
                <option value="">-- Select a listener --</option>
                {listeners.map((l) => (
                  <option key={l.id} value={l.id}>
                    {l.name || l.id.slice(0, 8)} — {l.protocol.toUpperCase()} {l.host}:{l.port} [{l.status}]
                  </option>
                ))}
              </select>
            )}
            {selectedListener && (
              <p className="text-muted" style={{ fontSize: '0.8rem', marginTop: 4 }}>
                Implant will connect to {selectedListener.host}:{selectedListener.port} ({connectionType === 'reverse' ? 'reverse' : 'bind'} mode)
              </p>
            )}
          </div>

          <div className="form-group">
            <label className="form-label">Connection Type</label>
            <select
              className="form-select"
              value={connectionType}
              onChange={(e) => setConnectionType(e.target.value)}
            >
              <option value="reverse">Reverse (Implant connects to server)</option>
              <option value="bind" disabled>Bind (Server connects to implant) — Coming soon</option>
            </select>
          </div>

          <div className="form-group">
            <label className="form-label">Target OS</label>
            <select className="form-select" value={os} onChange={(e) => setOs(e.target.value)}>
              <option value="windows">Windows</option>
              <option value="linux">Linux</option>
              <option value="darwin">macOS</option>
            </select>
          </div>

          <div className="form-group">
            <label className="form-label">Architecture</label>
            <select className="form-select" value={arch} onChange={(e) => setArch(e.target.value)}>
              <option value="amd64">amd64</option>
              <option value="386">386</option>
              <option value="arm64">arm64</option>
            </select>
          </div>

          <button className="btn btn-primary" type="submit" disabled={loading || listeners.length === 0}>
            {loading ? <span className="spinner" /> : 'Build Implant'}
          </button>
        </form>
      </div>
    </div>
  );
}
