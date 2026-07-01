import { useEffect, useRef, useState } from 'react';
import { startSocks, getSocksStatus, stopSocks } from '../api/socks';

interface Props {
  agentId: string;
  hostname: string;
  onClose: () => void;
}

type Phase = 'idle' | 'starting' | 'running' | 'stopping' | 'error';

export default function SocksModal({ agentId, hostname, onClose }: Props) {
  const [phase, setPhase] = useState<Phase>('starting');
  const [port, setPort] = useState(1080);
  const [connCount, setConnCount] = useState(0);
  const [errorMsg, setErrorMsg] = useState('');
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Check current status on mount
  useEffect(() => {
    let cancelled = false;

    async function check() {
      try {
        const status = await getSocksStatus(agentId);
        if (cancelled) return;
        if (status.active) {
          setPhase('running');
          setPort(status.port);
          setConnCount(status.connection_count);
          startPolling();
        } else {
          setPhase('idle');
        }
      } catch {
        if (!cancelled) setPhase('idle');
      }
    }

    check();

    return () => {
      cancelled = true;
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [agentId]);

  function startPolling() {
    if (pollRef.current) clearInterval(pollRef.current);
    pollRef.current = setInterval(async () => {
      try {
        const status = await getSocksStatus(agentId);
        if (status.active) {
          setConnCount(status.connection_count);
        } else {
          setPhase('idle');
          if (pollRef.current) clearInterval(pollRef.current);
        }
      } catch { /* ignore polling errors */ }
    }, 5000);
  }

  async function handleStart() {
    if (port < 1 || port > 65535) {
      setErrorMsg('Port must be between 1 and 65535');
      setPhase('error');
      return;
    }
    setPhase('starting');
    setErrorMsg('');
    try {
      await startSocks(agentId, port);
      setPhase('running');
      setConnCount(0);
      startPolling();
    } catch (err: unknown) {
      setPhase('error');
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setErrorMsg(msg || 'Failed to start SOCKS proxy');
    }
  }

  async function handleStop() {
    setPhase('stopping');
    try {
      await stopSocks(agentId);
      setPhase('idle');
      if (pollRef.current) clearInterval(pollRef.current);
    } catch (err: unknown) {
      setPhase('error');
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setErrorMsg(msg || 'Failed to stop SOCKS proxy');
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()} style={{ maxWidth: 480, minWidth: 380 }}>
        <div className="card-title" style={{ marginBottom: 16 }}>
          🌐 SOCKS Proxy — {hostname}
        </div>

        <div className="socks-container">
          {phase === 'idle' && (
            <>
              <p className="text-muted">Start a SOCKS5 proxy to tunnel traffic through this agent.</p>
              <div className="form-group" style={{ width: '100%' }}>
                <label className="form-label">Port</label>
                <input
                  type="number"
                  className="form-input"
                  value={port}
                  min={1}
                  max={65535}
                  onChange={(e) => setPort(Number(e.target.value))}
                />
              </div>
              <div className="screenshot-actions">
                <button className="btn btn-primary" onClick={handleStart}>
                  ▶ Start Proxy
                </button>
                <button className="btn btn-secondary" onClick={onClose}>
                  Close
                </button>
              </div>
            </>
          )}

          {phase === 'starting' && (
            <div className="screenshot-loading">
              <div className="spinner" />
              <span>Starting SOCKS5 proxy on port {port}...</span>
            </div>
          )}

          {phase === 'running' && (
            <>
              <div className="socks-status-active">
                ● Active on port {port}
              </div>
              <div className="socks-connection-count">
                {connCount} active connection{connCount !== 1 ? 's' : ''}
              </div>
              <p className="text-muted" style={{ fontSize: '0.8rem' }}>
                Use: socks5://&lt;teamserver&gt;:{port}
              </p>
              <div className="screenshot-actions">
                <button className="btn btn-danger" onClick={handleStop}>
                  ■ Stop Proxy
                </button>
                <button className="btn btn-secondary" onClick={onClose}>
                  Close
                </button>
              </div>
            </>
          )}

          {phase === 'stopping' && (
            <div className="screenshot-loading">
              <div className="spinner" />
              <span>Stopping SOCKS proxy...</span>
            </div>
          )}

          {phase === 'error' && (
            <>
              <div className="alert-error" style={{ width: '100%' }}>
                {errorMsg}
              </div>
              <div className="screenshot-actions">
                <button className="btn btn-primary" onClick={() => setPhase('idle')}>
                  🔄 Retry
                </button>
                <button className="btn btn-secondary" onClick={onClose}>
                  Close
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
