import { useEffect, useState } from 'react';
import { getAuditLogs } from '../api/dashboard';
import type { AuditLogEntry } from '../types';

export default function AuditLog() {
  const [logs, setLogs] = useState<AuditLogEntry[]>([]);
  const [page, setPage] = useState(1);

  useEffect(() => {
    async function load() {
      try {
        const { data } = await getAuditLogs(page);
        setLogs(data || []);
      } catch { /* ignore */ }
    }
    load();
  }, [page]);

  return (
    <div>
      <h1 className="page-title">Audit Log</h1>
      <div className="card">
        {logs.length === 0 ? (
          <p className="text-muted">No audit entries yet.</p>
        ) : (
          <>
            <table>
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Action</th>
                  <th>Resource</th>
                  <th>IP</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((l) => (
                  <tr key={l.id}>
                    <td className="text-muted">{new Date(l.created_at).toLocaleString()}</td>
                    <td>{l.action}</td>
                    <td>{l.resource_type}{l.resource_id ? `:${l.resource_id.slice(0, 8)}` : ''}</td>
                    <td className="text-muted">{l.ip_address || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="flex-between mt-16">
              <button className="btn btn-secondary btn-sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                Previous
              </button>
              <span className="text-muted">Page {page}</span>
              <button className="btn btn-secondary btn-sm" disabled={logs.length < 50} onClick={() => setPage((p) => p + 1)}>
                Next
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
