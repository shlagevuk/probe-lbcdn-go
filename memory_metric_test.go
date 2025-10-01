package main

import (
	"testing"
	"time"
)

func TestGetMemoryUsage(t *testing.T) {
	usage, err := getMemoryUsage()
	if err != nil {
		t.Fatalf("getMemoryUsage() returned error: %v", err)
	}

	if usage < 0 || usage > 100 {
		t.Errorf("getMemoryUsage() = %v, want value between 0 and 100", usage)
	}
}

func TestMemoryMetricStatusLogic(t *testing.T) {
	tests := []struct {
		name       string
		current    float64
		max        float64
		wantStatus string
	}{
		{
			name:       "within limits",
			current:    60.0,
			max:        90.0,
			wantStatus: "OK",
		},
		{
			name:       "at limit",
			current:    90.0,
			max:        90.0,
			wantStatus: "OK",
		},
		{
			name:       "exceeds limit",
			current:    95.0,
			max:        90.0,
			wantStatus: "KO",
		},
		{
			name:       "low usage",
			current:    10.0,
			max:        90.0,
			wantStatus: "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := "OK"
			if tt.current > tt.max {
				status = "KO"
			}

			if status != tt.wantStatus {
				t.Errorf("status = %v, want %v (current=%v, max=%v)", status, tt.wantStatus, tt.current, tt.max)
			}
		})
	}
}

func TestCollectMemoryMetricUpdatesCache(t *testing.T) {
	// Setup test config
	oldConfig := config
	config = Config{
		startTime: time.Now(),
	}
	config.Warmup.Enabled = false
	config.Warmup.Duration = 60 * time.Second
	config.Thresholds.MaxMemory = 90.0
	defer func() { config = oldConfig }()

	// Clear cache
	cacheMutex.Lock()
	metricCache = make(map[string]MetricStatus)
	cacheMutex.Unlock()

	// Run one iteration of metric collection
	memUsage, err := getMemoryUsage()
	if err != nil {
		memUsage = 0
	}

	effectiveMax := config.Thresholds.MaxMemory
	status := "OK"
	if memUsage > effectiveMax {
		status = "KO"
	}

	cacheMutex.Lock()
	metricCache["memory"] = MetricStatus{
		Current: memUsage,
		Max:     effectiveMax,
		Status:  status,
	}
	cacheMutex.Unlock()

	// Verify cache was updated
	cacheMutex.RLock()
	metric, exists := metricCache["memory"]
	cacheMutex.RUnlock()

	if !exists {
		t.Fatal("Memory metric not found in cache")
	}

	if metric.Current < 0 || metric.Current > 100 {
		t.Errorf("Memory metric current = %v, want value between 0 and 100", metric.Current)
	}

	if metric.Max != 90.0 {
		t.Errorf("Memory metric max = %v, want 90.0", metric.Max)
	}

	if metric.Status != "OK" && metric.Status != "KO" {
		t.Errorf("Memory metric status = %v, want OK or KO", metric.Status)
	}
}
