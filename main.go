package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MetricStatus represents the status of a single metric
type MetricStatus struct {
	Current float64 `json:"current"`
	Max     float64 `json:"max"`
	Status  string  `json:"status"`
}

// HealthResponse represents the JSON response structure
type HealthResponse struct {
	Status    string                  `json:"status"`
	Timestamp time.Time               `json:"timestamp"`
	Metrics   map[string]MetricStatus `json:"metrics"`
}

// ProbeConfig holds configuration for the probe
type ProbeConfig struct {
	WarmupEnabled      bool
	WarmupDuration     time.Duration
	MaxCPU             float64
	MaxIOWait          float64
	MaxIRQ             float64
	MaxSoftIRQ         float64
	MaxMemory          float64
	MaxDisk            float64
	MaxConnections     float64
	DiskPaths          []string
	NetworkInterfaces  []string
	Port               string
	LogFile            string
	DisplayEnabled     bool
	DisplayInterval    time.Duration
	startTime          time.Time
}

var (
	config      ProbeConfig
	metricCache = make(map[string]MetricStatus)
	cacheMutex  sync.RWMutex
)

// healthHandler handles the /health endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	cacheMutex.RLock()
	metrics := make(map[string]MetricStatus)
	for k, v := range metricCache {
		metrics[k] = v
	}
	cacheMutex.RUnlock()

	// Determine overall status
	overallStatus := "OK"
	for _, metric := range metrics {
		if metric.Status == "KO" {
			overallStatus = "KO"
			break
		}
	}

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Metrics:   metrics,
	}

	w.Header().Set("Content-Type", "application/json")
	if overallStatus == "KO" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(response)
}

func main() {
	// Get executable directory for default log file
	exePath, err := os.Executable()
	if err != nil {
		exePath = "."
	}
	exeDir := filepath.Dir(exePath)
	defaultLogFile := filepath.Join(exeDir, "probe.log")

	// Initialize configuration
	config = ProbeConfig{
		WarmupEnabled:     true,
		WarmupDuration:    60 * time.Second,
		MaxCPU:            80.0,
		MaxIOWait:         20.0,
		MaxIRQ:            5.0,
		MaxSoftIRQ:        10.0,
		MaxMemory:         90.0,
		MaxDisk:           95.0,
		MaxConnections:    1000.0,
		DiskPaths:         []string{"/", "/var", "/tmp"},
		NetworkInterfaces: []string{"eth0", "lo"},
		Port:              ":8080",
		LogFile:           defaultLogFile,
		DisplayEnabled:    true,
		DisplayInterval:   3 * time.Second,
		startTime:         time.Now(),
	}

	// Setup logging to file
	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Warning: Failed to open log file %s: %v. Logging to stderr only.", config.LogFile, err)
	} else {
		// Log to both file and stderr
		multiWriter := io.MultiWriter(os.Stderr, logFile)
		log.SetOutput(multiWriter)
		defer logFile.Close()
	}

	log.Printf("Starting probe with config: warmup=%v, duration=%v",
		config.WarmupEnabled, config.WarmupDuration)
	log.Printf("CPU Thresholds: Usage=%.1f%%, IOWait=%.1f%%, IRQ=%.1f%%, SoftIRQ=%.1f%%",
		config.MaxCPU, config.MaxIOWait, config.MaxIRQ, config.MaxSoftIRQ)
	log.Printf("Other Thresholds: Memory=%.1f%%, Disk=%.1f%%, Connections=%.0f",
		config.MaxMemory, config.MaxDisk, config.MaxConnections)
	log.Printf("Monitoring disk paths: %v", config.DiskPaths)
	log.Printf("Monitoring network interfaces: %v", config.NetworkInterfaces)
	log.Printf("Logging to: %s", config.LogFile)

	// Start metric collection goroutines
	go collectCPUMetric()
	go collectMemoryMetric()
	go collectDiskMetric()
	go collectNetworkMetric()

	// Start display if enabled
	if config.DisplayEnabled {
		log.Printf("Starting metrics display (interval: %v)", config.DisplayInterval)
		go displayMetrics()
	}

	// Setup HTTP handlers
	http.HandleFunc("/health", healthHandler)

	// Start HTTP server
	log.Printf("Probe listening on %s", config.Port)
	if err := http.ListenAndServe(config.Port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
