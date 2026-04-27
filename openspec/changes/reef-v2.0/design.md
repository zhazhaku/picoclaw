# Design: Reef v2.0

## 架构概览

```
                    ┌─────────────────────────────────────┐
                    │           Reef Server v2             │
                    │                                     │
                    │  ┌─────────┐  ┌──────────────────┐  │
                    │  │ Admin   │  │   Web UI (SPA)   │  │
                    │  │ API     │  │   embedded via    │  │
                    │  │ + TLS   │  │   go:embed        │  │
                    │  └────┬────┘  └──────────────────┘  │
                    │       │                              │
                    │  ┌────┴────────────────────────┐    │
                    │  │       Scheduler              │    │
                    │  │  (with NotificationManager)  │    │
                    │  └────┬──────────┬─────────────┘    │
                    │       │          │                   │
                    │  ┌────┴────┐ ┌───┴──────────────┐   │
                    │  │Persistent│ │  WebSocket Server │   │
                    │  │ Queue   │ │  + TLS            │   │
                    │  │ (Store) │ │                   │   │
                    │  └────┬────┘ └───────────────────┘   │
                    │       │                              │
                    │  ┌────┴──────────┐                   │
                    │  │ TaskStore     │                   │
                    │  │ ┌───────────┐ │                   │
                    │  │ │ Memory    │ │                   │
                    │  │ │ SQLite    │ │                   │
                    │  │ └───────────┘ │                   │
                    │  └───────────────┘                   │
                    │                                     │
                    │  ┌───────────────────┐              │
                    │  │ Federation (opt)  │              │
                    │  │ Raft consensus    │              │
                    │  └───────────────────┘              │
                    └─────────────────────────────────────┘
```

## 文件结构

```
新增文件:
  pkg/reef/server/store/
    store.go              — TaskStore 接口定义
    memory.go             — 内存实现
    sqlite.go             — SQLite WAL 实现
    sqlite_test.go        — SQLite 单元测试
    filter.go             — TaskFilter 类型定义
  pkg/reef/server/notify/
    notifier.go           — Notifier 接口 + Alert 类型
    manager.go            — NotificationManager 扇出
    webhook.go            — 从 webhook.go 迁移
    slack.go              — Slack Incoming Webhook
    smtp.go               — SMTP 邮件
    feishu.go             — 飞书 Webhook
    wecom.go              — 企业微信 Webhook
    notifier_test.go      — 单元测试
  pkg/reef/server/ui/
    ui.go                 — HTTP handler + go:embed
    static/
      index.html          — SPA 入口
      app.js              — 主逻辑
      style.css           — 样式
      favicon.svg         — 图标
  pkg/reef/server/federation/
    raft.go               — Raft 节点管理
    fsm.go                — Raft FSM（有限状态机）
    transport.go          — 跨 Server 通信
    router.go             — 任务路由
  test/perf/
    perf_test.go          — 基准测试框架
    scenarios_test.go     — 具体测试场景
    report.go             — JSON 报告生成
    results/
      .gitkeep

修改文件:
  pkg/reef/server/server.go     — 集成 Store + Notify + UI + TLS
  pkg/reef/server/scheduler.go  — 使用 PersistentQueue
  pkg/reef/server/admin.go      — UI 路由 + TLS
  pkg/reef/server/websocket.go  — TLS 支持
  pkg/reef/server/queue.go      — 包装 TaskStore
  pkg/reef/client/connector.go  — wss:// + 自定义 CA
  pkg/config/config_channel.go  — SwarmSettings 增加新字段
  pkg/gateway/gateway.go        — 传递新配置
  docker/docker-compose.reef.yml — 更新
  docs/reef/*.md                — 更新文档
```

---

## 1. 持久化任务队列 — 详细设计

### TaskStore 接口

```go
// pkg/reef/server/store/store.go
type TaskFilter struct {
    Statuses []reef.TaskStatus
    Roles    []string
    Limit    int
    Offset   int
}

type TaskStore interface {
    SaveTask(task *reef.Task) error
    GetTask(id string) (*reef.Task, error)
    UpdateTask(task *reef.Task) error
    DeleteTask(id string) error
    ListTasks(filter TaskFilter) ([]*reef.Task, error)
    SaveAttempt(taskID string, attempt reef.AttemptRecord) error
    GetAttempts(taskID string) ([]reef.AttemptRecord, error)
    Close() error
}
```

### SQLite Schema

```sql
CREATE TABLE IF NOT EXISTS tasks (
    id              TEXT PRIMARY KEY,
    status          TEXT NOT NULL,
    instruction     TEXT NOT NULL,
    required_role   TEXT NOT NULL,
    required_skills TEXT,  -- JSON array
    max_retries     INTEGER DEFAULT 3,
    timeout_ms      INTEGER DEFAULT 300000,
    model_hint      TEXT,
    assigned_client TEXT,
    result          TEXT,  -- JSON
    error           TEXT,  -- JSON
    escalation_count INTEGER DEFAULT 0,
    pause_reason    TEXT,
    created_at      INTEGER NOT NULL,
    assigned_at     INTEGER,
    started_at      INTEGER,
    completed_at    INTEGER
);

CREATE TABLE IF NOT EXISTS task_attempts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id         TEXT NOT NULL REFERENCES tasks(id),
    attempt_number  INTEGER NOT NULL,
    started_at      INTEGER NOT NULL,
    ended_at        INTEGER NOT NULL,
    status          TEXT NOT NULL,
    error_message   TEXT,
    client_id       TEXT
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_role ON tasks(required_role);
CREATE INDEX IF NOT EXISTS idx_task_attempts_task_id ON task_attempts(task_id);
```

### PersistentQueue 设计

```go
// 改造后的 queue.go
type PersistentQueue struct {
    store   store.TaskStore
    cache   []*reef.Task  // 内存缓存活跃任务
    mu      sync.Mutex
    maxLen  int
    maxAge  time.Duration
}

func NewPersistentQueue(s store.TaskStore, maxLen int, maxAge time.Duration) *PersistentQueue {
    q := &PersistentQueue{store: s, maxLen: maxLen, maxAge: maxAge}
    q.restore()  // 从 store 恢复未完成任务
    return q
}

func (q *PersistentQueue) restore() {
    tasks, _ := q.store.ListTasks(store.TaskFilter{
        Statuses: []reef.TaskStatus{reef.TaskQueued, reef.TaskRunning, reef.TaskAssigned, reef.TaskPaused},
    })
    for _, t := range tasks {
        if t.Status == reef.TaskRunning || t.Status == reef.TaskAssigned {
            t.Status = reef.TaskQueued  // 重置为 Queued
            _ = q.store.UpdateTask(t)
        }
        q.cache = append(q.cache, t)
    }
}
```

---

## 2. 性能基线测试 — 详细设计

### 测试场景矩阵

| 场景 | 并发数 | 任务数 | 测量指标 |
|------|--------|--------|----------|
| 任务提交吞吐 | 1/10/50/100 | 1000 | ops/sec, p50/p95/p99 |
| 调度延迟 | 1 | 100 | 提交→dispatch 延迟 |
| WebSocket 心跳吞吐 | 10/50/100 | 10000 msg | msg/sec |
| 并发连接建立 | 10/50/100 | — | 连接建立延迟 |
| 端到端任务完成 | 10 | 100 | 提交→完成延迟 |

### 报告格式

```json
{
  "test_name": "task_submit_throughput",
  "timestamp": "2026-04-28T10:00:00Z",
  "config": {
    "concurrency": 10,
    "total_tasks": 1000,
    "server_addr": "127.0.0.1:8080"
  },
  "results": {
    "duration_ms": 5230,
    "throughput_ops": 191.2,
    "latency_p50_ms": 12,
    "latency_p95_ms": 45,
    "latency_p99_ms": 120,
    "error_count": 0,
    "error_rate": 0
  }
}
```

---

## 3. Web UI 仪表盘 — 详细设计

### 技术选型

- **前端：** 纯 HTML + CSS + JavaScript（无框架，最小化依赖）
- **图表：** Chart.js（内嵌，~60KB gzip）
- **实时更新：** Server-Sent Events (SSE) 或 WebSocket
- **打包：** Go `embed` 嵌入二进制

### 页面结构

```
/                    → 概览（redirect to /ui）
/ui                  → SPA 入口
/ui/                 → 概览页面
/ui/tasks            → 任务列表
/ui/clients          → 客户端拓扑
/ui/metrics          → 指标图表
/api/v2/status       → 增强状态 API（JSON）
/api/v2/tasks        → 增强任务 API（JSON, 分页）
/api/v2/clients      → 增强客户端 API（JSON）
/api/v2/events       → SSE 实时事件流
```

### SSE 事件格式

```
event: task_update
data: {"task_id":"task-1","status":"Running","assigned_client":"coder-1"}

event: client_update
data: {"client_id":"coder-1","state":"connected","load":2}

event: stats_update
data: {"queued":5,"running":3,"completed":100,"failed":2}
```

---

## 4. TLS 原生支持 — 详细设计

### 配置结构

```go
type TLSConfig struct {
    Enabled  bool   `json:"enabled"`
    CertFile string `json:"cert_file"`
    KeyFile  string `json:"key_file"`
    CAFile   string `json:"ca_file,omitempty"`   // Client: 自定义 CA
    Verify   bool   `json:"verify,omitempty"`     // Client: 验证服务端证书
}
```

### Server TLS 启动

```go
func (s *Server) Start() error {
    // ... 现有逻辑
    if s.config.TLS != nil && s.config.TLS.Enabled {
        cert, err := tls.LoadX509KeyPair(s.config.TLS.CertFile, s.config.TLS.KeyFile)
        // ...
        tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
        wsTLSListener := tls.NewListener(wsListener, tlsCfg)
        adminTLSListener := tls.NewListener(adminListener, tlsCfg)
        // 使用 TLS listener
    }
}
```

### Client TLS 连接

```go
func (c *Connector) dialTLS(wsURL string, caFile string) (*websocket.Conn, error) {
    caCert, _ := os.ReadFile(caFile)
    pool := x509.NewCertPool()
    pool.AppendCertsFromPEM(caCert)
    tlsCfg := &tls.Config{RootCAs: pool}
    dialer := websocket.Dialer{
        TLSClientConfig:  tlsCfg,
        HandshakeTimeout: 10 * time.Second,
    }
    return dialer.Dial(wsURL, header)
}
```

---

## 5. 多通道告警通知 — 详细设计

### 配置结构

```go
// config_channel.go SwarmSettings 新增
type NotificationConfig struct {
    Type     string `json:"type"`               // "webhook" | "slack" | "smtp" | "feishu" | "wecom"
    // Webhook
    URL      string `json:"url,omitempty"`
    // Slack
    WebhookURL string `json:"webhook_url,omitempty"`
    // SMTP
    SMTPHost string `json:"smtp_host,omitempty"`
    SMTPPort int    `json:"smtp_port,omitempty"`
    From     string `json:"from,omitempty"`
    To       []string `json:"to,omitempty"`
    Username string `json:"username,omitempty"`
    Password string `json:"password,omitempty"`
    // 飞书/企业微信
    HookURL  string `json:"hook_url,omitempty"`
}
```

### Notifier 接口

```go
// pkg/reef/server/notify/notifier.go
type Alert struct {
    Event       string
    TaskID      string
    Status      string
    Instruction string
    Role        string
    Error       *reef.TaskError
    Attempts    []reef.AttemptRecord
    Timestamp   time.Time
}

type Notifier interface {
    Name() string
    Notify(ctx context.Context, alert Alert) error
}
```

### NotificationManager

```go
// pkg/reef/server/notify/manager.go
type Manager struct {
    notifiers []Notifier
    logger    *slog.Logger
}

func (m *Manager) NotifyAll(ctx context.Context, alert Alert) {
    for _, n := range m.notifiers {
        go func(notifier Notifier) {
            if err := notifier.Notify(ctx, alert); err != nil {
                m.logger.Warn("notification failed",
                    slog.String("notifier", notifier.Name()),
                    slog.String("error", err.Error()))
            }
        }(n)
    }
}
```

---

## 6. 多 Server 联邦 — 详细设计（探索性）

### 架构

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│ Server A │◄───►│ Server B │◄───►│ Server C │
│ (Leader) │ Raft│(Follower)│ Raft│(Follower) │
└────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │
  ┌──┴──┐          ┌──┴──┐          ┌──┴──┐
  │Cli 1│          │Cli 2│          │Cli 3│
  └─────┘          └─────┘          └─────┘
```

### 依赖

- `github.com/hashicorp/raft` — Raft 共识
- `github.com/hashicorp/raft-boltdb` — Raft 日志持久化

### 任务路由

- Client 连接到任意 Server 节点
- 非 Leader 节点将任务请求代理到 Leader
- Leader 负责调度决策
- 结果原路返回

**注意：此特性复杂度极高，建议作为 v2.1 独立里程碑。**
