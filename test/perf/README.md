# Reef Performance Baseline Tests

Performance benchmarking framework for the Reef distributed task server.

## Running Tests

```bash
# Run all performance tests
go test ./test/perf/... -v -count=1 -timeout 120s

# Run a specific scenario
go test ./test/perf/... -v -run TestPerf_TaskSubmitThroughput_1

# Run with race detector (slower, but catches data races)
go test ./test/perf/... -v -race -count=1 -timeout 120s
```

## Test Scenarios

### 1. Task Submission Throughput (`task_submit`)

Measures how fast the server accepts new tasks via `POST /tasks`.

| Test | Concurrency | Total Tasks |
|------|-------------|-------------|
| `TestPerf_TaskSubmitThroughput_1` | 1 | 100 |
| `TestPerf_TaskSubmitThroughput_10` | 10 | 100 |
| `TestPerf_TaskSubmitThroughput_50` | 50 | 100 |

### 2. Status Query Throughput (`status_query`)

Measures `GET /admin/status` latency under concurrent load.

| Test | Concurrency | Total Requests |
|------|-------------|----------------|
| `TestPerf_StatusQueryThroughput_1` | 1 | 200 |
| `TestPerf_StatusQueryThroughput_10` | 10 | 200 |
| `TestPerf_StatusQueryThroughput_50` | 50 | 200 |

### 3. Task List Throughput (`task_list`)

Measures `GET /admin/tasks` latency with pre-populated tasks.

| Test | Concurrency | Total Requests | Pre-populated |
|------|-------------|----------------|---------------|
| `TestPerf_TaskListThroughput_1` | 1 | 200 | 50 |
| `TestPerf_TaskListThroughput_10` | 10 | 200 | 50 |
| `TestPerf_TaskListThroughput_50` | 50 | 200 | 50 |

### 4. Regression Comparison (`TestPerf_RegressionComparison`)

Validates the `Compare()` function and report save/load round-trip.

## Output

Reports are saved to `test/perf/results/` as JSON files:

```
test/perf/results/
├── task_submit_c1.json
├── task_submit_c10.json
├── task_submit_c50.json
├── status_query_c1.json
├── status_query_c10.json
├── status_query_c50.json
├── task_list_c1.json
├── task_list_c10.json
└── task_list_c50.json
```

## Report Format

```json
{
  "test_name": "task_submit_c10",
  "timestamp": "2026-04-28T00:00:00Z",
  "config": {
    "concurrency": 10,
    "total_tasks": 100,
    "server_addr": "127.0.0.1:12345"
  },
  "results": {
    "duration_ms": 500,
    "throughput_ops": 200.0,
    "latency_p50_ms": 4,
    "latency_p95_ms": 8,
    "latency_p99_ms": 12,
    "error_count": 0,
    "error_rate": 0.0
  }
}
```

## Regression Detection

Use `Compare(baseline, current)` to detect regressions:

- **Latency regression**: P99 increases by more than 20%
- **Throughput regression**: Throughput decreases by more than 15%

## Interpreting Results

- **p50 (median)**: Typical request latency
- **p95**: Latency for 95% of requests (tail latency indicator)
- **p99**: Worst-case latency for 99% of requests
- **throughput_ops**: Requests completed per second
- **error_rate**: Percentage of failed requests

## Baseline Workflow

1. Run tests on a known-good commit and save the `results/` directory
2. Commit the results as your baseline
3. On future changes, run tests and compare with `Compare()`
4. If `IsRegression` is true, investigate before merging
