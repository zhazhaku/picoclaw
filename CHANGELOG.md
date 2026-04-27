# Changelog

All notable changes to the Reef distributed multi-agent swarm system are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.3.0] — Reef v2.0 (Phase 1: Persistent Queue)

### Added

- **Persistent task queue** — `TaskStore` interface with `MemoryStore` and `SQLiteStore` implementations
- **SQLite WAL mode** — Server restarts automatically restore non-terminal tasks from SQLite database
- **`PersistentQueue`** — wraps `TaskStore` with in-memory cache for high-performance reads
- **`Queue` interface** — abstracts queue operations, enabling both in-memory and persistent implementations
- **Store configuration** — `store_type` (`memory` | `sqlite`) and `store_path` fields in `SwarmSettings`
- **CLI flags** — `--store-type` and `--store-path` for `picoclaw reef-server` command
- **Auto-directory creation** — SQLite store creates parent directories automatically
- **Comprehensive store tests** — 19 unit tests covering MemoryStore and SQLiteStore (CRUD, concurrent access, WAL mode, auto-directory)

## [0.2.0] — Reef v1.1

### Added

- **Config-driven Server mode** — `SwarmSettings.Mode` field (`"server"` | `"client"`) enables starting Reef Server via `config.json` without CLI flags
- **Docker Compose deployment** — `docker/docker-compose.reef.yml` with pre-configured Server + Coder + Analyst clients
- **Admin API authentication** — Bearer token protection for all `/admin/*` and `/tasks` endpoints (skipped when token is empty)
- **Admin webhook alerts** — `webhook_urls` config triggers POST notifications when tasks escalate to admin
- **Model routing hint** — `model_hint` field on task submission and dispatch payload for explicit model selection
- **Scheduler logger** — Scheduler now has its own structured logger for webhook and escalation events

### Changed

- `SwarmSettings` struct expanded with `Mode`, `WSAddr`, `AdminAddr`, `MaxQueue`, `MaxEscalations`, `WebhookURLs` fields
- `NewAdminServer()` now requires a `token` parameter
- `SchedulerOptions` includes `Logger` and `WebhookURLs`
- `msgTaskDispatch()` now accepts full `*Task` to populate all dispatch payload fields
- `OnDispatch` callback signature changed from `(taskID, clientID)` to `(task, clientID)`

### Fixed

- Documentation config examples now match actual code (`mode` field previously documented but not implemented)

## [0.1.0] — Reef v1.0

### Added

- **Reef v1.0.0** — Distributed multi-agent swarm orchestration system
  - WebSocket-based hub-and-spoke topology for Server-Client communication
  - Role-based task routing (`coder`, `analyst`, `tester`)
  - Skill-based client matching with load balancing
  - Task lifecycle management: dispatch, progress, completion, cancellation, pause/resume
  - Automatic failure retry with escalation policy (max 2 retries by default)
  - Client heartbeat and stale detection
  - Connection resilience: buffered control messages, reconnection support
  - HTTP Admin API: `/admin/status`, `/admin/tasks`, `POST /tasks`
  - YAML-based custom role configuration in `skills/roles/`
  - CLI command: `picoclaw reef-server`
  - Comprehensive E2E integration test suite (17 scenarios)
  - Full documentation: architecture, deployment, API reference, protocol spec

### Fixed

- WebSocket handshake now calls `scheduler.HandleClientAvailable()` after client registration, ensuring queued tasks are dispatched to newly connected clients.
