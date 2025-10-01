package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// bandwidthSnapshot stores previous bandwidth reading for delta calculation
type bandwidthSnapshot struct {
	totalBytes uint64
	timestamp  time.Time
}

var (
	bandwidthCache      = make(map[string]bandwidthSnapshot)
	bandwidthCacheMutex sync.Mutex
)

// getNetworkConnections reads the number of active network connections
// Returns count of established TCP connections
func getNetworkConnections() (float64, error) {
	// Read /proc/net/tcp for IPv4 connections
	data, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		return 0, err
	}

	// Count lines with state 01 (ESTABLISHED)
	// Format: sl  local_address rem_address   st ...
	lines := string(data)
	count := 0
	lineStart := 0

	for i := 0; i < len(lines); i++ {
		if lines[i] == '\n' {
			line := lines[lineStart:i]
			// Check if line contains " 01 " which indicates ESTABLISHED state
			for j := 0; j < len(line)-3; j++ {
				if line[j:j+4] == " 01 " {
					count++
					break
				}
			}
			lineStart = i + 1
		}
	}

	// Also check IPv6 connections
	data6, err := os.ReadFile("/proc/net/tcp6")
	if err == nil {
		lines6 := string(data6)
		lineStart = 0
		for i := 0; i < len(lines6); i++ {
			if lines6[i] == '\n' {
				line := lines6[lineStart:i]
				for j := 0; j < len(line)-3; j++ {
					if line[j:j+4] == " 01 " {
						count++
						break
					}
				}
				lineStart = i + 1
			}
		}
	}

	return float64(count), nil
}

// collectNetworkMetric runs as a goroutine to collect network metrics
// Monitors all interfaces specified in config.NetworkInterfaces against config.MaxConnections
func collectNetworkMetric() {
	for {
		// First collect global connection count
		connections, err := getNetworkConnections()
		if err != nil {
			log.Printf("Error collecting network connections: %v", err)
			connections = 0
		}

		// Apply warmup factor if enabled
		effectiveMax := config.Thresholds.MaxConnections
		if config.Warmup.Enabled {
			warmupFactor := getWarmupFactor()
			effectiveMax = config.Thresholds.MaxConnections * warmupFactor
		}

		// Determine status for connections
		status := "OK"
		if connections > effectiveMax {
			status = "KO"
		}

		// Update cache for global connections
		cacheMutex.Lock()
		metricCache["network_connections"] = MetricStatus{
			Current: connections,
			Max:     effectiveMax,
			Status:  status,
		}
		cacheMutex.Unlock()

		// Check each network interface for traffic
		for _, iface := range config.Monitoring.NetworkInterfaces {
			bytesPerSec, err := getNetworkBandwidth(iface)
			if err != nil {
				log.Printf("Error collecting network bandwidth for %s: %v", iface, err)
				bytesPerSec = 0
			}

			// For bandwidth, we report bytes/sec
			// Status is always OK unless we add a bandwidth threshold later
			metricName := fmt.Sprintf("network_%s_bandwidth", iface)
			cacheMutex.Lock()
			metricCache[metricName] = MetricStatus{
				Current: bytesPerSec,
				Max:     0, // No max threshold for bandwidth yet
				Status:  "OK",
			}
			cacheMutex.Unlock()
		}

		time.Sleep(2 * time.Second)
	}
}

// formatBandwidth converts bytes/sec to human-readable ISO format
// Examples: 1000 -> "1k", 1000000 -> "1M", 1000000000 -> "1G"
func formatBandwidth(bytesPerSec float64) string {
	const unit = 1000.0 // ISO standard uses 1000 (not 1024)

	if bytesPerSec < unit {
		return fmt.Sprintf("%.0f", bytesPerSec)
	}

	div := unit
	suffixes := []string{"k", "M", "G", "T", "P"}

	for i, suffix := range suffixes {
		if bytesPerSec < div*unit || i == len(suffixes)-1 {
			return fmt.Sprintf("%.1f%s", bytesPerSec/div, suffix)
		}
		div *= unit
	}

	return fmt.Sprintf("%.0f", bytesPerSec)
}

// getNetworkBandwidth reads network bandwidth usage from /proc/net/dev
// Returns bytes/sec for the specified interface by calculating delta from last reading
func getNetworkBandwidth(iface string) (float64, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, err
	}

	lines := string(data)
	var rxBytes, txBytes uint64

	// Find the interface line
	lineStart := 0
	for i := 0; i < len(lines); i++ {
		if lines[i] == '\n' {
			line := lines[lineStart:i]
			// Check if line starts with interface name
			if len(line) > len(iface) && line[:len(iface)+1] == iface+":" {
				// Parse bytes received and transmitted
				_, err := fmt.Sscanf(line[len(iface)+1:], "%d %*d %*d %*d %*d %*d %*d %*d %d", &rxBytes, &txBytes)
				if err != nil {
					return 0, err
				}
				break
			}
			lineStart = i + 1
		}
	}

	currentTotalBytes := rxBytes + txBytes
	currentTime := time.Now()

	// Calculate bytes/sec using delta
	bandwidthCacheMutex.Lock()
	defer bandwidthCacheMutex.Unlock()

	snapshot, exists := bandwidthCache[iface]
	if !exists {
		// First reading, store it and return 0
		bandwidthCache[iface] = bandwidthSnapshot{
			totalBytes: currentTotalBytes,
			timestamp:  currentTime,
		}
		return 0, nil
	}

	// Calculate delta
	bytesDelta := currentTotalBytes - snapshot.totalBytes
	timeDelta := currentTime.Sub(snapshot.timestamp).Seconds()

	// Update cache with current reading
	bandwidthCache[iface] = bandwidthSnapshot{
		totalBytes: currentTotalBytes,
		timestamp:  currentTime,
	}

	if timeDelta <= 0 {
		return 0, nil
	}

	bytesPerSec := float64(bytesDelta) / timeDelta
	return bytesPerSec, nil
}
