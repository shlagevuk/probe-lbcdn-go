package main

import (
	"fmt"
	"log"
	"syscall"
	"time"
)

// getDiskUsage reads disk usage for a given path using statfs
// Returns percentage of disk space used
func getDiskUsage(path string) (float64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}

	// Calculate disk usage
	total := stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	used := total - available

	diskPercent := float64(used) / float64(total) * 100.0

	return diskPercent, nil
}

// collectDiskMetric runs as a goroutine to collect disk metrics
// Monitors all paths specified in config.DiskPaths against config.MaxDisk
func collectDiskMetric() {
	for {
		// Apply warmup factor if enabled
		effectiveMax := config.Thresholds.MaxDisk
		if config.Warmup.Enabled {
			warmupFactor := getWarmupFactor()
			effectiveMax = config.Thresholds.MaxDisk * warmupFactor
		}

		// Check each disk path
		for _, path := range config.Monitoring.DiskPaths {
			diskUsage, err := getDiskUsage(path)
			if err != nil {
				log.Printf("Error collecting disk metric for %s: %v", path, err)
				diskUsage = 0
			}

			// Determine status
			status := "OK"
			if diskUsage > effectiveMax {
				status = "KO"
			}

			// Update cache with path-specific metric name
			metricName := fmt.Sprintf("disk_%s", sanitizePath(path))
			cacheMutex.Lock()
			metricCache[metricName] = MetricStatus{
				Current: diskUsage,
				Max:     effectiveMax,
				Status:  status,
			}
			cacheMutex.Unlock()
		}

		time.Sleep(5 * time.Second) // Check disk less frequently
	}
}

// sanitizePath converts a filesystem path to a metric-friendly name
// e.g., "/" -> "root", "/var/log" -> "var_log"
func sanitizePath(path string) string {
	if path == "/" {
		return "root"
	}
	// Remove leading slash and replace remaining slashes with underscores
	sanitized := path[1:]
	result := ""
	for _, ch := range sanitized {
		if ch == '/' {
			result += "_"
		} else {
			result += string(ch)
		}
	}
	return result
}
