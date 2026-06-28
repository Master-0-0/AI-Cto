import { useEffect, useState } from 'react';
import { getListeners, createListener, startListener, stopListener, deleteListener } from '../api/listeners';
import ListenerForm from './ListenerForm';
import StagerPanel from './StagerPanel';
import type { Listener } from '../types';

export default function ListenerList() {
  const [listeners, setListeners] = useState<Listener[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [expandedStager, setExpandedStager] = useState<string | null>(null);

  async function load() {
    try {
      const data = await getListeners();
      setListeners(data);
    } catch { /* ignore */ }
  }

  useEffect(() => {
    load();
    const interval = setInterval(load, 10000);
    return () => clearInterval(interval);
  }, []);

  async function handleStart(id: string) {
    await startListener(id);
    load();
  }

  async function handleStop(id: string) {
    await stopListener(id);
    load();
  }

  async function handleDelete(id: string) {
    if (!confirm('Delete this listener?')) return;
    await deleteListener(id);
    load();
  }

  function toggleStager(id: string) {
    setExpandedStager((prev) => (prev === id ? null : id));
  }

  return (
    <div>
      <div className="flex-between mb-16">
        <h1 className="page-title" style={{ marginBottom: 0 }}>Listeners</h1>
        <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ Create</button>
      </div>

      {showForm && (
        <ListenerForm
          onClose={() => setShowForm(false)}
          onCreated={() => { setShowForm(false); load(); }}
        />
      )}

      <div className="card">
        {listeners.length === 0 ? (
          <p className="text-muted">No listeners created yet.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Protocol</th>
                <th>Address</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {listeners.map((l) => (
                <>
                  <tr key={l.id}>
                    <td>{l.name || l.id.slice(0, 8)}</td>
                    <td>{l.protocol.toUpperCase()}</td>
                    <td>{l.host}:{l.port}</td>
                    <td><span className={`badge badge-${l.status}`}>{l.status}</span></td>
                    <td>
                      <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                        <button
                          className="btn btn-info btn-xs"
                          onClick={() => toggleStager(l.id)}
                        >
                          {expandedStager === l.id ? '收起' : '上线方式'}
                        </button>
                        {l.status === 'stopped' && (
                          <button className="btn btn-primary btn-xs" onClick={() => handleStart(l.id)}>Start</button>
                        )}
                        {l.status === 'running' && (
                          <button className="btn btn-secondary btn-xs" onClick={() => handleStop(l.id)}>Stop</button>
                        )}
                        {l.status !== 'running' && (
                          <button className="btn btn-danger btn-xs" onClick={() => handleDelete(l.id)}>Delete</button>
                        )}
                      </div>
                    </td>
                  </tr>
                  {expandedStager === l.id && (
                    <tr key={`${l.id}-stager`} className="stager-expand-row">
                      <td colSpan={5}>
                        <StagerPanel listenerId={l.id} />
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
