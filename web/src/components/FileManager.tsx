import { useState, useEffect, useRef } from 'react';
import { createTask, getTasks } from '../api/tasks';
import { decodeBase64 } from '../utils';

interface Props {
  agentId: string;
  defaultPath?: string;
}

interface FileEntry {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  mod_time: string;
}

interface TreeNode extends FileEntry {
  children: TreeNode[];
  loaded: boolean;
  expanded: boolean;
  depth: number;
}

export default function FileManager({ agentId, defaultPath }: Props) {
  const [roots, setRoots] = useState<TreeNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [selectedFile, setSelectedFile] = useState<FileEntry | null>(null);
  const [fileContent, setFileContent] = useState('');
  const [editedContent, setEditedContent] = useState('');
  const [editing, setEditing] = useState(false);
  const [breadcrumb, setBreadcrumb] = useState<string[]>([]);
  const [uploading, setUploading] = useState(false);
  const uploadRef = useRef<HTMLInputElement>(null);

  // Load drives / root on mount
  useEffect(() => {
    loadDrives();
  }, [agentId]);

  async function loadDrives() {
    setLoading(true);
    setError('');
    try {
      await createTask(agentId, { command: 'listdrives', arguments: '' });
      await waitForResult((task) => {
        if (task.command === 'listdrives' && task.status === 'completed' && task.output) {
          try {
            const raw = decodeBase64(task.output);
            const data = JSON.parse(raw);
            const nodes: TreeNode[] = (data.drives || []).map((d: any) => ({
              name: d.name,
              path: d.path,
              is_dir: true,
              size: 0,
              mod_time: '',
              children: [],
              loaded: false,
              expanded: false,
              depth: 0,
            }));
            setRoots(nodes);
          } catch { setError('Failed to parse drives'); }
          return true;
        }
        if (task.status === 'error') { setError(task.error_message || 'Failed'); return true; }
        return false;
      });
    } catch { setError('Failed to list drives'); }
    finally { setLoading(false); }
  }

  async function expandNode(node: TreeNode) {
    if (node.loaded) {
      // Toggle collapse
      setRoots((prev) => toggleNode(prev, node.path, !node.expanded));
      updateBreadcrumb(node);
      return;
    }
    // Lazy load directory contents
    setLoading(true);
    setError('');
    try {
      await createTask(agentId, { command: 'listdir', arguments: node.path });
      await waitForResult((task) => {
        if (task.command === 'listdir' && task.status === 'completed' && task.output) {
          try {
            const raw = decodeBase64(task.output);
            const data = JSON.parse(raw);
            const children: TreeNode[] = (data.files || []).map((f: FileEntry) => ({
              ...f,
              children: [],
              loaded: false,
              expanded: false,
              depth: node.depth + 1,
            }));
            setRoots((prev) => populateChildren(prev, node.path, children));
          } catch { setError('Failed to parse directory'); }
          return true;
        }
        if (task.status === 'error') { setError(task.error_message || 'Failed'); return true; }
        return false;
      });
    } catch { setError('Failed to load directory'); }
    finally { setLoading(false); }
    updateBreadcrumb(node);
  }

  function toggleNode(nodes: TreeNode[], path: string, expanded: boolean): TreeNode[] {
    return nodes.map((n) => {
      if (n.path === path) return { ...n, expanded };
      if (n.children.length > 0) return { ...n, children: toggleNode(n.children, path, expanded) };
      return n;
    });
  }

  function populateChildren(nodes: TreeNode[], path: string, children: TreeNode[]): TreeNode[] {
    return nodes.map((n) => {
      if (n.path === path) return { ...n, children, loaded: true, expanded: true };
      if (n.children.length > 0) return { ...n, children: populateChildren(n.children, path, children) };
      return n;
    });
  }

  function updateBreadcrumb(node: TreeNode) {
    // Build breadcrumb from root to this node
    const parts: string[] = [];
    let current = node;
    while (current) {
      parts.unshift(current.name);
      // Find parent
      current = findParent(roots, current.path);
    }
    setBreadcrumb(parts.length > 0 ? parts : [node.name]);
  }

  function findParent(nodes: TreeNode[], childPath: string): TreeNode | null {
    for (const n of nodes) {
      if (n.children.some((c) => c.path === childPath)) return n;
      const found = findParent(n.children, childPath);
      if (found) return found;
    }
    return null as any;
  }

  // Read file
  async function readFile(file: FileEntry) {
    if (file.is_dir) {
      const node = findNode(roots, file.path);
      if (node) expandNode(node);
      return;
    }
    setLoading(true);
    try {
      await createTask(agentId, { command: 'readfile', arguments: file.path });
      await waitForResult((task) => {
        if (task.command === 'readfile' && task.status === 'completed' && task.output) {
          try {
            const raw = decodeBase64(task.output);
            const data = JSON.parse(raw);
            setSelectedFile(file);
            const content = decodeBase64(data.content);
            setFileContent(content);
            setEditedContent(content);
            setEditing(false);
          } catch { setError('Failed to decode'); }
          return true;
        }
        if (task.status === 'error') { setError(task.error_message || 'Failed'); return true; }
        return false;
      });
    } catch { setError('Failed to read file'); }
    finally { setLoading(false); }
  }

  // Save file
  async function saveFile() {
    if (!selectedFile) return;
    setLoading(true);
    try {
      const enc = new TextEncoder();
      const bytes = enc.encode(editedContent);
      const contentB64 = btoa(String.fromCharCode(...bytes));
      const args = `${selectedFile.path}|${contentB64}`;
      await createTask(agentId, { command: 'writefile', arguments: args });
      await waitForResult((task) => {
        if (task.command === 'writefile' && task.status === 'completed') {
          setFileContent(editedContent);
          setEditing(false);
          return true;
        }
        if (task.status === 'error') { setError(task.error_message || 'Failed'); return true; }
        return false;
      });
    } catch { setError('Failed to save'); }
    finally { setLoading(false); }
  }

  // Upload
  async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    const reader = new FileReader();
    reader.onload = async () => {
      const arr = new Uint8Array(reader.result as ArrayBuffer);
      const contentB64 = btoa(String.fromCharCode(...arr));
      const cwd = breadcrumb.length > 0 ? findNodeByBreadcrumb()?.path || defaultPath || '.' : defaultPath || '.';
      const remotePath = `${cwd}/${file.name}`.replace(/\/\//g, '/').replace(/\\/g, '/');
      const args = `${remotePath}|${contentB64}`;
      try {
        await createTask(agentId, { command: 'writefile', arguments: args });
        await waitForResult((task) => {
          if (task.command === 'writefile' && task.status === 'completed') {
            // Refresh current expanded node
            const target = findNodeByBreadcrumb();
            if (target && target.loaded) {
              expandNode({ ...target, loaded: false });
            }
            return true;
          }
          if (task.status === 'error') { setError(task.error_message || 'Failed'); return true; }
          return false;
        });
      } catch { setError('Upload failed'); }
      setUploading(false);
    };
    reader.readAsArrayBuffer(file);
  }

  function findNodeByBreadcrumb(): TreeNode | null {
    let current = roots;
    let result: TreeNode | null = null;
    for (const part of breadcrumb) {
      const found = current.find((n) => n.name === part);
      if (found) { result = found; current = found.children; }
      else break;
    }
    return result;
  }

  // Download
  function downloadFile(file: FileEntry) {
    if (!selectedFile || !fileContent) {
      readFile(file).then(() => {
        setTimeout(() => downloadBlob(file.name, fileContent), 200);
      });
    } else {
      downloadBlob(file.name, fileContent);
    }
  }

  function downloadBlob(name: string, content: string) {
    const blob = new Blob([content], { type: 'application/octet-stream' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = name;
    a.click();
    URL.revokeObjectURL(url);
  }

  function waitForResult(check: (task: any) => boolean): Promise<void> {
    return new Promise((resolve, reject) => {
      let attempts = 0;
      const interval = setInterval(async () => {
        if (++attempts > 30) { clearInterval(interval); reject(new Error('Timeout')); return; }
        try {
          const tasks = await getTasks(agentId);
          for (const t of tasks) {
            if (t.status === 'completed' || t.status === 'error') {
              if (check(t)) { clearInterval(interval); resolve(); return; }
            }
          }
        } catch { /* retry */ }
      }, 500);
    });
  }

  function findNode(nodes: TreeNode[], path: string): TreeNode | null {
    for (const n of nodes) {
      if (n.path === path) return n;
      const found = findNode(n.children, path);
      if (found) return found;
    }
    return null;
  }

  function formatSize(size: number): string {
    if (size < 1024) return `${size} B`;
    if (size < 1048576) return `${(size / 1024).toFixed(1)} KB`;
    return `${(size / 1048576).toFixed(1)} MB`;
  }

  function fileIcon(name: string, isDir: boolean): string {
    if (isDir) return '📁';
    const ext = name.split('.').pop()?.toLowerCase() || '';
    const icons: Record<string, string> = { txt: '📄', log: '📄', conf: '⚙', yaml: '⚙', yml: '⚙', json: '📋', xml: '📋', ini: '⚙', exe: '⚡', dll: '🔧', bat: '📜', cmd: '📜', ps1: '📜', sh: '📜', py: '🐍', js: '📜', zip: '📦', png: '🖼', jpg: '🖼' };
    return icons[ext] || '📄';
  }

  // Render a tree node and its children recursively
  function renderTree(nodes: TreeNode[]): JSX.Element {
    return (
      <div className="tree-list">
        {nodes.map((n) => (
          <div key={n.path}>
            <div
              className="tree-row"
              style={{ paddingLeft: n.depth * 16 + 4 }}
              onClick={() => n.is_dir ? expandNode(n) : readFile(n)}
              onDoubleClick={() => n.is_dir ? expandNode(n) : undefined}
            >
              {/* Tree connector lines */}
              {n.depth > 0 && (
                <span className="tree-connector" style={{ marginRight: 4 }}>
                  {n.expanded ? '└ ' : '├ '}
                </span>
              )}
              <span style={{ marginRight: 6 }}>{n.expanded ? '📂' : fileIcon(n.name, n.is_dir)}</span>
              <span style={{ color: n.is_dir ? 'var(--accent)' : undefined, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{n.name}</span>
              {!n.is_dir && <span className="text-muted" style={{ fontSize: '0.75rem', marginLeft: 8 }}>{formatSize(n.size)}</span>}
            </div>
            {n.expanded && n.children.length > 0 && renderTree(n.children)}
            {n.expanded && n.loaded && n.children.length === 0 && (
              <div style={{ paddingLeft: (n.depth + 1) * 16 + 20, color: 'var(--text-muted)', fontSize: '0.8rem', paddingTop: 4, paddingBottom: 4 }}>
                (empty)
              </div>
            )}
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="card mb-16">
      <div className="flex-between">
        <div className="card-title">File Manager</div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <button className="btn btn-secondary btn-xs" onClick={loadDrives} disabled={loading}>↻</button>
          <button className="btn btn-primary btn-xs" onClick={() => uploadRef.current?.click()} disabled={uploading}>
            {uploading ? '⏳' : '📤'} Upload
          </button>
          <input ref={uploadRef} type="file" style={{ display: 'none' }} onChange={handleUpload} />
        </div>
      </div>

      {/* Breadcrumb path */}
      {breadcrumb.length > 0 && (
        <div style={{
          padding: '6px 10px', marginBottom: 8,
          background: 'var(--bg-input)', borderRadius: 'var(--radius)',
          fontFamily: 'monospace', fontSize: '0.8rem',
          color: 'var(--accent)',
        }}>
          {breadcrumb.map((part, i) => (
            <span key={i}>
              {i > 0 && <span style={{ color: 'var(--text-muted)', margin: '0 4px' }}>▸</span>}
              <span
                style={{ cursor: 'pointer' }}
                onClick={() => {
                  // Find and expand to this breadcrumb level
                  if (i === 0) {
                    const root = roots.find((r) => r.name === part);
                    if (root) expandNode({ ...root, loaded: false });
                  }
                }}
              >{part}</span>
            </span>
          ))}
        </div>
      )}

      {error && <div className="alert-error">{error}</div>}

      <div style={{ display: 'flex', gap: 0, minHeight: 280 }}>
        {/* Tree panel */}
        <div style={{ flex: 1, maxHeight: 420, overflowY: 'auto', borderRight: '1px solid var(--border)' }}>
          {loading && roots.length === 0 ? (
            <div className="spinner" style={{ margin: 20 }} />
          ) : (
            renderTree(roots)
          )}
        </div>

        {/* File content / editor */}
        <div style={{ flex: 1, maxHeight: 420, overflowY: 'auto', padding: '0 0 0 12px' }}>
          {selectedFile ? (
            <div>
              <div className="flex-between" style={{ marginBottom: 8 }}>
                <span style={{ fontSize: '0.85rem', fontFamily: 'monospace' }}>{selectedFile.name}</span>
                <div style={{ display: 'flex', gap: 4 }}>
                  {!editing ? (
                    <>
                      <button className="btn btn-secondary btn-xs" onClick={() => setEditing(true)}>Edit</button>
                      <button className="btn btn-secondary btn-xs" onClick={() => downloadFile(selectedFile)}>📥 Download</button>
                    </>
                  ) : (
                    <>
                      <button className="btn btn-primary btn-xs" onClick={saveFile} disabled={loading}>Save</button>
                      <button className="btn btn-secondary btn-xs" onClick={() => { setEditing(false); setEditedContent(fileContent); }}>Cancel</button>
                    </>
                  )}
                </div>
              </div>
              {editing ? (
                <textarea
                  style={{ width: '100%', height: 340, background: 'var(--bg-input)', color: 'var(--text-primary)', border: '1px solid var(--border)', borderRadius: 'var(--radius)', padding: 8, fontFamily: 'monospace', fontSize: '0.8rem', resize: 'vertical' }}
                  value={editedContent}
                  onChange={(e) => setEditedContent(e.target.value)}
                />
              ) : (
                <pre style={{ background: '#0a0a0a', color: '#ccc', padding: 12, borderRadius: 'var(--radius)', fontFamily: 'monospace', fontSize: '0.78rem', whiteSpace: 'pre-wrap', wordBreak: 'break-all', maxHeight: 340, overflowY: 'auto' }}>
                  {fileContent || '(empty file)'}
                </pre>
              )}
            </div>
          ) : (
            <div className="text-muted" style={{ padding: 20, textAlign: 'center' }}>
              {loading ? <span className="spinner" /> : 'Click a folder to expand • Double-click a file to view'}
            </div>
          )}
        </div>
      </div>

      <style>{`
        .tree-list { font-size: 0.82rem; user-select: none; }
        .tree-row {
          display: flex; align-items: center; padding: 3px 8px; cursor: pointer;
          border-radius: 3px; transition: background 0.1s;
        }
        .tree-row:hover { background: rgba(233, 69, 96, 0.1); }
        .tree-connector { color: var(--text-muted); font-family: monospace; font-size: 0.75rem; }
      `}</style>
    </div>
  );
}
