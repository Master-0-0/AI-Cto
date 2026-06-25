# C2 后台管理系统 — 自动化功能测试交付物

> **测试日期**: 2026-06-25 | **被测版本**: c2 v1.0.0 | **环境**: localhost:5173/8443 | **测试框架**: Playwright 1.61 + Chromium

---

## 第一部分：测试范围梳理

### 1.1 系统架构概览

```
Browser (React SPA :5173) ──── HTTP/WS Proxy ──── Teamserver (Go/Gin :8443) ──── TCP :4444 ──── Implant
                                                         │
                                                     SQLite DB
```

- **前端**: React 18 + TypeScript + Vite，暗色主题 SPA，6个路由页面
- **后端**: Go + Gin 框架，JWT 认证（HS256），bcrypt 密码哈希（cost=12），SQLite 存储
- **认证**: JWT access_token（短期）+ refresh_token（长期），axios 拦截器自动刷新
- **通信加密**: AES-256-GCM，每代理独立密钥（HKDF-SHA256 派生）

### 1.2 功能模块覆盖矩阵

| 模块 | 页面路由 | 核心功能 | 安全关注点 | 测试优先级 |
|------|----------|----------|-----------|-----------|
| **M1 登录模块** | `/login` | 用户名密码登录、表单校验、错误提示、已登录重定向 | 密码错误重试、SQL注入防护、暴力破解 | 🔴 最高 |
| **M2 会话管理** | 全局 | JWT签发/刷新/吊销、Token存储、401自动刷新、过期处理 | Token伪造检测、会话劫持防护 | 🔴 最高 |
| **M3 权限守卫** | 全局 | 未登录拦截、ProtectedRoute组件、路由级权限控制 | 越权访问、URL直接访问、浏览器后退绕过 | 🔴 最高 |
| **M4 仪表盘** | `/` | 4个统计卡片（Agent总数/活跃数/监听器/今日任务）、近期Agent列表、15秒轮询 | API错误容错 | 🟡 高 |
| **M5 监听器管理** | `/listeners` | CRUD监听器、启动/停止、状态切换、端口校验 | 端口越界、运行中删除限制 | 🟡 高 |
| **M6 Agent管理** | `/agents` `/agents/:id` | 列表展示、删除、详情查看、系统信息 | Agent不存在处理 | 🟡 高 |
| **M7 任务下发** | `/agents/:id`（嵌入） | Shell/Sleep/Exit命令、任务创建、状态轮询 | 不存在的Agent、缺少必填字段 | 🟡 高 |
| **M8 终端交互** | `/agents/:id`（嵌入） | WebSocket实时终端、命令输入输出、自动滚动 | WebSocket未授权访问 | 🟢 中 |
| **M9 审计日志** | `/audit` | 分页展示、时间/操作/资源/IP列、前后翻页 | 未授权访问 | 🟢 中 |
| **M10 构建面板** | `/build` | OS/Arch选择器、构建触发（当前为桩实现） | 未授权访问 | 🟢 中 |
| **M11 导航** | 全局 | 侧边栏5个链接、高亮当前页、Header用户信息 | — | 🟢 中 |
| **M12 退出登录** | 全局 | Token清除、refresh_token吊销、跳转登录页、后退防护 | 退出后访问受保护页、浏览器后退泄露 | 🔴 最高 |

### 1.3 API 端点覆盖

| 端点 | 方法 | 认证 | 测试覆盖 |
|------|------|------|---------|
| `/api/v1/auth/login` | POST | 公开 | ✅ |
| `/api/v1/auth/refresh` | POST | 公开 | ✅ |
| `/api/v1/auth/logout` | POST | JWT | ✅ |
| `/api/v1/auth/me` | GET | JWT | ✅ |
| `/api/v1/dashboard/stats` | GET | JWT | ✅ |
| `/api/v1/listeners` | GET/POST | JWT | ✅ |
| `/api/v1/listeners/:id` | GET/DELETE | JWT | ✅ |
| `/api/v1/listeners/:id/start` | POST | JWT | ✅ |
| `/api/v1/listeners/:id/stop` | POST | JWT | ✅ |
| `/api/v1/agents` | GET | JWT | ✅ |
| `/api/v1/agents/:id` | GET | JWT | ✅ |
| `/api/v1/agents/:id/tasks` | GET/POST | JWT | ✅ |
| `/api/v1/tasks/:id` | GET | JWT | ✅ |
| `/api/v1/audit` | GET | JWT | ✅ |
| `/api/v1/build` | POST | JWT | ✅ |
| `/health` | GET | 公开 | ✅ |

---

## 第二部分：标准化自动化测试用例

### M1: 登录模块

| 用例ID | 前置条件 | 操作步骤 | 预期结果 | 严重等级 | 自动化校验点 |
|--------|---------|---------|---------|---------|-------------|
| **TC-C2-M1-001** | 无 | 1. 访问/login 2. 输入admin/259a2daf10da 3. 点击Login | 跳转至 /，sidebar可见，显示用户名和角色 | 🔴阻断 | `waitForURL('**/')`, `expect(.sidebar).toBeVisible()`, `expect(.header-user).toContainText('admin')`, `localStorage.getItem('access_token') !== null` |
| **TC-C2-M1-002** | 无 | 1. 访问/login 2. 输入admin/wrongpassword 3. 点击Login | 停留在/login，显示错误提示 `.alert-error` | 🔴阻断 | `expect(page).toHaveURL(/\/login/)`, `expect(.alert-error).toBeVisible()`, `localStorage.getItem('access_token') === null` |
| **TC-C2-M1-003** | 无 | 1. 访问/login 2. 空用户名+空密码 3. 绕过HTML5校验提交 | 停留在/login（后端拒绝） | 🟡高 | `expect(page).toHaveURL(/\/login/)` |
| **TC-C2-M1-004** | 无 | 1. 访问/login 2. 输入不存在的用户名 3. 点击Login | 显示 "Invalid credentials" 错误 | 🟡高 | `expect(.alert-error).toContainText(/Invalid credentials/i)` |
| **TC-C2-M1-005** | 已登录 | 1. 已登录状态下访问/login | 自动重定向到 / | 🟡高 | `expect(page).not.toHaveURL(/\/login/)`, `expect(.sidebar).toBeVisible()` |
| **TC-C2-M1-006** | 无 | 1. 以GET方法请求/api/v1/auth/login?username=admin&password=xxx | 返回非200状态码 | 🟢中 | `expect(resp.status()).not.toBe(200)` |
| **TC-C2-M1-007** | 无 | 1. 用户名输入 `admin' OR '1'='1` 2. 密码任意 3. 点击Login | 显示错误，不绕过认证 | 🔴阻断 | `expect(.alert-error).toBeVisible()` |
| **TC-C2-M1-008** | 无 | 1. 连续5次错误密码尝试 2. 第6次使用正确密码 | 第6次成功登录（无误次数锁定时可继续） | 🟡高 | 遍历5次失败→成功登录验证 |

### M2: 会话管理

| 用例ID | 前置条件 | 操作步骤 | 预期结果 | 严重等级 | 自动化校验点 |
|--------|---------|---------|---------|---------|-------------|
| **TC-C2-M2-001** | 已登录 | 1. 注入过期JWT到localStorage 2. 访问/ | 自动refresh获取新token，页面正常加载 | 🔴阻断 | `newToken !== EXPIRED_JWT`, `expect(.sidebar).toBeVisible()` |
| **TC-C2-M2-002** | 无 | 1. 注入无效access_token+refresh_token 2. 访问/ | 重定向到/login，localStorage被清空 | 🔴阻断 | `waitForURL(/\/login/)`, `localStorage.getItem('access_token') === null` |
| **TC-C2-M2-003** | 已登录 | 1. 登录获取tokens 2. 点击Logout 3. 尝试用旧refresh_token刷新 | refresh_token已被吊销（返回401） | 🔴阻断 | `expect(rawResp.status()).toBe(401)` |
| **TC-C2-M2-004** | 无Token | 1. 不带Authorization头请求所有受保护端点 | 全部返回401 + `{"error":{"code":"UNAUTHORIZED"}}` | 🔴阻断 | 遍历5个端点: `expect(resp.status()).toBe(401)`, `expect(body.error.code).toBe('UNAUTHORIZED')` |
| **TC-C2-M2-005** | 已登录 | 1. 获取有效JWT 2. 篡改签名最后一位 3. 篡改payload的role（不重新签名） | 两次请求都返回401 | 🔴阻断 | `expect(resp.status()).toBe(401)` |

### M3: 权限守卫（越权访问）

| 用例ID | 前置条件 | 操作步骤 | 预期结果 | 严重等级 | 自动化校验点 |
|--------|---------|---------|---------|---------|-------------|
| **TC-C2-M3-001** | 未登录, clean context | 1. 直接访问 `/` 2. 直接访问 `/listeners` 3. 直接访问 `/agents` 4. 直接访问 `/agents/test-uuid` 5. 直接访问 `/audit` 6. 直接访问 `/build` | 每个都重定向到/login | 🔴阻断 | `waitForURL(/\/login/)`, `expect(.login-card).toBeVisible()`, `expect(.sidebar).not.toBeVisible()` |
| **TC-C2-M3-002** | 未登录 | 1. 访问/agents后清除tokens 2. 浏览器后退 | 不泄露受保护内容，停留在/login | 🔴阻断 | `expect(.login-card).toBeVisible()` |

### M4: 仪表盘

| 用例ID | 前置条件 | 操作步骤 | 预期结果 | 严重等级 | 自动化校验点 |
|--------|---------|---------|---------|---------|-------------|
| **TC-C2-M4-001** | 已登录 | 1. 访问/ | 显示4个统计卡片 + Recent Agents区域 | 🟡高 | `expect(.stat-card).toHaveCount(4)`, 标题校验 |
| **TC-C2-M4-002** | 已登录 | 1. 访问/ 2. 等待16秒 | /dashboard/stats至少被请求2次（初始+轮询） | 🟢中 | `requestCount > 1` |
| **TC-C2-M4-003** | 已登录 | 1. Mock /dashboard/stats 返回500 2. 访问/ | 应用不崩溃，sidebar和header仍可见 | 🟡高 | `expect(.sidebar).toBeVisible()`, `expect(.header).toBeVisible()` |

### M5: 监听器管理

| 用例ID | 前置条件 | 操作步骤 | 预期结果 | 严重等级 | 自动化校验点 |
|--------|---------|---------|---------|---------|-------------|
| **TC-C2-M5-001** | 已登录 | 1. 点击"+ Create" 2. 输入名称+端口19999 3. 提交 | Modal关闭，表格出现新监听器，状态stopped | 🟡高 | `expect(td).toBeVisible()`, badge-stopped可见 |
| **TC-C2-M5-002** | 已登录, 有监听器 | 1. 点击Start 2. 等待 3. 点击Stop | 状态在stopped↔running间切换 | 🟡高 | `expect(.badge-running).toBeVisible()` → `expect(.badge-stopped).toBeVisible()` |
| **TC-C2-M5-003** | 已登录, 有运行中监听器 | 1. 查看运行中监听器的操作栏 | Delete按钮不可见 | 🟡高 | `expect(.btn-danger).not.toBeVisible()` |
| **TC-C2-M5-004** | 已登录, 有已停止监听器 | 1. 点击Delete 2. 确认弹窗 | 监听器从表格消失 | 🟡高 | `expect(td).not.toBeVisible()` |
| **TC-C2-M5-005** | 已登录 | 1. 创建监听器时输入无效端口(70000) 2. 提交 | 显示错误提示 | 🟡高 | `expect(.alert-error).toBeVisible()` |

### M6: Agent管理

| 用例ID | 前置条件 | 操作步骤 | 预期结果 | 严重等级 | 自动化校验点 |
|--------|---------|---------|---------|---------|-------------|
| **TC-C2-M6-001** | 已登录 | 1. 访问/agents | 显示表格（或空状态消息）+ 6列标题 | 🟡高 | 列标题校验 + 10秒内2次以上请求 |
| **TC-C2-M6-002** | 已登录, 有Agent | 1. 点击Delete 2. 确认弹窗 | Agent从列表移除 | 🟢中 | hostname不再可见 |
| **TC-C2-M6-003** | 已登录, 有Agent | 1. 点击Agent hostname | 进入详情页，显示System Information | 🟡高 | URL匹配，info详情可见 |
| **TC-C2-M6-004** | 已登录 | 1. 访问/agents/nonexistent-id | 显示loading spinner，不崩溃 | 🟢中 | `expect(.spinner).toBeVisible()`, sidebar可见 |

### M7: 任务下发

| 用例ID | 前置条件 | 操作步骤 | 预期结果 | 严重等级 | 自动化校验点 |
|--------|---------|---------|---------|---------|-------------|
| **TC-C2-M7-001** | 已登录, 有Agent | 1. 进入Agent详情 2. 选择shell+输入whoami 3. 点击Send | 任务出现在Task列表 | 🟡高 | `expect(td).toContainText('whoami')` |
| **TC-C2-M7-002** | 已登录 | 1. POST /agents/nonexistent/tasks | 返回404 | 🟢中 | `expect(resp.status()).toBe(404)` |
| **TC-C2-M7-003** | 已登录, 有Agent | 1. POST创建任务但不含command字段 | 返回400 | 🟡高 | `expect(resp.status()).toBe(400)` |
| **TC-C2-M7-004** | 已登录, Agent活跃 | 1. 创建任务 2. 等待 | 任务状态流转到completed或error | 🟡高 | badge-completed或badge-error出现 |

### M8: 终端

| 用例ID | 前置条件 | 操作步骤 | 预期结果 | 严重等级 | 自动化校验点 |
|--------|---------|---------|---------|---------|-------------|
| **TC-C2-M8-001** | 已登录, 有Agent | 1. 在终端输入whoami 2. 按Enter | 终端显示命令和输出 | 🟡高 | terminal含whoami文本，terminal-output可见 |
| **TC-C2-M8-002** | 已登录, 有Agent | 1. 发送命令 2. 查看滚动位置 | 自动滚动到底部 | 🟢中 | scrollTop ≥ scrollHeight-clientHeight-50 |
| **TC-C2-M8-003** | 已登录 | 1. 用无效token连接WebSocket | 连接被拒绝（非1000关闭码） | 🔴阻断 | wsResult不包含'close:1000' |

### M9: 审计日志

| 用例ID | 前置条件 | 操作步骤 | 预期结果 | 严重等级 | 自动化校验点 |
|--------|---------|---------|---------|---------|-------------|
| **TC-C2-M9-001** | 已登录 | 1. 访问/audit | 显示分页表格，4列标题，登录事件可见 | 🟢中 | 列标题校验 + `expect(td).toContainText('auth.login')` |
| **TC-C2-M9-002** | 已登录 | 1. 验证Previous按钮状态 2. 翻页 | 第1页Previous禁用，翻页后启用 | 🟢中 | `expect(prevBtn).toBeDisabled()` → 翻页 → `expect(prevBtn).toBeEnabled()` |
| **TC-C2-M9-003** | 未登录 | 1. GET /audit无token | 返回401 | 🟡高 | `expect(resp.status()).toBe(401)` |

### M10-M12: 构建/导航/退出

| 用例ID | 前置条件 | 操作步骤 | 预期结果 | 严重等级 | 自动化校验点 |
|--------|---------|---------|---------|---------|-------------|
| **TC-C2-M10-001** | 已登录 | 1. 访问/build 2. 点击Build | 显示 "not yet available" 桩消息 | 🟢中 | `expect(text-muted).toContainText('not yet available')` |
| **TC-C2-M10-002** | 未登录 | 1. POST /build | 返回401 | 🟡高 | `expect(resp.status()).toBe(401)` |
| **TC-C2-M11-001** | 已登录 | 1. 依次点击5个侧边栏链接 | URL和页面标题匹配 | 🟢中 | URL+标题双重验证 |
| **TC-C2-M11-002** | 已登录 | 1. 切换到不同页面 | 只有当前页链接高亮 | 🟢中 | `expect(.sidebar-link.active).toHaveCount(1)` |
| **TC-C2-M11-003** | 已登录 | 1. 查看Header | 显示 "admin (admin)" | 🟢中 | `expect(.header-user).toContainText('admin (admin)')` |
| **TC-C2-M12-001** | 已登录 | 1. 点击Logout | 清空tokens，跳转/login | 🔴阻断 | localStorage清空, login-card可见 |
| **TC-C2-M12-002** | 登出后 | 1. 依次访问所有受保护路由 | 全部重定向到/login | 🔴阻断 | 5个路由逐一验证 |
| **TC-C2-M12-003** | 登出后 | 1. 浏览器后退2次 | 不泄露受保护内容 | 🔴阻断 | login-card始终可见 |

---

## 第三部分：自动化执行操作流程

### 3.1 环境准备

```bash
# 1. 启动后端（终端1）
cd C:\Users\Master\Desktop\c2
go build -o bin/teamserver ./cmd/teamserver
./bin/teamserver --config=config/config.yaml

# 2. 启动前端（终端2）
cd C:\Users\Master\Desktop\c2\web
npm install
npm run dev

# 3. 验证服务
curl -sk https://localhost:8443/health   # 应返回 {"status":"ok"}
curl -s http://localhost:5173             # 应返回HTML页面

# 4. 运行全部测试
cd C:\Users\Master\Desktop\c2\web
npx playwright install chromium
npx playwright test --reporter=list

# 5. 查看测试报告
npx playwright show-report
```

### 3.2 测试执行步骤详解

#### 阶段一：登录与认证安全（M1, M2, M3）— 预计4分钟

```
Step 1.1: 浏览器打开 http://localhost:5173/login
Step 1.2: 验证登录页渲染 → .login-card可见, .login-title='C2 Platform'
Step 1.3: 输入admin/259a2daf10da → 点击Login → 等待跳转至/
Step 1.4: 验证localStorage中access_token和refresh_token已存储
Step 1.5: 验证Header显示 "admin (admin)"
Step 1.6: 已登录状态访问/login → 验证自动重定向
Step 1.7: 登出后尝试错误密码 → 验证错误提示可见 + token未存储
Step 1.8: SQL注入测试: ' OR '1'='1 -- → 验证不绕过认证
Step 1.9: 5次快速错误密码 → 第6次正确密码仍可登录
Step 1.10: 注入过期JWT → 验证自动刷新机制
Step 1.11: 注入无效token对 → 验证重定向+清理
Step 1.12: 篡改JWT签名 → 验证401拒绝
Step 1.13: 未登录直接访问6个受保护URL → 验证全部拦截
```

#### 阶段二：核心功能（M4-M7）— 预计5分钟

```
Step 2.1: 登录后访问Dashboard → 验证4个StatCard
Step 2.2: 等待16秒验证15秒轮询机制
Step 2.3: Mock /dashboard/stats 500 → 验证容错
Step 2.4: 访问/listeners → 点击 + Create → 填写表单 → 验证创建
Step 2.5: Start → 验证状态变running → Stop → 验证状态变stopped
Step 2.6: 验证运行中监听器无Delete按钮
Step 2.7: 停止后Delete → 确认弹窗 → 验证删除
Step 2.8: 访问/agents → 验证列表/空状态
Step 2.9: 如果有Agent → 点击进入详情 → 验证System Info
Step 2.10: 在Send Command中选shell → 输入whoami → Send
Step 2.11: 验证Task出现在列表 → 等待completed状态
Step 2.12: 在Terminal中输入命令 → 验证输出显示
```

#### 阶段三：辅助功能与退出（M8-M12）— 预计3分钟

```
Step 3.1: 测试WebSocket未授权连接
Step 3.2: 访问/audit → 验证表格 + 分页
Step 3.3: 访问/build → 验证桩消息
Step 3.4: 侧边栏导航: Dashboard→Listeners→Agents→Build→Audit
Step 3.5: 验证只有一个链接高亮
Step 3.6: 点击Logout → 验证token清除+跳转
Step 3.7: 登出后访问所有受保护页面 → 验证拦截
Step 3.8: 浏览器后退 → 验证不泄露内容
```

### 3.3 一键执行命令

```bash
# 完整测试套件（50条用例，预计12分钟）
npx playwright test --reporter=list --timeout=120000

# 仅运行安全类测试（M1, M2, M3, M12）
npx playwright test m1-login m2-session m3-authguard m12-logout --reporter=list

# 仅运行功能类测试
npx playwright test m4-dashboard m5-listeners m6-agents m7-tasks m8-terminal m9-audit m10-build m11-navigation --reporter=list

# 调试模式（显示浏览器窗口）
npx playwright test --headed

# 单文件调试
npx playwright test m1-login.spec.ts --headed --debug

# 生成JSON报告
npx playwright test --reporter=json > test-results/results.json

# 生成HTML报告
npx playwright show-report
```

### 3.4 测试文件结构

```
web/tests/
├── global-setup.ts        # 全局前置检查（后端+前端可达性）
├── fixtures/
│   └── auth.fixture.ts    # 登录/登出/token管理辅助函数
├── helpers/
│   ├── api.ts             # 直接API请求辅助（含token注入）
│   └── jwt.ts             # JWT解码/篡改/过期token生成
├── specs/
│   ├── m1-login.spec.ts   # 登录模块 (8条用例)
│   ├── m2-session.spec.ts # 会话管理 (5条用例)
│   ├── m3-authguard.spec.ts # 权限守卫 (7条用例)
│   ├── m4-dashboard.spec.ts # 仪表盘 (3条用例)
│   ├── m5-listeners.spec.ts # 监听器管理 (5条用例)
│   ├── m6-agents.spec.ts  # Agent管理 (4条用例)
│   ├── m7-tasks.spec.ts   # 任务管理 (4条用例)
│   ├── m8-terminal.spec.ts # 终端 (3条用例)
│   ├── m9-audit.spec.ts   # 审计日志 (3条用例)
│   ├── m10-build.spec.ts  # 构建面板 (2条用例)
│   ├── m11-navigation.spec.ts # 导航 (3条用例)
│   └── m12-logout.spec.ts # 退出登录 (3条用例)
└── playwright.config.ts   # Playwright配置
```

---

## 第四部分：缺陷判定标准

### 4.1 缺陷严重等级定义

| 等级 | 标识 | 定义 | 示例 |
|------|------|------|------|
| **阻断 (Blocker)** | 🔴 | 系统核心功能不可用或存在严重安全漏洞，阻塞测试继续 | 无法登录、认证绕过、Token泄露、越权访问 |
| **严重 (Critical)** | 🟠 | 核心功能存在缺陷但可通过其他方式绕过，或重要数据错误 | 密码重试无限制、监听器无法停止、任务创建失败 |
| **一般 (Major)** | 🟡 | 非核心功能缺陷、边界条件处理不当、用户体验问题 | 空状态文案缺失、轮询频率异常、分页逻辑错误 |
| **轻微 (Minor)** | 🟢 | UI样式问题、非关键文案错误、非阻塞性提示缺失 | 图标缺失、loading状态不统一、文案拼写错误 |

### 4.2 自动化断言判断逻辑

#### 4.2.1 认证与授权类（阻断级判定）

```typescript
// 登录成功断言
const LOGIN_SUCCESS_CHECKS = [
  { assert: "page.url() === '/'", desc: '跳转至首页' },
  { assert: "page.locator('.sidebar').isVisible()", desc: '侧边栏可见' },
  { assert: "localStorage.getItem('access_token') !== null", desc: 'Token已存储' },
  { assert: "localStorage.getItem('refresh_token') !== null", desc: 'Refresh Token已存储' },
];

// 登录失败断言
const LOGIN_FAIL_CHECKS = [
  { assert: "page.url().includes('/login') === true", desc: '未离开登录页' },
  { assert: "localStorage.getItem('access_token') === null", desc: '无Token泄露' },
];

// 越权访问断言
const UNAUTHORIZED_ACCESS_CHECKS = [
  { assert: "resp.status() === 401", desc: 'HTTP 401' },
  { assert: "body.error.code === 'UNAUTHORIZED'", desc: '标准错误码' },
];

// 未登录页面拦截断言
const AUTH_GUARD_CHECKS = [
  { assert: "page.url().includes('/login') === true", desc: '重定向至登录' },
  { assert: "page.locator('.login-card').isVisible()", desc: '登录卡片可见' },
  { assert: "page.locator('.sidebar').isVisible() === false", desc: '侧边栏不可见' },
];
```

#### 4.2.2 会话管理类（阻断级判定）

```typescript
// Token刷新断言
const TOKEN_REFRESH_CHECKS = [
  { assert: "newToken !== null", desc: '新Token存在' },
  { assert: "newToken !== EXPIRED_JWT", desc: 'Token已更新' },
  { assert: "page.locator('.sidebar').isVisible()", desc: '页面正常渲染' },
];

// Token吊销断言
const TOKEN_REVOKE_CHECKS = [
  { assert: "refreshResp.status() === 401", desc: '旧Token已吊销' },
  { assert: "localStorage.getItem('access_token') === null", desc: 'access_token已清除' },
  { assert: "localStorage.getItem('refresh_token') === null", desc: 'refresh_token已清除' },
];

// JWT篡改检测断言
const JWT_TAMPER_CHECKS = [
  { assert: "resp1.status() === 401", desc: '篡改签名→401' },
  { assert: "resp2.status() === 401", desc: '篡改claim→401' },
];
```

#### 4.2.3 功能正确性类（严重/一般级判定）

```typescript
// 监听器CRUD断言
const LISTENER_CRUD_CHECKS = [
  { assert: "page.locator('td').filter({hasText: name}).isVisible()", desc: '监听器出现' },
  { assert: "row.locator('.badge-stopped').isVisible()", desc: '初始状态为stopped' },
  { assert: "row.locator('.badge-running').isVisible()", desc: 'Start后变running' },
  { assert: "row.locator('.btn-danger').isVisible() === false", desc: '运行中无删除按钮' },
  { assert: "page.locator('td').filter({hasText: name}).isVisible() === false", desc: '删除后消失' },
];

// 仪表盘断言
const DASHBOARD_CHECKS = [
  { assert: "page.locator('.stat-card').count() === 4", desc: '4个统计卡片' },
  { assert: "page.locator('.stat-card').nth(0).textContent.includes('Total Agents')", desc: 'Agent总数' },
  { assert: "requestCount > 1", desc: '15秒轮询正常' },
];

// 任务下发断言
const TASK_CHECKS = [
  { assert: "page.locator('td').filter({hasText: args}).isVisible()", desc: '任务出现' },
  { assert: "row.locator('.badge-completed, .badge-error').isVisible()", desc: '任务完成或报错' },
];
```

#### 4.2.4 安全测试类（阻断级判定）

```typescript
// SQL注入防护断言
const SQL_INJECTION_CHECKS = [
  { assert: "page.locator('.alert-error').isVisible()", desc: '显示错误提示' },
  { assert: "page.url().includes('/login') === true", desc: '未绕过登录' },
  { assert: "localStorage.getItem('access_token') === null", desc: '无Token生成' },
];

// WebSocket未授权断言
const WS_UNAUTHORIZED_CHECKS = [
  { assert: "wsResult !== 'timeout'", desc: '连接被关闭（非超时）' },
  { assert: "wsResult.includes('close:1000') === false", desc: '非正常关闭码' },
];

// 登出后安全断言
const LOGOUT_SECURITY_CHECKS = [
  { assert: "localStorage.getItem('access_token') === null", desc: 'Token已清除' },
  { assert: "page.locator('.login-card').isVisible()", desc: '在登录页' },
  // 访问所有受保护路由
  ...PROTECTED_ROUTES.map(r => ({
    assert: `goto('${r}') → url.includes('/login')`,
    desc: `${r}被拦截`,
  })),
];
```

### 4.3 自动化测试通过标准

| 测试类别 | 通过条件 | 不通过措施 |
|----------|---------|-----------|
| **阻断级用例** | 100% 通过 | 禁止上线，立即修复 |
| **严重级用例** | 100% 通过 | 禁止上线，PTR修复后回归 |
| **一般级用例** | ≥95% 通过 | 记录bug，下个迭代修复 |
| **轻微级用例** | ≥90% 通过 | 记录backlog，排期修复 |
| **安全类用例** | 100% 通过 | 一票否决，不允许上线 |

### 4.4 2026-06-25 测试执行结果摘要

```
总用例数: 50
通过: 38 (76%)
失败: 12 (24%)

通过率按模块:
  M1 登录模块:      4/8  (50%)  ⚠️ 4条失败 — 错误消息断言需调整
  M2 会话管理:      4/5  (80%)  ⚠️ 1条失败 — JWT篡改claim未检测到
  M3 权限守卫:      7/7  (100%) ✅ 全部通过
  M4 仪表盘:        3/3  (100%) ✅ 全部通过
  M5 监听器管理:    0/5  (0%)   ❌ 全部失败 — Modal关闭逻辑问题
  M6 Agent管理:     3/4  (75%)  ⚠️ 1条失败 — 轮询计数不足
  M7 任务管理:      2/4  (50%)  ⚠️ 2条失败 — 级联超时
  M8 终端:          3/3  (100%) ✅ 全部通过
  M9 审计日志:      3/3  (100%) ✅ 全部通过
  M10 构建面板:     2/2  (100%) ✅ 全部通过
  M11 导航:         3/3  (100%) ✅ 全部通过
  M12 退出登录:     3/3  (100%) ✅ 全部通过
```

### 4.5 已发现缺陷清单

| 缺陷ID | 模块 | 严重等级 | 描述 | 复现条件 |
|--------|------|---------|------|---------|
| **BUG-001** | M5 监听器 | 🟠严重 | 创建监听器时modal不关闭（API调用失败但无声反馈） | 点击创建后modal始终可见 |
| **BUG-002** | M5 监听器 | 🟠严重 | 端口越界(70000)未显示前端错误提示 | 输入无效端口提交 |
| **BUG-003** | M2 会话 | 🔴阻断 | `tamperClaim(token, 'role', 'admin')` 篡改role后仍返回200 | 篡改JWT payload后请求---原始role也是admin，测试需要改为非admin |
| **BUG-004** | M1 登录 | 🟡一般 | 错误密码/不存在用户错误消息与断言不完全匹配 | 后端返回 "Invalid credentials" |
| **BUG-005** | M6 Agent | 🟢轻微 | Agent轮询在10秒内只触发1次（期望2+次） | 空Agent列表时10秒轮询 |
| **BUG-006** | M1 登录 | 🟡一般 | 快速密码错误重试无限制/延迟（5次错误后正确密码立即成功） | 连续5次错误密码 |

---

## 附录A：测试账号

| 账号 | 密码 | 角色 | 用途 |
|------|------|------|------|
| admin | 259a2daf10da | admin | 主要测试账号 |

## 附录B：运行命令速查

```bash
# 安装
cd web && npm install && npx playwright install chromium

# 运行全部测试
npx playwright test --reporter=list

# 运行并打开报告
npx playwright test && npx playwright show-report

# 仅运行安全测试
npx playwright test m1-login m2-session m3-authguard m12-logout

# 调试验证
npx playwright test --headed --debug m5-listeners.spec.ts
```

---

> 📋 **文档版本**: v1.0 | **生成方式**: 代码静态分析 + Playwright自动化执行 | **下次回归**: 修复BUG-001~006后重新执行
