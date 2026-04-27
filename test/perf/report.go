// Package perf provides performance baseline tests for the Reef server.
package perf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// Report holds the results of a performance test.
type Report struct {
	TestName  string      `json:"test_name"`
	Timestamp string      `json:"timestamp"`
	Config    TestConfig  `json:"config"`
	Results   TestResults `json:"results"`
}

// TestConfig holds the test parameters.
type TestConfig struct {
	Concurrency int    `json:"concurrency"`
	TotalTasks  int    `json:"total_tasks"`
	ServerAddr  string `json:"server_addr"`
}

// TestResults holds the measured results.
type TestResults struct {
	DurationMs    int64   `json:"duration_ms"`
	ThroughputOps float64 `json:"throughput_ops"`
	LatencyP50Ms  int64   `json:"latency_p50_ms"`
	LatencyP95Ms  int64   `json:"latency_p95_ms"`
	LatencyP99Ms  int64   `json:"latency_p99_ms"`
	ErrorCount    int     `json:"error_count"`
	ErrorRate     float64 `json:"error_rate"`
}

// RegressionReport compares two reports.
type RegressionReport struct {
	TestName           string  `json:"test_name"`
	BaselineP99        int64   `json:"baseline_p99"`
	CurrentP99         int64   `json:"current_p99"`
	P99Change          float64 `json:"p99_change_pct"`
	BaselineThroughput float64 `json:"baseline_throughput"`
	CurrentThroughput  float64 `json:"current_throughput"`
	ThroughputChange   float64 `json:"throughput_change_pct"`
	IsRegression       bool    `json:"is_regression"`
}

// SaveReport writes a report to a JSON file.
func SaveReport(path string, r Report) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadReport reads a report from a JSON file.
func LoadReport(path string) (Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Report{}, err
	}
	var r Report
	err = json.Unmarshal(data, &r)
	return r, err
}

// Compare checks if current has regressed compared to baseline.
// Thresholds: p99 latency +20%, throughput -15%.
func Compare(baseline, current Report) RegressionReport {
	reg := RegressionReport{
		TestName:           current.TestName,
		BaselineP99:        baseline.Results.LatencyP99Ms,
		CurrentP99:         current.Results.LatencyP99Ms,
		BaselineThroughput: baseline.Results.ThroughputOps,
		CurrentThroughput:  current.Results.ThroughputOps,
	}

	if baseline.Results.LatencyP99Ms > 0 {
		reg.P99Change = float64(current.Results.LatencyP99Ms-baseline.Results.LatencyP99Ms) / float64(baseline.Results.LatencyP99Ms) * 100
	}
	if baseline.Results.ThroughputOps > 0 {
		reg.ThroughputChange = (current.Results.ThroughputOps - baseline.Results.ThroughputOps) / baseline.Results.ThroughputOps * 100
	}

	reg.IsRegression = reg.P99Change > 20 || reg.ThroughputChange < -15
	return reg
}

// CalculatePercentiles computes p50, p95, p99 from a sorted slice of latencies.
func CalculatePercentiles(latencies []int64) (p50, p95, p99 int64) {
	if len(latencies) == 0 {
		return 0, 0, 0
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p50 = percentile(latencies, 50)
	p95 = percentile(latencies, 95)
	p99 = percentile(latencies, 99)
	return
}

func percentile(sorted []int64, p int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(p) / 100 * float64(len(sorted)-1))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
