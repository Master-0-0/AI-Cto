# C2 Platform

参考 vshell 架构的命令与控制（C2）平台，Go + React 技术栈，适用于学习研究和红队演练。

## 架构

```
Browser (React SPA) ──── HTTP/WS ──── Teamserver (Go/Gin) ──── TCP (:4444) ──── Implant (Go)
                                           │
                                       SQLite DB
```

- **Teamserver**: Go + Gin 框架，SQLite 数据库，JWT 认证，REST + WebSocket API
- **Implant**: Go 编译的可执行文件，AES-256-GCM 加密通信，TCP 主动回连
- **Web UI**: React 18 + TypeScript + Vite，暗色主题，响应式布局

## 环境要求

| 组件 | 要求 |
|------|------|
| Go | 1.21+ |
| Node.js | 18+ (仅前端开发需要) |
| npm | 9+ (仅前端开发需要) |
| 操作系统 | Windows / Linux / macOS |

## 快速开始

### 1. 克隆项目

```bash
git clone <repo-url> c2
cd c2
```

### 2. 启动 Teamserver

**方式一：使用 Make**

```bash
make dev
```

**方式二：直接运行**

```bash
go run ./cmd/teamserver --config=config/config.yaml
```

**方式三：构建后运行**

```bash
make build-teamserver
./bin/teamserver --config=config/config.yaml
```

首次运行会自动执行初始化：
- 生成 JWT 签名密钥（写入 `config/config.yaml`）
- 生成 AES-256 主加密密钥（写入 `config/config.yaml`）
- 创建默认 `admin` 账户，密码打印到终端 **（请务必保存！）**
- 自动创建 `data/` 目录和 SQLite 数据库
- 若未配置 TLS 证书，自动生成自签名证书

### 3. 启动 Web 前端

```bash
cd web
npm install
npm run dev
```

前端开发服务器运行在 `http://localhost:5173`，API 请求自动代理到 Teamserver (`https://localhost:8443`)。

### 4. 访问平台

浏览器打开 `http://localhost:5173`，使用初始化时打印的 `admin` 账户登录。

> 生产部署时，先 `npm run build` 构建前端静态文件，Teamserver 会自动托管 `web/dist/` 目录。

---

## 配置说明

配置文件：`config/config.yaml`（参考模板 `config/config.example.yaml`）

```yaml
server:
  host: "0.0.0.0"       # 监听地址
  port: 8443             # HTTPS 端口

database:
  path: "./data/c2.db"   # SQLite 数据库路径

jwt:
  secret: ""             # JWT 签名密钥（留空自动生成）
  access_expiry: 15m     # Access Token 有效期
  refresh_expiry: 168h   # Refresh Token 有效期（7 天）

master_key: ""           # AES-256 主加密密钥（留空自动生成）

log_level: "info"        # 日志级别：debug / info / warn / error

tls:
  cert_file: ""          # TLS 证书文件（留空自动生成自签名证书）
  key_file: ""           # TLS 密钥文件
```

> ⚠️ **安全提醒**：`config/config.yaml` 包含敏感密钥，已被 `.gitignore` 排除，切勿提交到版本控制。

---

## 使用流程

### 第一步：创建监听器

登录后进入 **Listeners** 页面，点击「创建监听器」：

1. 填写名称（如 `primary-tcp`）
2. 协议选择 `TCP`
3. 填写监听地址（`0.0.0.0`）和端口（`4444`）
4. 点击「创建」，然后点击「启动」

此时 Teamserver 开始在指定端口监听 Implant 连接。

### 第二步：构建 Implant

进入 **Build** 页面：

1. 填写 Teamserver 的 IP 地址（Implant 回连的地址）
2. 填写端口（与监听器端口一致，如 `4444`）
3. 选择目标操作系统和架构
4. 点击「构建」

构建完成后，可从下载链接获取编译好的 Implant 可执行文件。

> 也可使用 Make 手动构建：
> ```bash
> # Windows
> make build-implant-windows
> # Linux
> make build-implant-linux
> ```

### 第三步：部署 Implant

将编译好的 Implant 上传到目标机器并执行：

```bash
# Windows
implant-windows-amd64.exe

# Linux
chmod +x implant-linux-amd64 && ./implant-linux-amd64
```

Implant 启动后会：
1. 主动连接 Teamserver 的 TCP 监听器
2. 发送握手包（含主机名、OS、IP 等系统信息）
3. 完成密钥协商
4. 进入心跳 + 任务轮询循环

### 第四步：管理 Agent

Implant 上线后，进入 **Agents** 页面：

- 查看所有已上线 Agent 列表（主机名、OS、最后心跳时间）
- 点击 Agent 进入详情页
- 在 **Terminal** 标签页中通过 WebSocket 实时交互
- 在 **Tasks** 标签页中下发后台任务
- 在 **Files** 标签页中浏览和管理远程文件

### 第五步：下发任务

在 Agent 详情页的 Tasks 标签页中：

1. 输入要执行的命令（如 `whoami`、`ipconfig`、`dir C:\`）
2. 点击「创建任务」
3. Agent 下次轮询时会拉取并执行任务
4. 执行结果自动回传显示

支持的任务类型：
- **Shell 命令**：在目标机器上执行任意命令
- **文件操作**：上传/下载/列出/删除文件
- **系统信息**：查询进程列表、网络连接等

---

## 开发指南

### 项目结构

```
c2/
├── cmd/
│   ├── teamserver/main.go     # Teamserver 入口
│   └── implant/main.go        # Implant 入口
├── internal/
│   ├── teamserver/            # Teamserver 核心
│   │   ├── server.go          # HTTP 路由、TLS、核心编排
│   │   ├── config.go          # YAML 配置加载/自动生成
│   │   ├── database/          # SQLite 数据库 + 模型 + 迁移
│   │   ├── auth/              # JWT 签发/验证、bcrypt、中间件、限流
│   │   ├── listener/          # 监听器生命周期 + TCP 实现
│   │   ├── agent/             # Agent 注册、心跳、任务调度、事件通知
│   │   ├── api/               # REST 处理器 + WebSocket + 构建器
│   │   └── crypto/            # AES-256-GCM 主密钥 + HKDF 派生
│   └── implant/               # Implant 实现
│       ├── implant.go         # 主循环（握手→心跳→轮询→执行）
│       ├── comm.go            # TCP 连接 + 指数退避重连
│       ├── executor.go        # 命令执行（Windows: cmd /c，Unix: /bin/sh）
│       ├── files.go           # 文件操作
│       ├── sysinfo.go         # 系统信息采集
│       └── hide_*.go          # 控制台窗口隐藏
├── pkg/protocol/              # 通信协议共享库
│   ├── constants.go           # 消息类型常量
│   ├── envelope.go            # 二进制信封编解码
│   ├── messages.go            # 协议消息体（Gob 序列化）
│   └── serializer.go          # Gob 封装
├── web/                       # React 前端
│   ├── src/
│   │   ├── components/        # UI 组件
│   │   ├── api/               # Axios API 客户端
│   │   ├── contexts/          # AuthContext
│   │   ├── hooks/             # useAuth, useWebSocket
│   │   └── types/             # TypeScript 类型定义
│   └── tests/                 # Playwright E2E 测试
├── config/
│   ├── config.yaml            # 运行配置（gitignore）
│   └── config.example.yaml    # 配置模板
├── data/                      # 运行时数据（数据库文件）
├── payloads/                  # 编译后的 Implant 输出
├── bin/                       # 编译后的 Teamserver 输出
└── Makefile
```

### 常用命令

```bash
# 构建
make build-teamserver        # 编译 Teamserver
make build-implant-windows   # 编译 Windows Implant
make build-implant-linux     # 编译 Linux Implant
make build-web               # 构建前端

# 运行
make dev                     # 启动 Teamserver (开发模式)

# 前端开发
cd web && npm run dev        # 启动前端开发服务器 (热更新)
cd web && npm run build      # 生产构建
cd web && npm run preview    # 预览生产构建

# 测试
cd web && npx playwright test    # 运行 E2E 测试

# 清理
make clean                   # 清除所有构建产物
```

### API 端点

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | `/health` | 无 | 健康检查 |
| POST | `/api/v1/auth/login` | 无 | 登录（限流） |
| POST | `/api/v1/auth/refresh` | 无 | 刷新 Access Token |
| POST | `/api/v1/auth/logout` | JWT | 登出 |
| GET | `/api/v1/auth/me` | JWT | 当前用户信息 |
| GET | `/api/v1/dashboard/stats` | JWT | 仪表盘统计 |
| GET | `/api/v1/listeners` | JWT | 列出监听器 |
| GET | `/api/v1/listeners/:id` | JWT | 获取监听器详情 |
| POST | `/api/v1/listeners` | Admin | 创建监听器 |
| DELETE | `/api/v1/listeners/:id` | Admin | 删除监听器 |
| POST | `/api/v1/listeners/:id/start` | Admin | 启动监听器 |
| POST | `/api/v1/listeners/:id/stop` | Admin | 停止监听器 |
| GET | `/api/v1/agents` | JWT | 列出 Agent |
| GET | `/api/v1/agents/:id` | JWT | 获取 Agent 详情 |
| DELETE | `/api/v1/agents/:id` | Admin | 删除 Agent |
| GET | `/api/v1/agents/:id/tasks` | JWT | 获取 Agent 任务列表 |
| POST | `/api/v1/agents/:id/tasks` | JWT | 创建任务 |
| GET | `/api/v1/tasks/:id` | JWT | 获取任务结果 |
| DELETE | `/api/v1/tasks/:id` | Admin | 删除任务 |
| GET | `/api/v1/audit` | JWT | 审计日志（分页） |
| POST | `/api/v1/build` | Admin | 按需编译 Implant |
| GET | `/api/v1/download/:filename` | Token | 下载编译产物 |
| WS | `/ws/agents/:id` | Token | 实时 Agent 终端 |

---

## 通信协议

Implant 与 Teamserver 之间使用自定义二进制协议，通过 TCP 传输：

```
[TotalLength:4 字节][MessageType:4 字节][AES-GCM 密文（Nonce + 加密数据 + Tag）]
```

### 消息类型

| 类型 | 值 | 说明 |
|------|-----|------|
| HandshakeRequest | `0x01` | Implant 注册（含系统信息） |
| HandshakeResponse | `0x02` | 服务器确认（含分配的 Agent ID） |
| Heartbeat | `0x03` | 心跳包（fire-and-forget，无响应） |
| TaskRequest | `0x04` | Agent 轮询新任务 |
| TaskResponse | `0x05` | 服务器下发任务列表 |
| TaskResult | `0x06` | Agent 回传任务执行结果 |

### 加密方案

- **算法**：AES-256-GCM（认证加密）
- **主密钥**：256 位随机生成，存储于 `config/config.yaml`
- **Agent 密钥**：通过 HKDF-SHA256 从主密钥 + Agent ID 派生，每个 Agent 独立
- **优点**：单个 Agent 密钥泄露不影响其他 Agent 的通信安全

---

## 技术栈

| 层级 | 技术 | 版本 |
|------|------|------|
| 后端语言 | Go | 1.21+ |
| HTTP 框架 | Gin | v1.12 |
| 数据库 | SQLite (modernc.org/sqlite) | v1.53 |
| 认证 | JWT (golang-jwt/jwt) + bcrypt | v5 |
| WebSocket | gorilla/websocket | v1.5 |
| 配置 | YAML (gopkg.in/yaml.v3) | v3 |
| 序列化 | Gob（二进制） | 标准库 |
| 加密 | AES-256-GCM + HKDF | 标准库 |
| 前端框架 | React | 18 |
| 前端语言 | TypeScript | 5 |
| 构建工具 | Vite | 5 |
| HTTP 客户端 | Axios | - |
| 路由 | React Router DOM | v6 |
| E2E 测试 | Playwright | - |

---

## 设计决策

1. **纯 Go SQLite** (`modernc.org/sqlite`): 无需 CGO，跨平台交叉编译无障碍
2. **Implant 主动拉取任务**: 对 NAT/防火墙环境更友好，无需服务端主动推送
3. **心跳无响应**: 减少网络流量，降低检测风险
4. **每 Agent 独立密钥**: HKDF 派生，单个被破解不影响全局
5. **Gob 序列化**: 比 JSON/Protobuf 更小，且通信双方均为 Go
6. **原生 database/sql**: MVP 阶段保持完全控制，避免 ORM 抽象泄漏

---

## 常见问题

### Q: 首次启动后找不到 admin 密码？

admin 密码仅在**首次启动**时打印到终端。如果丢失，删除 `data/c2.db` 数据库文件和 `config/config.yaml` 中的 `jwt.secret`、`master_key`，重新启动即可重新初始化。

### Q: 前端无法连接后端？

检查 `web/vite.config.ts` 中的代理配置，确保目标端口与 `config/config.yaml` 中的 `server.port` 一致。

### Q: TLS 证书警告？

开发环境使用自签名证书，浏览器会提示不安全。点击「高级」→「继续访问」即可。生产环境请配置有效的 TLS 证书。

### Q: Implant 无法上线？

1. 确认监听器已创建并启动
2. 确认防火墙放行监听器端口
3. 确认 Implant 编译时注入的 `ServerHost` 和 `ServerPort` 正确
4. 检查 Teamserver 日志中的错误信息

### Q: 如何修改监听端口？

编辑 `config/config.yaml` 中的 `server.port`（Web 端口）或通过 Web UI 创建监听器时指定不同的 TCP 端口。

---

## 安全声明

本工具仅限授权安全测试、红队演练和学习研究使用。使用者应遵守当地法律法规，获得被测系统的明确授权。作者不对任何未授权或非法使用承担责任。
