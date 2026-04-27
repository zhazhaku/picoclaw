package perf

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Scenario 1: Task Submission Throughput
// ---------------------------------------------------------------------------

func TestPerf_TaskSubmitThroughput_1(t *testing.T) {
	perfTaskSubmitThroughput(t, 1, 100)
}

func TestPerf_TaskSubmitThroughput_10(t *testing.T) {
	perfTaskSubmitThroughput(t, 10, 100)
}

func TestPerf_TaskSubmitThroughput_50(t *testing.T) {
	perfTaskSubmitThroughput(t, 50, 100)
}

func perfTaskSubmitThroughput(t *testing.T, concurrency, totalTasks int) {
	srv := NewPerfServer(t)
	defer srv.Shutdown()

	// Use a counter for unique task instructions
	var counter int64
	counterPtr := &counter

	latenciesUs, errCount := RunConcurrent(concurrency, totalTasks, func() error {
		n := int(*counterPtr)
		*counterPtr++
		_, err := srv.SubmitTask(
			fmt.Sprintf("perf-task-%d", n),
			"coder",
		)
		return err
	})

	// Convert to ms for percentiles
	latenciesMs := LatenciesToMs(latenciesUs)

	// Calculate total duration from sum of latencies
	var totalUs int64
	for _, l := range latenciesUs {
		totalUs += l
	}
	durationMs := totalUs / 1000
	if durationMs == 0 {
		durationMs = 1
	}

	p50, p95, p99 := CalculatePercentiles(latenciesMs)
	throughput := float64(totalTasks) / (float64(durationMs) / 1000.0)
	errRate := float64(errCount) / float64(totalTasks) * 100

	report := Report{
		TestName:  fmt.Sprintf("task_submit_c%d", concurrency),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Config: TestConfig{
			Concurrency: concurrency,
			TotalTasks:  totalTasks,
			ServerAddr:  srv.AdminAddr,
		},
		Results: TestResults{
			DurationMs:    durationMs,
			ThroughputOps: throughput,
			LatencyP50Ms:  p50,
			LatencyP95Ms:  p95,
			LatencyP99Ms:  p99,
			ErrorCount:    errCount,
			ErrorRate:     errRate,
		},
	}

	// Save report
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	reportPath := filepath.Join(dir, "results", fmt.Sprintf("task_submit_c%d.json", concurrency))
	if err := SaveReport(reportPath, report); err != nil {
		t.Fatalf("save report: %v", err)
	}

	t.Logf("Task Submit (c=%d): p50=%dms p95=%dms p99=%dms throughput=%.1f ops/s errors=%d (%.1f%%)",
		concurrency, p50, p95, p99, throughput, errCount, errRate)

	if errCount > 0 {
		t.Errorf("got %d errors out of %d requests", errCount, totalTasks)
	}
}

// ---------------------------------------------------------------------------
// Scenario 2: Admin API Status Throughput
// ---------------------------------------------------------------------------

func TestPerf_StatusQueryThroughput_1(t *testing.T) {
	perfStatusQueryThroughput(t, 1, 200)
}

func TestPerf_StatusQueryThroughput_10(t *testing.T) {
	perfStatusQueryThroughput(t, 10, 200)
}

func TestPerf_StatusQueryThroughput_50(t *testing.T) {
	perfStatusQueryThroughput(t, 50, 200)
}

func perfStatusQueryThroughput(t *testing.T, concurrency, totalOps int) {
	srv := NewPerfServer(t)
	defer srv.Shutdown()

	latenciesUs, errCount := RunConcurrent(concurrency, totalOps, func() error {
		return srv.GetStatus()
	})

	latenciesMs := LatenciesToMs(latenciesUs)

	var totalUs int64
	for _, l := range latenciesUs {
		totalUs += l
	}
	durationMs := totalUs / 1000
	if durationMs == 0 {
		durationMs = 1
	}

	p50, p95, p99 := CalculatePercentiles(latenciesMs)
	throughput := float64(totalOps) / (float64(durationMs) / 1000.0)
	errRate := float64(errCount) / float64(totalOps) * 100

	report := Report{
		TestName:  fmt.Sprintf("status_query_c%d", concurrency),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Config: TestConfig{
			Concurrency: concurrency,
			TotalTasks:  totalOps,
			ServerAddr:  srv.AdminAddr,
		},
		Results: TestResults{
			DurationMs:    durationMs,
			ThroughputOps: throughput,
			LatencyP50Ms:  p50,
			LatencyP95Ms:  p95,
			LatencyP99Ms:  p99,
			ErrorCount:    errCount,
			ErrorRate:     errRate,
		},
	}

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	reportPath := filepath.Join(dir, "results", fmt.Sprintf("status_query_c%d.json", concurrency))
	if err := SaveReport(reportPath, report); err != nil {
		t.Fatalf("save report: %v", err)
	}

	t.Logf("Status Query (c=%d): p50=%dms p95=%dms p99=%dms throughput=%.1f ops/s errors=%d (%.1f%%)",
		concurrency, p50, p95, p99, throughput, errCount, errRate)

	if errCount > 0 {
		t.Errorf("got %d errors out of %d requests", errCount, totalOps)
	}
}

// ---------------------------------------------------------------------------
// Scenario 3: Task List Throughput (with pre-populated tasks)
// ---------------------------------------------------------------------------

func TestPerf_TaskListThroughput_1(t *testing.T) {
	perfTaskListThroughput(t, 1, 200, 50)
}

func TestPerf_TaskListThroughput_10(t *testing.T) {
	perfTaskListThroughput(t, 10, 200, 50)
}

func TestPerf_TaskListThroughput_50(t *testing.T) {
	perfTaskListThroughput(t, 50, 200, 50)
}

func perfTaskListThroughput(t *testing.T, concurrency, totalOps, prePopulate int) {
	srv := NewPerfServer(t)
	defer srv.Shutdown()

	// Pre-populate with tasks
	for i := 0; i < prePopulate; i++ {
		if _, err := srv.SubmitTask(fmt.Sprintf("prepop-%d", i), "coder"); err != nil {
			t.Fatalf("pre-populate task %d: %v", i, err)
		}
	}

	latenciesUs, errCount := RunConcurrent(concurrency, totalOps, func() error {
		return srv.GetTasks()
	})

	latenciesMs := LatenciesToMs(latenciesUs)

	var totalUs int64
	for _, l := range latenciesUs {
		totalUs += l
	}
	durationMs := totalUs / 1000
	if durationMs == 0 {
		durationMs = 1
	}

	p50, p95, p99 := CalculatePercentiles(latenciesMs)
	throughput := float64(totalOps) / (float64(durationMs) / 1000.0)
	errRate := float64(errCount) / float64(totalOps) * 100

	report := Report{
		TestName:  fmt.Sprintf("task_list_c%d", concurrency),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Config: TestConfig{
			Concurrency: concurrency,
			TotalTasks:  totalOps,
			ServerAddr:  srv.AdminAddr,
		},
		Results: TestResults{
			DurationMs:    durationMs,
			ThroughputOps: throughput,
			LatencyP50Ms:  p50,
			LatencyP95Ms:  p95,
			LatencyP99Ms:  p99,
			ErrorCount:    errCount,
			ErrorRate:     errRate,
		},
	}

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	reportPath := filepath.Join(dir, "results", fmt.Sprintf("task_list_c%d.json", concurrency))
	if err := SaveReport(reportPath, report); err != nil {
		t.Fatalf("save report: %v", err)
	}

	t.Logf("Task List (c=%d, prepop=%d): p50=%dms p95=%dms p99=%dms throughput=%.1f ops/s errors=%d (%.1f%%)",
		concurrency, prePopulate, p50, p95, p99, throughput, errCount, errRate)

	if errCount > 0 {
		t.Errorf("got %d errors out of %d requests", errCount, totalOps)
	}
}

// ---------------------------------------------------------------------------
// Scenario 4: Regression comparison (utility test)
// ---------------------------------------------------------------------------

func TestPerf_RegressionComparison(t *testing.T) {
	// Create two synthetic reports and verify Compare works correctly
	baseline := Report{
		TestName: "test_scenario",
		Results: TestResults{
			LatencyP99Ms:  100,
			ThroughputOps: 1000,
		},
	}

	// Case 1: No regression
	current := Report{
		TestName: "test_scenario",
		Results: TestResults{
			LatencyP99Ms:  110, // +10% — under 20% threshold
			ThroughputOps: 900, // -10% — under 15% threshold
		},
	}
	reg := Compare(baseline, current)
	if reg.IsRegression {
		t.Errorf("expected no regression, got regression (p99 change=%.1f%%, throughput change=%.1f%%)",
			reg.P99Change, reg.ThroughputChange)
	}

	// Case 2: Latency regression
	current.Results.LatencyP99Ms = 130 // +30% — over 20% threshold
	current.Results.ThroughputOps = 900
	reg = Compare(baseline, current)
	if !reg.IsRegression {
		t.Error("expected latency regression to be detected")
	}

	// Case 3: Throughput regression
	current.Results.LatencyP99Ms = 110
	current.Results.ThroughputOps = 800 // -20% — over 15% threshold
	reg = Compare(baseline, current)
	if !reg.IsRegression {
		t.Error("expected throughput regression to be detected")
	}

	// Verify percentile calculation with enough data points for meaningful percentiles
	latencies := make([]int64, 1000)
	for i := range latencies {
		latencies[i] = int64(i + 1) // 1..1000
	}
	latencies[999] = 10000 // outlier at the tail
	p50, p95, p99 := CalculatePercentiles(latencies)
	if p50 < 490 || p50 > 510 {
		t.Errorf("p50: got %d, want ~500", p50)
	}
	if p95 < 940 || p95 > 960 {
		t.Errorf("p95: got %d, want ~950", p95)
	}
	if p99 < 980 {
		t.Errorf("p99: got %d, want >= 980", p99)
	}
	t.Logf("Percentiles (n=1000): p50=%d p95=%d p99=%d", p50, p95, p99)

	// Also test with small slice
	small := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 100}
	p50s, p95s, p99s := CalculatePercentiles(small)
	t.Logf("Percentiles (n=10): p50=%d p95=%d p99=%d", p50s, p95s, p99s)
	// With 10 elements, p95 and p99 both map to index 8 (value 9)
	if p50s != 5 && p50s != 6 {
		t.Errorf("p50: got %d, want 5 or 6", p50s)
	}

	// Verify report save/load round-trip
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "test_report.json")
	report := Report{
		TestName:  "round_trip_test",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Config:    TestConfig{Concurrency: 1, TotalTasks: 10},
		Results:   TestResults{DurationMs: 500, ThroughputOps: 20, LatencyP50Ms: 5, LatencyP95Ms: 8, LatencyP99Ms: 9},
	}
	if err := SaveReport(reportPath, report); err != nil {
		t.Fatalf("save report: %v", err)
	}
	loaded, err := LoadReport(reportPath)
	if err != nil {
		t.Fatalf("load report: %v", err)
	}
	if loaded.TestName != report.TestName {
		t.Errorf("test name mismatch: got %s, want %s", loaded.TestName, report.TestName)
	}
	if loaded.Results.LatencyP99Ms != report.Results.LatencyP99Ms {
		t.Errorf("p99 mismatch: got %d, want %d", loaded.Results.LatencyP99Ms, report.Results.LatencyP99Ms)
	}
}
