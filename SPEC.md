# C2 平台 MVP 实现计划

> 参考 vshell 架构 | Go + React | 学习研究项目

---

## 目录

1. [项目背景](#1-项目背景)
2. [架构概览](#2-架构概览)
3. [项目目录结构](#3-项目目录结构)
4. [数据库设计](#4-数据库设计)
5. [API 端点设计](#5-api-端点设计)
6. [通信协议设计](#6-通信协议设计)
7. [分阶段实施计划](#7-分阶段实施计划)
8. [关键设计决策](#8-关键设计决策)
9. [依赖项](#9-依赖项)
10. [验证计划](#10-验证计划)

---

## 1. 项目背景

从零开始构建一个参考 vshell 架构的 C2（Command & Control）平台。

- **技术栈**: Go 后端 + React 前端（Vite + TypeScript）
- **范围**: MVP 核心功能
- **MVP 目标**: Teamserver + Web管理界面 + TCP监听器 + 基础植入体(Windows/Linux) + 命令执行
- **参考架构**: vshell（集中式Teamserver, Go, Gin, SQLite, JWT, React）
- **定位**: 学习研究，注重代码质量、可维护性和清晰的架构设计

### vshell 架构参考要点

Vshell 是一个 Go 语言编写的 C2 框架，架构特点：

| 维度 | 设计 |
|------|------|
| 架构风格 | 单层，集中式 Teamserver |
| Web框架 | Gin（HTTP路由）+ JWT认证 |
| 数据库 | SQLite（嵌入式，无需外部依赖） |
| 植入体协议 | TCP/WS/DNS/DoH/KCP/OSS |
| 加密 | AES-256-GCM，静态嵌入密钥 |
| 前端 | React Web界面 |
| 部署 | 单个Go二进制文件，低复杂度 |

---

## 2. 架构概览

### 系统组件交互图

```
┌─────────────────────────────────────────────────────────────────────┐
│                         WEB BROWSER (React SPA)                     │
│   Login → Dashboard → Listeners → Agents → Terminal                │
└──────────┬──────────────────────────────────┬───────────────────────┘
           │  HTTP REST (/api/v1/*)           │  WebSocket (/ws/agents/*)
           ▼                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     GIN HTTP SERVER (:8443)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │ Auth (JWT)   │  │ API Handlers │  │  WS Hub      │              │
│  │ bcrypt+双令牌│  │ RESTful CRUD │  │  实时推送    │              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │
│         │                 │                  │                       │
│  ┌──────▼─────────────────▼──────────────────▼───────┐              │
│  │  Listener Manager  │  Agent Manager  │  Task Manager│            │
│  │  生命周期管理       │  注册表+心跳     │  队列/完成    │           │
│  └──────────┬─────────┴────────┬────────┴──────┬──────┘              │
│             │                  │               │                      │
│  ┌──────────▼──────────────────▼───────────────▼──────┐              │
│  │                   SQLite Database                  │              │
│  │  operators | listeners | agents | tasks | audit    │              │
│  └────────────────────────────────────────────────────┘              │
│             │                                                        │
│  ┌──────────▼──────────┐                                            │
│  │   TCP Listener      │  (:4444)                                   │
│  │   net.Listen accept │                                            │
│  └──────────┬──────────┘                                            │
└─────────────┼────────────────────────────────────────────────────────┘
              │  TCP (AES-256-GCM 加密)
              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    IMPLANT (目标机器)                                │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐              │
│  │  Comm    │ │  Crypto  │ │  SysInfo │ │ Executor │              │
│  │ TCP+重连 │ │AES-256   │ │ 系统信息 │ │ 命令执行 │              │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘              │
│                                                                     │
│  主循环: Register → Heartbeat(jitter) → Poll Tasks → Execute       │
│           → Send Results → Sleep → Repeat                          │
└─────────────────────────────────────────────────────────────────────┘
```

### 核心数据流

**代理注册流程**:
1. 植入体 TCP 连接服务器 → 2. 发送 HandshakeRequest（agent_id, hostname, os, arch, user, pid）→ 3. 服务器验证，派生独立密钥，插入/更新DB → 4. 返回 HandshakeResponse（heartbeat_interval, jitter）→ 5. 植入体进入主轮询循环

**命令执行流程**:
1. 操作员在Web终端输入命令 → 2. POST /api/v1/agents/:id/tasks 创建任务 → 3. 任务写入DB (status=pending) → 4. 植入体下次轮询时发送 TaskRequest → 5. 服务器从队列取出任务 → 6. 植入体解密并执行 → 7. 植入体返回 TaskResult → 8. 服务器更新DB，WebSocket推送结果到前端

**心跳/存活检测**:
- 植入体每 heartbeat_interval ± jitter% 秒发送 Heartbeat
- 服务器更新内存中的 last_seen，每30秒批量更新DB
- 后台 goroutine 每30秒检测：last_seen > 3×heartbeat_interval → 标记为 dead

---

## 3. 项目目录结构

```
c2/
├── cmd/
│   ├── teamserver/
│   │   └── main.go                          # 入口: 解析配置, 启动teamserver
│   └── implant/
│       └── main.go                          # 入口: 植入体启动桩
│
├── internal/
│   ├── teamserver/
│   │   ├── server.go                        # Server struct: New(), Run(), Shutdown()
│   │   ├── config.go                        # YAML配置加载, 默认值
│   │   │
│   │   ├── database/
│   │   │   ├── database.go                  # SQLite连接 (modernc.org/sqlite, 纯Go, 无CGO)
│   │   │   ├── migrations.go               # Schema迁移, 版本追踪
│   │   │   └── models.go                   # 数据模型结构体定义
│   │   │
│   │   ├── auth/
│   │   │   ├── jwt.go                       # JWT令牌生成/验证 (access 15min + refresh 7d)
│   │   │   ├── password.go                  # bcrypt密码哈希
│   │   │   ├── middleware.go                # Gin中间件: AuthRequired()
│   │   │   └── handler.go                  # 认证HTTP handlers: Login, Refresh, Logout, Me
│   │   │
│   │   ├── listener/
│   │   │   ├── interface.go                 # Listener接口: Start/Stop/ID/Status/Type
│   │   │   ├── manager.go                   # ListenerManager: Create/Start/Stop/Delete/List/Recover
│   │   │   ├── tcp.go                       # TCPListener: 实现Listener接口
│   │   │   └── handler.go                  # HTTP handlers: CRUD + start/stop
│   │   │
│   │   ├── agent/
│   │   │   ├── manager.go                   # AgentManager: 内存注册表 + DB持久化
│   │   │   ├── handler.go                   # 核心C2逻辑: 处理连接/握手/心跳/任务轮询
│   │   │   ├── notifier.go                  # 发布/订阅事件系统 (WebSocket广播)
│   │   │   └── task.go                      # TaskManager: Enqueue/Dequeue/Complete/List
│   │   │
│   │   ├── api/
│   │   │   ├── router.go                    # Gin路由设置, 所有中间件注入
│   │   │   ├── handlers.go                  # 通用API handlers (agents, tasks, dashboard, audit)
│   │   │   ├── websocket.go                 # WebSocket hub: 升级/读写泵
│   │   │   ├── builder.go                   # ImplantBuilder: go build + ldflags注入
│   │   │   └── handler_builder.go           # Build implant HTTP handler
│   │   │
│   │   └── crypto/
│   │       ├── session.go                   # CipherSession: per-agent AES-256-GCM加解密
│   │       └── keys.go                      # 主密钥生成/加载 + HKDF派生每代理密钥
│   │
│   └── implant/
│       ├── implant.go                       # 主循环: 连接→注册→心跳→任务循环
│       ├── config.go                        # 嵌入式配置 (ldflags注入: ServerHost, ServerPort, MasterKey)
│       ├── comm.go                          # TCP连接 + 帧收发 + 指数退避重连 (1s→2s→4s→...→5min max)
│       ├── crypto.go                        # AES-256-GCM 加解密
│       ├── sysinfo.go                       # 系统信息收集 (hostname, os, arch, user, pid, IPs)
│       └── executor.go                      # 命令执行 (Windows: cmd /c, Linux: /bin/sh -c)
│
├── pkg/
│   └── protocol/
│       ├── constants.go                     # 消息类型枚举, 协议版本, MaxMessageSize
│       ├── envelope.go                      # 二进制信封: EncodeEnvelope/DecodeEnvelope
│       ├── messages.go                      # 所有消息结构体 + Gob注册
│       └── serializer.go                    # Gob序列化/反序列化
│
├── web/                                     # React前端 (Vite + TypeScript)
│   ├── package.json
│   ├── vite.config.ts                       # Vite配置 + API代理
│   ├── tsconfig.json
│   ├── index.html
│   ├── public/
│   │   └── favicon.ico
│   └── src/
│       ├── main.tsx                         # ReactDOM.createRoot
│       ├── App.tsx                          # 路由配置 + AuthProvider
│       ├── vite-env.d.ts
│       │
│       ├── api/                             # Axios API客户端层
│       │   ├── client.ts                    # Axios实例 + JWT拦截器 (自动附加token + 401刷新)
│       │   ├── auth.ts                      # login(), refresh(), logout(), getMe()
│       │   ├── listeners.ts                 # 监听器CRUD API
│       │   ├── agents.ts                    # 代理API
│       │   ├── tasks.ts                     # 任务API
│       │   ├── dashboard.ts                 # 仪表盘统计API
│       │   └── builder.ts                   # 植入体构建API
│       │
│       ├── components/
│       │   ├── Layout.tsx                   # 应用外壳: 侧边栏 + 顶栏 + 内容区
│       │   ├── Sidebar.tsx                  # 导航链接: Dashboard, Listeners, Agents, Build, Audit
│       │   ├── Header.tsx                   # 当前用户 + 登出按钮
│       │   ├── LoginForm.tsx                # 登录表单 + 错误提示
│       │   ├── ProtectedRoute.tsx           # 未认证重定向到 /login
│       │   │
│       │   ├── Dashboard.tsx                # 统计卡片 + 最近代理列表
│       │   ├── StatCard.tsx                 # 可复用统计卡片
│       │   │
│       │   ├── ListenerList.tsx             # 监听器表格 + 状态徽标 + 操作按钮
│       │   ├── ListenerForm.tsx             # 创建/编辑监听器表单
│       │   │
│       │   ├── AgentList.tsx                # 代理表格 (主机名/OS/状态/最后心跳)
│       │   ├── AgentDetail.tsx              # 代理详情: 信息面板 + 任务列表 + 终端
│       │   ├── AgentTerminal.tsx            # 交互式命令终端 (暗色背景, 等宽字体)
│       │   ├── TaskList.tsx                 # 任务历史表格
│       │   ├── TaskForm.tsx                 # 命令输入表单
│       │   │
│       │   ├── BuildPanel.tsx               # 植入体构建: 选择OS/Arch + 构建 + 下载
│       │   │
│       │   └── AuditLog.tsx                 # 审计日志分页表格
│       │
│       ├── contexts/
│       │   └── AuthContext.tsx              # 认证Provider: user/login/logout/isAuthenticated
│       │
│       ├── hooks/
│       │   ├── useAuth.ts                   # 认证上下文快捷方式
│       │   └── useWebSocket.ts              # WebSocket连接管理 + 自动重连
│       │
│       ├── types/
│       │   └── index.ts                     # 所有TypeScript接口定义
│       │
│       └── styles/
│           └── globals.css                  # 暗色主题 + 终端样式 + 状态徽标颜色
│
├── config/
│   └── config.yaml                          # 默认配置模板
│
├── data/                                    # SQLite数据库运行时目录
│   └── .gitkeep
│
├── payloads/                                # 编译后植入体输出目录
│   └── .gitkeep
│
├── go.mod
├── go.sum
├── Makefile                                 # build-teamserver, build-implant-*, build-web, dev, clean
├── .gitignore
└── README.md
```

---

## 4. 数据库设计

使用 SQLite，所有时间戳使用 ISO 8601 文本格式存储。

### 表结构

```sql
-- 操作员表
CREATE TABLE operators (
    id              TEXT PRIMARY KEY,                          -- UUID v4
    username        TEXT UNIQUE NOT NULL,
    password_hash   TEXT NOT NULL,                             -- bcrypt hash
    role            TEXT NOT NULL DEFAULT 'operator',          -- 'admin' | 'operator'
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    last_login      TEXT,
    active          INTEGER NOT NULL DEFAULT 1                 -- boolean 0/1
);

-- 刷新令牌表
CREATE TABLE refresh_tokens (
    id              TEXT PRIMARY KEY,
    operator_id     TEXT NOT NULL,
    token_hash      TEXT UNIQUE NOT NULL,                      -- SHA-256 of refresh token
    expires_at      TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    revoked         INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (operator_id) REFERENCES operators(id) ON DELETE CASCADE
);

-- 监听器表
CREATE TABLE listeners (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    protocol        TEXT NOT NULL,                             -- 'tcp' (MVP), 后续扩展 'ws', 'dns'
    host            TEXT NOT NULL DEFAULT '0.0.0.0',
    port            INTEGER NOT NULL,
    status          TEXT NOT NULL DEFAULT 'stopped',           -- 'running' | 'stopped' | 'error'
    config_json     TEXT,                                      -- JSON blob for protocol-specific config
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    started_at      TEXT
);

-- 代理/植入体表
CREATE TABLE agents (
    id              TEXT PRIMARY KEY,                          -- UUID v4 (植入体生成并持久化)
    listener_id     TEXT,
    hostname        TEXT NOT NULL,
    os              TEXT NOT NULL,                             -- 'windows' | 'linux' | 'darwin'
    arch            TEXT NOT NULL,                             -- 'amd64' | '386' | 'arm64'
    username        TEXT,
    pid             INTEGER,
    local_ips       TEXT,                                      -- JSON array of IP strings
    version         TEXT NOT NULL DEFAULT '1.0.0',
    first_seen      TEXT NOT NULL DEFAULT (datetime('now')),
    last_seen       TEXT NOT NULL DEFAULT (datetime('now')),
    heartbeat_interval INTEGER NOT NULL DEFAULT 60,
    status          TEXT NOT NULL DEFAULT 'active',            -- 'active' | 'inactive' | 'dead'
    session_key     TEXT,                                      -- AES key encrypted with master key (hex)
    metadata_json   TEXT,
    FOREIGN KEY (listener_id) REFERENCES listeners(id) ON DELETE SET NULL
);

CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_last_seen ON agents(last_seen);

-- 任务表
CREATE TABLE tasks (
    id              TEXT PRIMARY KEY,
    agent_id        TEXT NOT NULL,
    operator_id     TEXT,
    command         TEXT NOT NULL,                             -- 'shell' (MVP) | 'sleep' | 'exit'
    arguments       TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',           -- 'pending' | 'sent' | 'completed' | 'error' | 'cancelled'
    output          TEXT,                                      -- base64-encoded output
    error_message   TEXT,
    timeout_sec     INTEGER NOT NULL DEFAULT 60,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    sent_at         TEXT,
    completed_at    TEXT,
    duration_ms     INTEGER,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    FOREIGN KEY (operator_id) REFERENCES operators(id) ON DELETE SET NULL
);

CREATE INDEX idx_tasks_agent_status ON tasks(agent_id, status);
CREATE INDEX idx_tasks_status ON tasks(status);

-- 审计日志表
CREATE TABLE audit_logs (
    id              TEXT PRIMARY KEY,
    operator_id     TEXT,
    action          TEXT NOT NULL,                             -- 'listener.create', 'agent.task', 'auth.login' 等
    resource_type   TEXT NOT NULL,                             -- 'listener' | 'agent' | 'task' | 'operator'
    resource_id     TEXT,
    details_json    TEXT,
    ip_address      TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (operator_id) REFERENCES operators(id) ON DELETE SET NULL
);

CREATE INDEX idx_audit_created ON audit_logs(created_at);
CREATE INDEX idx_audit_action ON audit_logs(action);

-- Schema版本追踪
CREATE TABLE schema_migrations (
    version         INTEGER PRIMARY KEY,
    applied_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
```

### 设计说明

- `session_key`: 每代理独立AES-256密钥(32 bytes)，hex编码后用主密钥加密存储
- 外键策略: `ON DELETE CASCADE` 用于强依赖关系（删除代理同时删除其任务），`ON DELETE SET NULL` 用于保留孤立记录
- 任务输出使用 base64 编码存储（支持二进制输出）
- 首次运行时自动创建默认 admin 操作员（随机生成密码并打印）

---

## 5. API 端点设计

所有API返回JSON。错误响应统一格式：`{"error": {"code": "ERROR_CODE", "message": "描述"}}`

### 端点列表

```
METHOD  PATH                          AUTH    描述
------  ----                          ----    ----
POST    /api/v1/auth/login            公开    登录 → {access_token, refresh_token, expires_in}
POST    /api/v1/auth/refresh          公开    刷新令牌 → {access_token, expires_in}
POST    /api/v1/auth/logout           鉴权    注销 (撤销refresh token)
GET     /api/v1/auth/me               鉴权    当前用户 → {id, username, role}

GET     /api/v1/dashboard/stats       鉴权    仪表盘统计 (agent/listener/task数量)

GET     /api/v1/listeners             鉴权    列出所有监听器
POST    /api/v1/listeners             鉴权    创建监听器 {name, protocol, host, port}
GET     /api/v1/listeners/:id         鉴权    获取单个监听器
DELETE  /api/v1/listeners/:id         鉴权    删除监听器 (需先停止)
POST    /api/v1/listeners/:id/start   鉴权    启动监听器
POST    /api/v1/listeners/:id/stop    鉴权    停止监听器

GET     /api/v1/agents                鉴权    列出代理 [?status=active]
GET     /api/v1/agents/:id            鉴权    获取代理完整详情
DELETE  /api/v1/agents/:id            鉴权    删除代理及其任务

GET     /api/v1/agents/:id/tasks      鉴权    列出代理的任务
POST    /api/v1/agents/:id/tasks      鉴权    创建任务 {command, arguments, timeout?}
GET     /api/v1/tasks/:id             鉴权    获取任务结果
DELETE  /api/v1/tasks/:id             鉴权    删除待处理任务

GET     /api/v1/audit                 鉴权    审计日志 [?page=1&limit=50]

POST    /api/v1/build                 鉴权    构建植入体 {os, arch} → {download_url, filename, checksum}

WS      /ws/agents/:id                鉴权*   实时终端 (token通过query参数传递)
```

### WebSocket 消息格式

```
Client → Server:  {type: "task", command: "shell", arguments: "whoami"}
Server → Client:  {type: "task_result", task_id: "...", output: "...", status: "completed"}
Server → Client:  {type: "agent_status", status: "active" | "inactive" | "dead"}
Server → Client:  {type: "error", message: "..."}
```

---

## 6. 通信协议设计

### 二进制信封格式

所有消息双向使用统一信封：

```
+--------+----------+-------------------+------------+
| Offset | Size     | Field             | Notes                    |
+--------+----------+-------------------+---------------------------|
| 0      | 4 bytes  | TotalLength       | Big-endian uint32,       |
|        |          |                   | 包含本字段 + Payload + Tag|
| 4      | 4 bytes  | MessageType       | Big-endian uint32        |
| 8      | variable | EncryptedPayload  | AES-256-GCM加密          |
| 8+N    | 16 bytes | GCMTag            | 认证标签                  |
+--------+----------+-------------------+---------------------------+
```

`TotalLength = 4 (length) + 4 (type) + len(EncryptedPayload) + 16 (tag)`

### 加密格式

加密前的 Payload（EncryptedPayload 解密后）:
```
+--------+----------+-------------------+
| Offset | Size     | Field             |
+--------+----------+-------------------+
| 0      | 12 bytes | Nonce (random)    |
| 12     | variable | Gob-encoded msg   |
+--------+----------+-------------------+
```

### 消息类型

| 常量 | 值 | 方向 | 结构体 |
|------|-----|------|--------|
| MsgHandshakeRequest | 0x01 | Implant→Server | HandshakeRequest{AgentID, Hostname, OS, Arch, Username, PID, LocalIPs, Version} |
| MsgHandshakeResponse | 0x02 | Server→Implant | HandshakeResponse{Accepted, AgentID, HeartbeatInterval, JitterPercent, ServerTime} |
| MsgHeartbeat | 0x03 | Implant→Server | Heartbeat{AgentID, Timestamp} (fire-and-forget, 无响应) |
| MsgTaskRequest | 0x04 | Implant→Server | TaskRequest{AgentID} |
| MsgTaskResponse | 0x05 | Server→Implant | TaskResponse{Tasks: []TaskDef{TaskID, Command, Arguments, Timeout}} |
| MsgTaskResult | 0x06 | Implant→Server | TaskResult{TaskID, AgentID, Status, Output, ErrorMsg, DurationMs} |
| MsgError | 0xFF | 双向 | ProtocolError{Code, Message} |

### 加密方案

- **算法**: AES-256-GCM（密钥 32 字节，Nonce 12 字节，Tag 16 字节）
- **主密钥**: 首次运行时随机生成 32 字节，hex 编码保存在 config.yaml
- **每代理密钥**: `HKDF-SHA256(masterKey, agentID, "c2-session-key", 32)`
- **植入体密钥**: 主密钥通过 ldflags 编译时嵌入
- **每条消息**: 随机生成新 Nonce

### 连接生命周期

```
植入体                                        服务器
  │                                              │
  ├── TCP连接 ──────────────────────────────►    │
  ├── HandshakeRequest ─────────────────────►    │
  │                                              ├── 验证 + 派生密钥
  │                                              ├── 插入/更新DB
  │    ◄────────────────────────── HandshakeResponse
  │                                              │
  │  === 主循环开始 ===                          │
  │                                              │
  │  Sleep(heartbeat_interval ± jitter%)         │
  ├── Heartbeat ────────────────────────────►    ├── 更新last_seen
  ├── TaskRequest ──────────────────────────►    ├── 从队列取任务
  │    ◄────────────────────────── TaskResponse  │
  ├── 执行任务...                                │
  ├── TaskResult ───────────────────────────►    ├── 更新DB + WS推送
  │                                              │
  │  === 循环 ===                                │
  │                                              │
  │  连接断开 → 指数退避重连 → 重新握手           │
```

---

## 7. 分阶段实施计划

### 阶段 1: 项目骨架和核心基础设施

**目标**: 可编译的 Go 项目，数据库连接正常，认证系统可用

1. 初始化 `go.mod` (module `c2`, Go 1.21+)
2. 创建完整目录树结构
3. 实现 `internal/teamserver/config.go` — YAML 配置加载 (`gopkg.in/yaml.v3`)，默认值填充
4. 实现 `internal/teamserver/database/database.go` — SQLite 连接，WAL模式，外键启用
5. 实现 `internal/teamserver/database/migrations.go` — Schema 迁移框架，版本追踪
6. 实现 `internal/teamserver/database/models.go` — 6 张表的 Go 结构体定义
7. 实现 `internal/teamserver/auth/password.go` — bcrypt 哈希/验证 (cost=12)
8. 实现 `internal/teamserver/auth/jwt.go` — access token (15min) + refresh token (7d, opaque)
9. 实现 `internal/teamserver/auth/middleware.go` — Gin JWT 验证中间件
10. 实现 `internal/teamserver/auth/handler.go` — Login/Refresh/Logout/Me
11. 实现 `internal/teamserver/server.go` — 核心编排: 组装所有组件
12. 实现 `cmd/teamserver/main.go` — 标志解析，启动服务器
13. 实现 `pkg/protocol/` — 消息结构体，常量定义
14. 创建 `config/config.yaml` 默认配置模板
15. 创建 `Makefile` (build-teamserver 目标) + `.gitignore`
16. 首次运行时自动创建默认 admin 操作员 (随机生成密码打印到stdout)

**交付物**: Teamserver 编译启动，监听 HTTPS，`/api/v1/auth/login` 可用，返回 JWT

---

### 阶段 2: 监听器系统

**目标**: TCP 监听器接受连接，植入体握手端到端可用

1. 实现 `internal/teamserver/listener/interface.go` — `Listener` 接口
2. 实现 `internal/teamserver/listener/manager.go` — 生命周期管理 + DB 持久化
3. 实现 `internal/teamserver/listener/tcp.go` — TCP 监听器 (net.Listen accept 循环)
4. 实现 `internal/teamserver/crypto/session.go` — `CipherSession` 每代理加解密
5. 实现 `internal/teamserver/crypto/keys.go` — 主密钥管理 + HKDF 派生
6. 实现 `pkg/protocol/envelope.go` — 二进制信封编解码
7. 实现 `pkg/protocol/serializer.go` — Gob 序列化
8. 实现 `internal/teamserver/agent/manager.go` — 内存代理注册表 (sync.RWMutex + map)
9. 实现 `internal/teamserver/agent/handler.go` — 连接处理: 握手验证 + 注册 + 心跳
10. 实现 `internal/teamserver/listener/handler.go` — 监听器 CRUD HTTP handlers

**交付物**: 可通过 API 创建/启动/停止 TCP 监听器，桩植入体可连接握手并发送心跳

---

### 阶段 3: 任务系统

**目标**: 操作员创建任务 → 代理拉取 → 执行 → 返回结果

1. 实现 `internal/teamserver/agent/task.go` — TaskManager: Enqueue/Dequeue/Complete/List/Delete
2. 完善 `agent/handler.go` 任务轮询循环: 接收 TaskRequest → Dequeue → 发送 TaskResponse → 接收 TaskResult → Complete
3. 实现 `internal/teamserver/api/router.go` — 完整路由配置
4. 实现 `internal/teamserver/api/handlers.go` — 代理/任务/仪表盘/审计 API

**交付物**: 完整任务生命周期打通

---

### 阶段 4: 审计日志

**目标**: 所有操作员行为可追踪

1. 实现审计日志工具函数 `LogAction(ctx, action, resourceType, resourceID, details)`
2. 集成到所有变更 API handlers
3. 实现审计日志 API 端点（分页查询）

---

### 阶段 5: Web 前端 — 基础

**目标**: React 应用可登录，有基础布局

1. Vite + React + TypeScript 脚手架
2. 安装依赖: `react-router-dom`, `axios`
3. 实现 `types/index.ts` — 所有 TS 接口
4. 实现 `api/client.ts` — Axios 实例 + JWT 拦截器 (自动附加 token, 401 时尝试 refresh)
5. 实现 `api/auth.ts` — 认证 API 封装
6. 实现 `contexts/AuthContext.tsx` — React Context 管理认证状态
7. 实现 `hooks/useAuth.ts`
8. 实现 `components/LoginForm.tsx`
9. 实现 `components/ProtectedRoute.tsx`
10. 实现 `components/Layout.tsx` + `Sidebar.tsx` + `Header.tsx`
11. 实现 `App.tsx` 路由配置
12. 配置 Vite proxy 转发 `/api` 和 `/ws` 到 teamserver

**交付物**: 完整登录流程可用，Dashboard 空白页面可见，侧边栏导航可用

---

### 阶段 6: Web 前端 — 功能页面

**目标**: 所有管理页面功能完整

1. 实现 `api/listeners.ts`, `api/agents.ts`, `api/tasks.ts`, `api/dashboard.ts`, `api/builder.ts`
2. 实现 `components/Dashboard.tsx` + `StatCard.tsx`
3. 实现 `components/ListenerList.tsx` + `ListenerForm.tsx`
4. 实现 `components/AgentList.tsx`
5. 实现 `components/AgentDetail.tsx` + `TaskList.tsx` + `TaskForm.tsx`
6. 实现 `components/BuildPanel.tsx`
7. 实现 `components/AuditLog.tsx`
8. 实现暗色主题 CSS

**交付物**: 完整 Web UI，可管理监听器/代理/任务，可构建植入体

---

### 阶段 7: WebSocket + 交互终端

**目标**: 实时命令终端

1. 实现 `internal/teamserver/api/websocket.go` — WS Hub + 升级 + 读写泵
2. 实现 `internal/teamserver/agent/notifier.go` — 发布/订阅事件
3. 集成 notifier 到 agent handler (任务完成/状态变化时发布事件)
4. 实现 `hooks/useWebSocket.ts` — React hook
5. 实现 `components/AgentTerminal.tsx` — 暗色终端 UI + 命令历史 + 自动滚动

**交付物**: 操作员可在 Agent 详情页输入命令，实时查看结果

---

### 阶段 8: 植入体

**目标**: 功能完整的 Windows/Linux 植入体

1. 实现 `internal/implant/config.go` — ldflags 配置注入 (ServerHost, ServerPort, MasterKeyHex)
2. 实现 `internal/implant/crypto.go` — AES-256-GCM 加解密
3. 实现 `internal/implant/sysinfo.go` — 系统信息收集
4. 实现 `internal/implant/comm.go` — TCP 连接 + 帧收发 + 指数退避重连
5. 实现 `internal/implant/executor.go` — 命令执行 (Windows: cmd /c, Linux: /bin/sh -c)
6. 实现 `internal/implant/implant.go` — 主循环
7. 实现 `internal/teamserver/api/builder.go` — Go 交叉编译 + ldflags 注入
8. 实现 `internal/teamserver/api/handler_builder.go` — Build HTTP handler
9. 添加 Makefile 目标: `build-implant-linux`, `build-implant-windows`

**交付物**: 端到端完整流程打通 — 创建监听器 → 构建植入体 → 运行 → Agent上线 → 发送命令 → 返回结果

---

### 阶段 9: 优化和加固

**目标**: 生产就绪的细节打磨

1. 优雅关闭: SIGINT/SIGTERM → 停止所有监听器 → 关闭所有代理连接 → 刷新DB → 退出
2. 状态恢复: 启动时恢复所有 status='running' 的监听器
3. 死代理检测: 后台 goroutine 每 30 秒检测，last_seen > 3×heartbeat_interval → 标记 dead
4. 结构化日志 + 可配置日志级别
5. CORS 配置
6. 前端细节: 加载状态 spinner, 错误 toast, 空状态提示
7. 前端响应式布局（最小可用）
8. 编写 `README.md`（项目说明、构建命令、架构概述）

---

## 8. 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| SQLite驱动 | `modernc.org/sqlite` | 纯Go实现，无CGO，植入体交叉编译无障碍 |
| 序列化 | Gob (非JSON) | 二进制更小，无需Schema协商，双方都是Go |
| 任务模型 | 植入体主动拉取 | NAT穿透更友好，简化重连逻辑 |
| 心跳 | Fire-and-forget | 减少流量，服务端超时看门狗检测存活 |
| 每代理密钥 | HKDF派生 | 单植入体被破解不影响其他 |
| WebSocket | 仅用于实时推送 | 任务创建走REST保证可靠，结果推送走WS |
| 数据库访问 | 原生 `database/sql` | MVP阶段完全控制，避免ORM复杂性 |
| TLS | 开发者自签证书 | 自动生成，生产环境替换 |
| 前端构建 | Vite + 纯CSS | 轻量，无需重量级UI框架 |

---

## 9. 依赖项

### Go 依赖

| 包 | 用途 |
|----|------|
| `github.com/gin-gonic/gin` | HTTP Web框架 |
| `gopkg.in/yaml.v3` | YAML配置解析 |
| `modernc.org/sqlite` | 纯Go SQLite驱动 |
| `github.com/golang-jwt/jwt/v5` | JWT令牌 |
| `golang.org/x/crypto` | bcrypt + HKDF |
| `github.com/gorilla/websocket` | WebSocket |
| `github.com/google/uuid` | UUID生成 |

### 前端依赖

| 包 | 用途 |
|----|------|
| `react` + `react-dom` | UI框架 |
| `react-router-dom` | 路由 |
| `axios` | HTTP客户端 |
| `vite` + `@vitejs/plugin-react` | 构建工具 |
| `typescript` | 类型检查 |

---

## 10. 验证计划

### 端到端测试流程

1. **启动 Teamserver**
   ```bash
   make dev
   # 验证: 终端打印 "Server listening on :8443", 首次运行打印admin密码
   ```

2. **Web 登录**
   - 浏览器打开 `https://localhost:8443`
   - 使用 admin 凭据登录
   - 验证: 跳转到 Dashboard

3. **创建 TCP 监听器**
   - Listeners → Create → name=TCP1, protocol=tcp, port=4444 → Start
   - 验证: 状态变为 "Running"

4. **构建植入体**
   - Build → OS=linux, Arch=amd64 → Build
   - 验证: 返回下载链接，文件存在

5. **运行植入体**
   - 复制到目标机器运行
   - 验证: Dashboard 显示新 Agent 上线

6. **命令执行**
   - 点击 Agent → 终端输入 `whoami` → Enter
   - 验证: 实时返回执行结果

7. **存活检测验证**
   - 关闭植入体进程 → 等待 ~3 分钟
   - 验证: Agent 状态变为 "dead"

### 单元测试

- `pkg/protocol/envelope_test.go` — 信封编解码往返测试
- `internal/teamserver/crypto/session_test.go` — 加解密正确性
- `internal/teamserver/auth/jwt_test.go` — 令牌生成/验证
- `internal/teamserver/auth/password_test.go` — 哈希/验证

### 交叉编译验证

```bash
make build-teamserver
GOOS=windows GOARCH=amd64 go build -o bin/teamserver.exe ./cmd/teamserver
make build-implant-linux
make build-implant-windows
```
