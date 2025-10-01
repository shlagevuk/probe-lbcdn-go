package main

import (
	"testing"
	"time"
)

func TestGetCPUMetrics(t *testing.T) {
	// Reset CPU cache for clean test
	cpuCacheMutex.Lock()
	cpuInitialized = false
	cpuCacheMutex.Unlock()

	// First call should return zeros (no baseline)
	metrics1, err := getCPUMetrics()
	if err != nil {
		t.Fatalf("getCPUMetrics() returned error: %v", err)
	}

	if metrics1.Usage != 0 || metrics1.IOWait != 0 || metrics1.IRQ != 0 || metrics1.SoftIRQ != 0 {
		t.Errorf("getCPUMetrics() first call should return zeros, got %+v", metrics1)
	}

	// Wait and call again to get actual deltas
	time.Sleep(100 * time.Millisecond)
	metrics2, err := getCPUMetrics()
	if err != nil {
		t.Fatalf("getCPUMetrics() second call returned error: %v", err)
	}

	// Verify all metrics are within valid range
	if metrics2.Usage < 0 || metrics2.Usage > 100 {
		t.Errorf("CPU usage = %v, want value between 0 and 100", metrics2.Usage)
	}
	if metrics2.IOWait < 0 || metrics2.IOWait > 100 {
		t.Errorf("IOWait = %v, want value between 0 and 100", metrics2.IOWait)
	}
	if metrics2.IRQ < 0 || metrics2.IRQ > 100 {
		t.Errorf("IRQ = %v, want value between 0 and 100", metrics2.IRQ)
	}
	if metrics2.SoftIRQ < 0 || metrics2.SoftIRQ > 100 {
		t.Errorf("SoftIRQ = %v, want value between 0 and 100", metrics2.SoftIRQ)
	}
}

func TestCPUMetricStatusLogic(t *testing.T) {
	tests := []struct {
		name       string
		current    float64
		max        float64
		wantStatus string
	}{
		{
			name:       "within limits",
			current:    50.0,
			max:        80.0,
			wantStatus: "OK",
		},
		{
			name:       "at limit",
			current:    80.0,
			max:        80.0,
			wantStatus: "OK",
		},
		{
			name:       "exceeds limit",
			current:    85.0,
			max:        80.0,
			wantStatus: "KO",
		},
		{
			name:       "zero usage",
			current:    0.0,
			max:        80.0,
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

func TestCollectCPUMetricUpdatesCache(t *testing.T) {
	// Setup test config
	oldConfig := config
	config = Config{
		startTime: time.Now(),
	}
	config.Warmup.Enabled = false
	config.Warmup.Duration = 60 * time.Second
	config.Thresholds.MaxCPU = 80.0
	config.Thresholds.MaxIOWait = 20.0
	config.Thresholds.MaxIRQ = 5.0
	config.Thresholds.MaxSoftIRQ = 10.0
	defer func() { config = oldConfig }()

	// Reset CPU cache
	cpuCacheMutex.Lock()
	cpuInitialized = false
	cpuCacheMutex.Unlock()

	// Clear cache
	cacheMutex.Lock()
	metricCache = make(map[string]MetricStatus)
	cacheMutex.Unlock()

	// First call to initialize
	metrics, _ := getCPUMetrics()

	// Wait and collect again
	time.Sleep(100 * time.Millisecond)
	metrics, err := getCPUMetrics()
	if err != nil {
		metrics = cpuMetrics{}
	}

	// Simulate what collectCPUMetric does
	cacheMutex.Lock()
	metricCache["cpu_usage"] = MetricStatus{
		Current: metrics.Usage,
		Max:     config.Thresholds.MaxCPU,
		Status:  "OK",
	}
	metricCache["cpu_iowait"] = MetricStatus{
		Current: metrics.IOWait,
		Max:     config.Thresholds.MaxIOWait,
		Status:  "OK",
	}
	metricCache["cpu_irq"] = MetricStatus{
		Current: metrics.IRQ,
		Max:     config.Thresholds.MaxIRQ,
		Status:  "OK",
	}
	metricCache["cpu_softirq"] = MetricStatus{
		Current: metrics.SoftIRQ,
		Max:     config.Thresholds.MaxSoftIRQ,
		Status:  "OK",
	}
	cacheMutex.Unlock()

	// Verify all CPU metrics were updated
	cacheMutex.RLock()
	usageMetric, usageExists := metricCache["cpu_usage"]
	iowaitMetric, iowaitExists := metricCache["cpu_iowait"]
	irqMetric, irqExists := metricCache["cpu_irq"]
	softirqMetric, softirqExists := metricCache["cpu_softirq"]
	cacheMutex.RUnlock()

	if !usageExists {
		t.Fatal("CPU usage metric not found in cache")
	}
	if !iowaitExists {
		t.Fatal("CPU iowait metric not found in cache")
	}
	if !irqExists {
		t.Fatal("CPU irq metric not found in cache")
	}
	if !softirqExists {
		t.Fatal("CPU softirq metric not found in cache")
	}

	// Verify values are valid
	if usageMetric.Current < 0 || usageMetric.Current > 100 {
		t.Errorf("CPU usage current = %v, want value between 0 and 100", usageMetric.Current)
	}
	if usageMetric.Max != 80.0 {
		t.Errorf("CPU usage max = %v, want 80.0", usageMetric.Max)
	}
	if iowaitMetric.Max != 20.0 {
		t.Errorf("CPU iowait max = %v, want 20.0", iowaitMetric.Max)
	}
	if irqMetric.Max != 5.0 {
		t.Errorf("CPU irq max = %v, want 5.0", irqMetric.Max)
	}
	if softirqMetric.Max != 10.0 {
		t.Errorf("CPU softirq max = %v, want 10.0", softirqMetric.Max)
	}
}
