package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
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

var (
	config      Config
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
	// Parse command line flags
	flags := parseCommandLineFlags()

	// Show help and exit
	if flags.Help {
		showHelp()
		os.Exit(0)
	}

	// Generate config file and exit
	if flags.GenerateConfig {
		if err := generateConfigFile(flags.ConfigFile); err != nil {
			log.Fatalf("Failed to generate config file: %v", err)
		}
		os.Exit(0)
	}

	// Load configuration
	var err error
	config, err = loadConfig(flags)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	logFile, err := setupLogging(config)
	if err != nil {
		log.Printf("Warning: Failed to setup logging: %v", err)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	logInfo("Starting probe with config: warmup=%v, duration=%v",
		config.Warmup.Enabled, config.Warmup.Duration)
	logInfo("CPU Thresholds: Usage=%.1f%%, IOWait=%.1f%%, IRQ=%.1f%%, SoftIRQ=%.1f%%",
		config.Thresholds.MaxCPU, config.Thresholds.MaxIOWait, config.Thresholds.MaxIRQ, config.Thresholds.MaxSoftIRQ)
	logInfo("Other Thresholds: Memory=%.1f%%, Disk=%.1f%%, Connections=%.0f",
		config.Thresholds.MaxMemory, config.Thresholds.MaxDisk, config.Thresholds.MaxConnections)
	logInfo("Monitoring disk paths: %v", config.Monitoring.DiskPaths)
	logInfo("Monitoring network interfaces: %v", config.Monitoring.NetworkInterfaces)
	logInfo("Logging to: %s", config.Logging.File)
	logDebug(config, "Debug logging enabled")

	// Start metric collection goroutines
	go collectCPUMetric()
	go collectMemoryMetric()
	go collectDiskMetric()
	go collectNetworkMetric()

	// Start display if enabled
	if config.Display.Enabled {
		logInfo("Starting metrics display (interval: %v)", config.Display.Interval)
		go displayMetrics(config)
	}

	// Setup HTTP handlers
	http.HandleFunc("/health", healthHandler)

	// Start HTTP server
	logInfo("Probe listening on %s", config.Server.Port)
	if err := http.ListenAndServe(config.Server.Port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
