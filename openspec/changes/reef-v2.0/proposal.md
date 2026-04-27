# Proposal: Reef v2.0 — 生产级增强

## 背景

Reef v1.0/v1.1 已完成核心功能：WebSocket 协议、Server/Client、任务生命周期、
角色路由、E2E 测试、Docker Compose 部署、Admin 认证、Webhook 告警。

**当前限制：**
- 任务队列纯内存，Server 重启丢失所有任务
- 无性能基准，不知道系统瓶颈在哪
- 无可视化运维界面，只能 curl Admin API
- 单 Server 单点故障
- WebSocket 明文传输，生产环境不安全
- 告警仅支持 HTTP Webhook，无邮件/IM 通知

## 目标

将 Reef 从"可用的原型"升级为"生产级分布式任务编排系统"。

## v2.0 范围

| # | 特性 | 优先级 | 说明 |
|---|------|--------|------|
| 1 | **持久化任务队列** | P0 | SQLite WAL 模式，Server 重启不丢任务 |
| 2 | **性能基线测试** | P0 | 基准测试框架 + 10/50/100 并发客户端压测 |
| 3 | **Web UI 仪表盘** | P1 | 实时状态、任务列表、客户端拓扑、指标图表 |
| 4 | **TLS 原生支持** | P1 | Server/Client WebSocket 和 Admin API 支持 TLS |
| 5 | **多通道告警通知** | P2 | 邮件 (SMTP)、Slack、飞书、企业微信通知 |
| 6 | **多 Server 联邦** | P3 | 跨数据中心任务路由，Raft 共识（探索性） |

**不在 v2.0 范围：** 任务 DAG 编排（v3）、GPU 资源调度（v3）、Kubernetes Operator（v3）。

## Approach

### 1. 持久化任务队列

- 新增 `pkg/reef/server/store/` 包，定义 `TaskStore` 接口
- 实现 `SQLiteStore`：WAL 模式，任务表 + 尝试记录表
- `TaskQueue` 改为 `PersistentQueue`，包装 `TaskStore` + 内存缓存
- Server 启动时从 SQLite 恢复未完成任务
- 提供 `MemoryStore` 实现用于测试和小规模部署
- 配置项：`store_type`（`memory` | `sqlite`）、`store_path`

### 2. 性能基线测试

- 新增 `test/perf/` 包
- 基准测试场景：任务提交吞吐、调度延迟、WebSocket 消息吞吐、并发连接
- 压测工具：可配置并发数、任务数、超时
- 输出 JSON 报告：p50/p95/p99 延迟、吞吐 ops/sec、错误率
- CI 集成：性能回归检测

### 3. Web UI 仪表盘

- 新增 `pkg/reef/server/ui/` 包
- 嵌入式 SPA（Go `embed`），无外部依赖
- 页面：概览、任务列表、客户端拓扑、实时指标
- 数据源：复用 Admin API + WebSocket 推送实时更新
- 技术：纯 HTML/CSS/JS，Chart.js 图表

### 4. TLS 原生支持

- `Config` 增加 `TLS` 配置块（cert_file、key_file、ca_file）
- WebSocket 和 Admin HTTP Server 支持 `tls.ListenAndServe`
- Client Connector 支持 `wss://` + 自定义 CA
- 配置项：`tls.enabled`、`tls.cert_file`、`tls.key_file`、`tls.ca_file`、`tls.verify`

### 5. 多通道告警通知

- 新增 `pkg/reef/server/notify/` 包
- 定义 `Notifier` 接口
- 实现：`WebhookNotifier`（已有）、`SMTPNotifier`、`SlackNotifier`、`FeishuNotifier`、`WeComNotifier`
- `NotificationManager` 管理多个 Notifier，扇出发送
- 配置项：`notifications` 数组，每个元素指定 type + 参数

### 6. 多 Server 联邦（探索性）

- 新增 `pkg/reef/server/federation/` 包
- Raft 共识：使用 `hashicorp/raft` 库
- Leader 选举 + 任务分片
- 跨 Server 任务路由协议
- **注意：此特性为探索性，可能推迟到 v2.1**

## 成功标准

- `go test ./pkg/reef/... ./test/e2e/... ./test/perf/... -v` 全部通过
- 持久化队列：Server 重启后任务自动恢复
- 性能基线：100 并发客户端下 p99 延迟 < 500ms
- Web UI：可通过浏览器访问并实时显示状态
- TLS：wss:// 连接正常工作
- 告警：至少 2 个通道（Webhook + Slack）验证通过
- `make lint-docs` 通过
