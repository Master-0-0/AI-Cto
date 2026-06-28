import { useEffect, useRef, useState } from 'react';
import { createTask, getTask } from '../api/tasks';
import type { Task } from '../types';

interface Props {
  agentId: string;
  hostname: string;
  onClose: () => void;
}

type Phase = 'creating' | 'polling' | 'displaying' | 'error';

function base64ToArrayBuffer(b64: string): ArrayBuffer {
  const binary = atob(b64);
  const buf = new ArrayBuffer(binary.length);
  const view = new Uint8Array(buf);
  for (let i = 0; i < binary.length; i++) {
    view[i] = binary.charCodeAt(i);
  }
  return buf;
}

function formatTimestamp(): string {
  const now = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
}

export default function ScreenshotModal({ agentId, hostname, onClose }: Props) {
  const [phase, setPhase] = useState<Phase>('creating');
  const [imageUrl, setImageUrl] = useState<string | null>(null);
  const [errorMsg, setErrorMsg] = useState('');
  const [elapsed, setElapsed] = useState(0);

  const taskIdRef = useRef<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const attemptsRef = useRef(0);
  const blobUrlRef = useRef<string | null>(null);

  const MAX_ATTEMPTS = 30; // 30 seconds

  // Cleanup blob URL on unmount
  useEffect(() => {
    return () => {
      if (blobUrlRef.current) {
        URL.revokeObjectURL(blobUrlRef.current);
      }
    };
  }, []);

  // Start screenshot flow on mount
  useEffect(() => {
    let cancelled = false;

    async function start() {
      // 1. Create the task
      setPhase('creating');
      try {
        const task = await createTask(agentId, {
          command: 'screenshot',
          timeout: 30,
        });
        if (cancelled) return;

        taskIdRef.current = task.id;
        setPhase('polling');

        // 2. Poll for completion
        const poll = setInterval(async () => {
          if (cancelled) {
            clearInterval(poll);
            return;
          }

          attemptsRef.current++;
          setElapsed(attemptsRef.current);

          if (attemptsRef.current > MAX_ATTEMPTS) {
            clearInterval(poll);
            setPhase('error');
            setErrorMsg('Screenshot timed out (30s)');
            return;
          }

          try {
            const result: Task = await getTask(task.id);

            if (result.status === 'completed' && result.output) {
              clearInterval(poll);
              try {
                const buf = base64ToArrayBuffer(result.output);
                const blob = new Blob([buf], { type: 'image/png' });
                const url = URL.createObjectURL(blob);
                blobUrlRef.current = url;
                setImageUrl(url);
                setPhase('displaying');
              } catch {
                setPhase('error');
                setErrorMsg('Failed to decode screenshot data');
              }
            } else if (result.status === 'error') {
              clearInterval(poll);
              setPhase('error');
              setErrorMsg(result.error_message || 'Screenshot command failed');
            } else if (result.status === 'cancelled') {
              clearInterval(poll);
              setPhase('error');
              setErrorMsg('Screenshot task was cancelled');
            }
            // 'pending' or 'sent' → keep polling
          } catch {
            // Network error — silently retry next interval
          }
        }, 1000);
        pollRef.current = poll;
      } catch (err: unknown) {
        if (!cancelled) {
          setPhase('error');
          const apiErr = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
          setErrorMsg(apiErr || 'Failed to create screenshot task');
        }
      }
    }

    start();

    return () => {
      cancelled = true;
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [agentId]);

  function handleDownload() {
    if (!imageUrl) return;
    const a = document.createElement('a');
    a.href = imageUrl;
    a.download = `screenshot-${hostname}-${formatTimestamp()}.png`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  }

  function handleRetry() {
    // Reset state and restart
    if (blobUrlRef.current) {
      URL.revokeObjectURL(blobUrlRef.current);
      blobUrlRef.current = null;
    }
    setImageUrl(null);
    setErrorMsg('');
    setElapsed(0);
    attemptsRef.current = 0;
    taskIdRef.current = null;

    // Re-trigger the effect by forcing remount of the flow
    // We use a key-less approach: restart manually
    setPhase('creating');

    async function restart() {
      try {
        const task = await createTask(agentId, {
          command: 'screenshot',
          timeout: 30,
        });
        taskIdRef.current = task.id;
        setPhase('polling');

        const poll = setInterval(async () => {
          attemptsRef.current++;
          setElapsed(attemptsRef.current);

          if (attemptsRef.current > MAX_ATTEMPTS) {
            clearInterval(poll);
            setPhase('error');
            setErrorMsg('Screenshot timed out (30s)');
            return;
          }

          try {
            const result = await getTask(task.id);
            if (result.status === 'completed' && result.output) {
              clearInterval(poll);
              try {
                const buf = base64ToArrayBuffer(result.output);
                const blob = new Blob([buf], { type: 'image/png' });
                const url = URL.createObjectURL(blob);
                blobUrlRef.current = url;
                setImageUrl(url);
                setPhase('displaying');
              } catch {
                setPhase('error');
                setErrorMsg('Failed to decode screenshot data');
              }
            } else if (result.status === 'error') {
              clearInterval(poll);
              setPhase('error');
              setErrorMsg(result.error_message || 'Screenshot command failed');
            } else if (result.status === 'cancelled') {
              clearInterval(poll);
              setPhase('error');
              setErrorMsg('Screenshot task was cancelled');
            }
          } catch { /* retry */ }
        }, 1000);
        pollRef.current = poll;
      } catch (err: unknown) {
        setPhase('error');
        const apiErr = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
        setErrorMsg(apiErr || 'Failed to create screenshot task');
      }
    }

    restart();
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()} style={{ maxWidth: '90vw', minWidth: 420 }}>
        <div className="card-title" style={{ marginBottom: 16 }}>
          📸 Screenshot — {hostname}
        </div>

        <div className="screenshot-container">
          {(phase === 'creating' || phase === 'polling') && (
            <div className="screenshot-loading">
              <div className="spinner" />
              <span>
                {phase === 'creating'
                  ? 'Creating screenshot task...'
                  : `Waiting for screenshot... (${elapsed}s)`}
              </span>
            </div>
          )}

          {phase === 'displaying' && imageUrl && (
            <>
              <img
                src={imageUrl}
                alt={`Screenshot of ${hostname}`}
                className="screenshot-image"
              />
              <div className="screenshot-actions">
                <button className="btn btn-primary" onClick={handleDownload}>
                  ⬇ Download PNG
                </button>
                <button className="btn btn-secondary" onClick={onClose}>
                  Close
                </button>
              </div>
            </>
          )}

          {phase === 'error' && (
            <>
              <div className="alert-error" style={{ width: '100%' }}>
                {errorMsg}
              </div>
              <div className="screenshot-actions">
                <button className="btn btn-primary" onClick={handleRetry}>
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
