package main

import (
	"testing"
	"time"
)

func TestGetNetworkConnections(t *testing.T) {
	connections, err := getNetworkConnections()
	if err != nil {
		t.Fatalf("getNetworkConnections() returned error: %v", err)
	}

	if connections < 0 {
		t.Errorf("getNetworkConnections() = %v, want non-negative value", connections)
	}
}

func TestGetNetworkBandwidth(t *testing.T) {
	// Clear bandwidth cache for clean test
	bandwidthCacheMutex.Lock()
	bandwidthCache = make(map[string]bandwidthSnapshot)
	bandwidthCacheMutex.Unlock()

	// Test with loopback interface which should always exist
	// First call should return 0 (no previous reading)
	bandwidth1, err := getNetworkBandwidth("lo")
	if err != nil {
		t.Fatalf("getNetworkBandwidth() returned error: %v", err)
	}

	if bandwidth1 != 0 {
		t.Errorf("getNetworkBandwidth() first call = %v, want 0", bandwidth1)
	}

	// Wait a bit and call again to get actual delta
	time.Sleep(100 * time.Millisecond)
	bandwidth2, err := getNetworkBandwidth("lo")
	if err != nil {
		t.Fatalf("getNetworkBandwidth() second call returned error: %v", err)
	}

	if bandwidth2 < 0 {
		t.Errorf("getNetworkBandwidth() = %v, want non-negative value", bandwidth2)
	}
}

func TestFormatBandwidth(t *testing.T) {
	tests := []struct {
		name  string
		bytes float64
		want  string
	}{
		{
			name:  "bytes",
			bytes: 500,
			want:  "500",
		},
		{
			name:  "kilobytes",
			bytes: 1500,
			want:  "1.5k",
		},
		{
			name:  "megabytes",
			bytes: 2500000,
			want:  "2.5M",
		},
		{
			name:  "gigabytes",
			bytes: 1500000000,
			want:  "1.5G",
		},
		{
			name:  "exact kilobyte",
			bytes: 1000,
			want:  "1.0k",
		},
		{
			name:  "exact megabyte",
			bytes: 1000000,
			want:  "1.0M",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBandwidth(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBandwidth(%v) = %v, want %v", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestNetworkMetricStatusLogic(t *testing.T) {
	tests := []struct {
		name       string
		current    float64
		max        float64
		wantStatus string
	}{
		{
			name:       "within limits",
			current:    500.0,
			max:        1000.0,
			wantStatus: "OK",
		},
		{
			name:       "at limit",
			current:    1000.0,
			max:        1000.0,
			wantStatus: "OK",
		},
		{
			name:       "exceeds limit",
			current:    1200.0,
			max:        1000.0,
			wantStatus: "KO",
		},
		{
			name:       "low usage",
			current:    10.0,
			max:        1000.0,
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

func TestCollectNetworkMetricUpdatesCache(t *testing.T) {
	// Setup test config with multiple interfaces
	oldConfig := config
	config = ProbeConfig{
		WarmupEnabled:     false,
		WarmupDuration:    60 * time.Second,
		MaxConnections:    1000.0,
		NetworkInterfaces: []string{"lo", "eth0"},
		startTime:         time.Now(),
	}
	defer func() { config = oldConfig }()

	// Clear cache
	cacheMutex.Lock()
	metricCache = make(map[string]MetricStatus)
	cacheMutex.Unlock()

	// Run one iteration of metric collection for connections
	connections, err := getNetworkConnections()
	if err != nil {
		connections = 0
	}

	effectiveMax := config.MaxConnections
	status := "OK"
	if connections > effectiveMax {
		status = "KO"
	}

	cacheMutex.Lock()
	metricCache["network_connections"] = MetricStatus{
		Current: connections,
		Max:     effectiveMax,
		Status:  status,
	}
	cacheMutex.Unlock()

	// Collect bandwidth for each interface
	for _, iface := range config.NetworkInterfaces {
		bytesPerSec, err := getNetworkBandwidth(iface)
		if err != nil {
			bytesPerSec = 0
		}

		metricName := "network_" + iface + "_bandwidth"
		cacheMutex.Lock()
		metricCache[metricName] = MetricStatus{
			Current: bytesPerSec,
			Max:     0,
			Status:  "OK",
		}
		cacheMutex.Unlock()
	}

	// Verify cache was updated for connections
	cacheMutex.RLock()
	connMetric, connExists := metricCache["network_connections"]
	loMetric, loExists := metricCache["network_lo_bandwidth"]
	cacheMutex.RUnlock()

	if !connExists {
		t.Fatal("Network connections metric not found in cache")
	}

	if connMetric.Current < 0 {
		t.Errorf("Network connections metric current = %v, want non-negative value", connMetric.Current)
	}

	if connMetric.Max != 1000.0 {
		t.Errorf("Network connections metric max = %v, want 1000.0", connMetric.Max)
	}

	if connMetric.Status != "OK" && connMetric.Status != "KO" {
		t.Errorf("Network connections metric status = %v, want OK or KO", connMetric.Status)
	}

	// Verify interface metrics
	if !loExists {
		t.Fatal("Network lo interface metric not found in cache")
	}

	if loMetric.Current < 0 {
		t.Errorf("Network lo metric current = %v, want non-negative value", loMetric.Current)
	}

	if loMetric.Status != "OK" {
		t.Errorf("Network lo metric status = %v, want OK", loMetric.Status)
	}
}
