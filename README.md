# C2 Platform

参考 vshell 架构的命令与控制（C2）平台。Go + React 技术栈，适用于学习研究和红队演练。

## 架构

```
Browser (React SPA) ──── HTTP/WS ──── Teamserver (Go/Gin) ──── TCP (:4444) ──── Implant (Go)
                                           │
                                       SQLite DB
```

- **Teamserver**: Go + Gin 框架，SQLite 数据库，JWT 认证，REST API
- **Implant**: Go 编译的可执行文件，AES-256-GCM 加密通信，TCP 连接
- **Web UI**: React + TypeScript + Vite，暗色主题

## 快速开始

### 构建 Teamserver

```bash
make build-teamserver
# 或
go build -o bin/teamserver ./cmd/teamserver
```

### 启动 Teamserver

```bash
make dev
# 或
./bin/teamserver --config=config/config.yaml
```

首次运行会自动生成：
- JWT 签名密钥
- 主加密密钥
- 默认 admin 账户（密码打印到终端）

### 构建 Implant

```bash
# Linux
make build-implant-linux
# Windows
make build-implant-windows
```

构建时需要注入配置：
```bash
go build -ldflags="-s -w \
  -X c2/internal/implant.ServerHost=192.168.1.100 \
  -X c2/internal/implant.ServerPort=4444 \
  -X c2/internal/implant.MasterKeyHex=<32字节hex密钥>" \
  -o implant ./cmd/implant
```

### Web 前端

```bash
cd web && npm install && npm run dev
```

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/auth/login | 登录 |
| POST | /api/v1/auth/refresh | 刷新令牌 |
| POST | /api/v1/auth/logout | 登出 |
| GET | /api/v1/auth/me | 当前用户 |
| GET | /api/v1/dashboard/stats | 仪表盘统计 |
| GET | /api/v1/listeners | 列出监听器 |
| POST | /api/v1/listeners | 创建监听器 |
| DELETE | /api/v1/listeners/:id | 删除监听器 |
| POST | /api/v1/listeners/:id/start | 启动监听器 |
| POST | /api/v1/listeners/:id/stop | 停止监听器 |
| GET | /api/v1/agents | 列出代理 |
| GET | /api/v1/agents/:id | 获取代理详情 |
| GET | /api/v1/agents/:id/tasks | 获取代理任务 |
| POST | /api/v1/agents/:id/tasks | 创建任务 |
| GET | /api/v1/tasks/:id | 获取任务结果 |
| GET | /api/v1/audit | 审计日志 |
| POST | /api/v1/build | 构建植入体 |
| WS | /ws/agents/:id | 实时终端 |

## 通信协议

二进制信封格式：`[TotalLength:4][MessageType:4][AES-GCM密文][GCM Tag:16]`

消息类型：
- `0x01` HandshakeRequest — 植入体注册
- `0x02` HandshakeResponse — 服务器确认
- `0x03` Heartbeat — 心跳（fire-and-forget）
- `0x04` TaskRequest — 拉取任务
- `0x05` TaskResponse — 任务列表
- `0x06` TaskResult — 任务结果

加密：AES-256-GCM，每代理独立密钥（HKDF-SHA256 派生）。

## 项目结构

```
c2/
├── cmd/
│   ├── teamserver/main.go     # Teamserver 入口
│   └── implant/main.go        # Implant 入口
├── internal/
│   ├── teamserver/            # Teamserver 核心
│   │   ├── server.go          # 核心编排
│   │   ├── config.go          # 配置
│   │   ├── database/          # SQLite 层
│   │   ├── auth/              # JWT 认证
│   │   ├── listener/          # 监听器管理
│   │   ├── agent/             # 代理管理
│   │   ├── api/               # API + WebSocket
│   │   └── crypto/            # 加密
│   └── implant/               # 植入体实现
├── pkg/protocol/              # 通信协议共享库
├── web/                       # React 前端
├── config/config.yaml         # 配置模板
├── data/                      # 运行时数据
├── payloads/                  # 编译输出
└── Makefile
```

## 技术栈

- **语言**: Go 1.21+
- **Web 框架**: Gin
- **数据库**: SQLite (modernc.org/sqlite, 纯 Go)
- **认证**: JWT (golang-jwt/jwt/v5) + bcrypt
- **序列化**: Gob (二进制)
- **加密**: AES-256-GCM + HKDF
- **前端**: React 18 + TypeScript + Vite

## 设计决策

1. **纯 Go SQLite** (`modernc.org/sqlite`): 无 CGO，交叉编译无障碍
2. **植入体主动拉取任务**: NAT 穿透更友好
3. **心跳无响应**: 减少网络流量
4. **每代理独立密钥**: 单个被破解不影响全局
5. **Gob 序列化**: 二进制更小，双方都是 Go
6. **原生 database/sql**: MVP 阶段完全控制
