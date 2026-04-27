# Tasks: Reef v1.1

## Phase 1: 配置与数据结构（基础层）

- [ ] `SwarmSettings` 增加 `Mode`、`WSAddr`、`AdminAddr`、`MaxQueue`、`MaxEscalations`、`WebhookURLs` 字段
- [ ] `Task` 增加 `ModelHint string` 字段
- [ ] `TaskDispatchPayload` 增加 `ModelHint string` 字段
- [ ] `SubmitTaskRequest` 增加 `ModelHint string` 字段
- [ ] `server.Config` 增加 `WebhookURLs []string` 字段
- [ ] `NewServer()` 从 Config 传递 WebhookURLs 到 Scheduler
- [ ] 编译验证：`go build ./...`

## Phase 2: Admin API 认证

- [ ] `AdminServer` 增加 `token` 字段
- [ ] 实现 `authMiddleware` 方法
- [ ] `RegisterRoutes` 中所有端点包裹 `authMiddleware`
- [ ] `NewAdminServer` 接收 token 参数
- [ ] 单元测试：有效 Token → 200
- [ ] 单元测试：无效 Token → 401
- [ ] 单元测试：无 Token → 401
- [ ] 单元测试：Token 为空时跳过认证

## Phase 3: Admin Webhook 告警

- [ ] 创建 `pkg/reef/server/webhook.go`
- [ ] 定义 `WebhookPayload` 结构体
- [ ] 实现 `sendWebhookAlert()` 函数（并发、超时、错误处理）
- [ ] `Scheduler` 增加 `webhookURLs` 字段
- [ ] `escalate()` 的 `EscalationToAdmin` 分支调用 `sendWebhookAlert`
- [ ] 单元测试：Webhook 被调用（httptest mock server）
- [ ] 单元测试：Webhook 失败不影响任务状态
- [ ] 单元测试：未配置 Webhook 时不发送

## Phase 4: 模型路由提示

- [ ] `NewTask()` 接受 `modelHint` 参数（或 setter）
- [ ] `admin.handleSubmitTask` 从请求体读取 `model_hint` 并设置到 Task
- [ ] `scheduler.dispatch` 将 `ModelHint` 传递到 `TaskDispatchPayload`
- [ ] `SwarmChannel` 接收 `task_dispatch` 时提取 `ModelHint`
- [ ] `TaskRunner` 将 `ModelHint` 传入 AgentLoop session
- [ ] 单元测试：ModelHint 从 Task → Dispatch → Client 端传递

## Phase 5: Mode 字段与 Gateway 集成

- [ ] `pkg/gateway/gateway.go` 启动时检测 `swarm.mode`
- [ ] `mode = "server"` 时：构建 `server.Config` 并启动 Server
- [ ] `mode = "server"` 时：阻塞等待信号（SIGTERM/SIGINT）
- [ ] `mode = "server"` 且 `ws_addr` 为空时：返回明确错误
- [ ] `mode = "client"` 或空时：行为不变
- [ ] CLI `picoclaw reef-server` 保持不变（独立于 config mode）

## Phase 6: Docker Compose

- [ ] 创建 `docker/docker-compose.reef.yml`
- [ ] 创建 `docker/reef-server-config.json`（mode=server）
- [ ] 创建 `docker/reef-client-coder-config.json`（mode=client, role=coder）
- [ ] 创建 `docker/reef-client-analyst-config.json`（mode=client, role=analyst）
- [ ] 验证：`docker compose -f docker/docker-compose.reef.yml config` 通过

## Phase 7: 文档更新

- [ ] `docs/reef/README.md` — 更新配置示例，移除不存在的字段
- [ ] `docs/reef/deployment.md` — 更新 Docker Compose 示例，指向实际文件
- [ ] `docs/reef/api.md` — 添加认证说明、model_hint 参数
- [ ] `docs/reef/protocol.md` — 添加 model_hint payload 字段
- [ ] `docs/reef/roles.md` — 无变更
- [ ] `README.md` — 无变更（v1.1 不改根 README）
- [ ] `CHANGELOG.md` — 添加 v1.1 变更记录
- [ ] 验证：`make lint-docs` 通过

## Phase 8: E2E 测试

- [ ] 测试：Admin API Token 认证（有效/无效/缺失/空）
- [ ] 测试：任务升级触发 Webhook（httptest mock）
- [ ] 测试：任务携带 model_hint 调度
- [ ] 测试：Server 模式通过 config 启动（需要 Gateway 集成测试或单独测试）
- [ ] 运行全部测试：`go test ./pkg/reef/... ./pkg/channels/swarm/... ./test/e2e/... -v`
- [ ] 验证无 flake（3 次运行）

## Phase 9: 提交与验证

- [ ] Git commit：`feat(reef): v1.1 — mode config, docker compose, webhook, auth, model hint`
- [ ] Git push
- [ ] 更新 `.planning/STATE.md`
