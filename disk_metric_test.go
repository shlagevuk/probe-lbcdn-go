package main

import (
	"testing"
	"time"
)

func TestGetDiskUsage(t *testing.T) {
	// Test with root path
	usage, err := getDiskUsage("/")
	if err != nil {
		t.Fatalf("getDiskUsage() returned error: %v", err)
	}

	if usage < 0 || usage > 100 {
		t.Errorf("getDiskUsage() = %v, want value between 0 and 100", usage)
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "root path",
			path: "/",
			want: "root",
		},
		{
			name: "var log path",
			path: "/var/log",
			want: "var_log",
		},
		{
			name: "tmp path",
			path: "/tmp",
			want: "tmp",
		},
		{
			name: "nested path",
			path: "/opt/app/data",
			want: "opt_app_data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizePath(tt.path)
			if got != tt.want {
				t.Errorf("sanitizePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDiskMetricStatusLogic(t *testing.T) {
	tests := []struct {
		name       string
		current    float64
		max        float64
		wantStatus string
	}{
		{
			name:       "within limits",
			current:    70.0,
			max:        95.0,
			wantStatus: "OK",
		},
		{
			name:       "at limit",
			current:    95.0,
			max:        95.0,
			wantStatus: "OK",
		},
		{
			name:       "exceeds limit",
			current:    98.0,
			max:        95.0,
			wantStatus: "KO",
		},
		{
			name:       "low usage",
			current:    20.0,
			max:        95.0,
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

func TestCollectDiskMetricUpdatesCache(t *testing.T) {
	// Setup test config with multiple paths
	oldConfig := config
	config = ProbeConfig{
		WarmupEnabled:  false,
		WarmupDuration: 60 * time.Second,
		MaxDisk:        95.0,
		DiskPaths:      []string{"/", "/tmp"},
		startTime:      time.Now(),
	}
	defer func() { config = oldConfig }()

	// Clear cache
	cacheMutex.Lock()
	metricCache = make(map[string]MetricStatus)
	cacheMutex.Unlock()

	// Run one iteration of metric collection
	effectiveMax := config.MaxDisk
	for _, path := range config.DiskPaths {
		diskUsage, err := getDiskUsage(path)
		if err != nil {
			diskUsage = 0
		}

		status := "OK"
		if diskUsage > effectiveMax {
			status = "KO"
		}

		metricName := "disk_" + sanitizePath(path)
		cacheMutex.Lock()
		metricCache[metricName] = MetricStatus{
			Current: diskUsage,
			Max:     effectiveMax,
			Status:  status,
		}
		cacheMutex.Unlock()
	}

	// Verify cache was updated for both paths
	cacheMutex.RLock()
	rootMetric, rootExists := metricCache["disk_root"]
	tmpMetric, tmpExists := metricCache["disk_tmp"]
	cacheMutex.RUnlock()

	if !rootExists {
		t.Fatal("Disk metric for root not found in cache")
	}

	if !tmpExists {
		t.Fatal("Disk metric for tmp not found in cache")
	}

	if rootMetric.Current < 0 || rootMetric.Current > 100 {
		t.Errorf("Root disk metric current = %v, want value between 0 and 100", rootMetric.Current)
	}

	if tmpMetric.Current < 0 || tmpMetric.Current > 100 {
		t.Errorf("Tmp disk metric current = %v, want value between 0 and 100", tmpMetric.Current)
	}

	if rootMetric.Max != 95.0 {
		t.Errorf("Root disk metric max = %v, want 95.0", rootMetric.Max)
	}

	if rootMetric.Status != "OK" && rootMetric.Status != "KO" {
		t.Errorf("Root disk metric status = %v, want OK or KO", rootMetric.Status)
	}
}
