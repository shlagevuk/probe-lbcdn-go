package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

// getMemoryUsage reads memory usage from /proc/meminfo
// Returns percentage of memory used
func getMemoryUsage() (float64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	var memTotal, memAvailable uint64
	lines := string(data)

	// Parse MemTotal and MemAvailable
	_, err = fmt.Sscanf(lines, "MemTotal: %d kB\nMemFree: %*d kB\nMemAvailable: %d kB", &memTotal, &memAvailable)
	if err != nil {
		// Try alternative parsing
		for i := 0; i < len(lines); i++ {
			if lines[i:i+9] == "MemTotal:" {
				fmt.Sscanf(lines[i:], "MemTotal: %d kB", &memTotal)
			}
			if i+13 <= len(lines) && lines[i:i+13] == "MemAvailable:" {
				fmt.Sscanf(lines[i:], "MemAvailable: %d kB", &memAvailable)
				break
			}
		}
	}

	if memTotal == 0 {
		return 0, fmt.Errorf("failed to parse memory information")
	}

	memUsed := memTotal - memAvailable
	memPercent := float64(memUsed) / float64(memTotal) * 100.0

	return memPercent, nil
}

// collectMemoryMetric runs as a goroutine to collect memory metrics
func collectMemoryMetric() {
	for {
		memUsage, err := getMemoryUsage()
		if err != nil {
			log.Printf("Error collecting memory metric: %v", err)
			memUsage = 0
		}

		// Apply warmup factor if enabled
		effectiveMax := config.Thresholds.MaxMemory
		if config.Warmup.Enabled {
			warmupFactor := getWarmupFactor()
			effectiveMax = config.Thresholds.MaxMemory * warmupFactor
		}

		// Determine status
		status := "OK"
		if memUsage > effectiveMax {
			status = "KO"
		}

		// Update cache
		cacheMutex.Lock()
		metricCache["memory"] = MetricStatus{
			Current: memUsage,
			Max:     effectiveMax,
			Status:  status,
		}
		cacheMutex.Unlock()

		time.Sleep(2 * time.Second)
	}
}
