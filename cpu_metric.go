package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// cpuMetrics holds detailed CPU usage percentages
type cpuMetrics struct {
	Usage   float64 // Overall CPU usage (user + nice + system)
	IOWait  float64 // IO wait percentage
	IRQ     float64 // Hardware interrupt percentage
	SoftIRQ float64 // Software interrupt percentage
}

// cpuSnapshot stores previous CPU readings for delta calculation
type cpuSnapshot struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	steal   uint64
}

var (
	cpuCache      cpuSnapshot
	cpuCacheMutex sync.Mutex
	cpuInitialized bool
)

// getCPUMetrics reads detailed CPU metrics from /proc/stat
// Returns percentages for usage, iowait, irq, and softirq
func getCPUMetrics() (cpuMetrics, error) {
	// Read /proc/stat for CPU metrics
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuMetrics{}, err
	}

	// Parse first line (cpu total)
	var user, nice, system, idle, iowait, irq, softirq, steal uint64
	_, err = fmt.Sscanf(string(data), "cpu %d %d %d %d %d %d %d %d",
		&user, &nice, &system, &idle, &iowait, &irq, &softirq, &steal)
	if err != nil {
		return cpuMetrics{}, err
	}

	cpuCacheMutex.Lock()
	defer cpuCacheMutex.Unlock()

	// First reading, initialize cache
	if !cpuInitialized {
		cpuCache = cpuSnapshot{user, nice, system, idle, iowait, irq, softirq, steal}
		cpuInitialized = true
		return cpuMetrics{}, nil // Return zeros on first call
	}

	// Calculate deltas
	userDelta := user - cpuCache.user
	niceDelta := nice - cpuCache.nice
	systemDelta := system - cpuCache.system
	idleDelta := idle - cpuCache.idle
	iowaitDelta := iowait - cpuCache.iowait
	irqDelta := irq - cpuCache.irq
	softirqDelta := softirq - cpuCache.softirq
	stealDelta := steal - cpuCache.steal

	totalDelta := userDelta + niceDelta + systemDelta + idleDelta + iowaitDelta + irqDelta + softirqDelta + stealDelta

	// Update cache
	cpuCache = cpuSnapshot{user, nice, system, idle, iowait, irq, softirq, steal}

	if totalDelta == 0 {
		return cpuMetrics{}, nil
	}

	// Calculate percentages
	metrics := cpuMetrics{
		Usage:   float64(userDelta+niceDelta+systemDelta) / float64(totalDelta) * 100.0,
		IOWait:  float64(iowaitDelta) / float64(totalDelta) * 100.0,
		IRQ:     float64(irqDelta) / float64(totalDelta) * 100.0,
		SoftIRQ: float64(softirqDelta) / float64(totalDelta) * 100.0,
	}

	return metrics, nil
}

// collectCPUMetric runs as a goroutine to collect CPU metrics
func collectCPUMetric() {
	for {
		metrics, err := getCPUMetrics()
		if err != nil {
			log.Printf("Error collecting CPU metrics: %v", err)
			metrics = cpuMetrics{}
		}

		// Apply warmup factor if enabled
		warmupFactor := 1.0
		if config.Warmup.Enabled {
			warmupFactor = getWarmupFactor()
		}

		// Calculate effective max values with warmup
		effectiveMaxCPU := config.Thresholds.MaxCPU * warmupFactor
		effectiveMaxIOWait := config.Thresholds.MaxIOWait * warmupFactor
		effectiveMaxIRQ := config.Thresholds.MaxIRQ * warmupFactor
		effectiveMaxSoftIRQ := config.Thresholds.MaxSoftIRQ * warmupFactor

		// Determine status for each metric
		cpuStatus := "OK"
		if metrics.Usage > effectiveMaxCPU {
			cpuStatus = "KO"
		}

		iowaitStatus := "OK"
		if metrics.IOWait > effectiveMaxIOWait {
			iowaitStatus = "KO"
		}

		irqStatus := "OK"
		if metrics.IRQ > effectiveMaxIRQ {
			irqStatus = "KO"
		}

		softirqStatus := "OK"
		if metrics.SoftIRQ > effectiveMaxSoftIRQ {
			softirqStatus = "KO"
		}

		// Update cache for all CPU metrics
		cacheMutex.Lock()
		metricCache["cpu_usage"] = MetricStatus{
			Current: metrics.Usage,
			Max:     effectiveMaxCPU,
			Status:  cpuStatus,
		}
		metricCache["cpu_iowait"] = MetricStatus{
			Current: metrics.IOWait,
			Max:     effectiveMaxIOWait,
			Status:  iowaitStatus,
		}
		metricCache["cpu_irq"] = MetricStatus{
			Current: metrics.IRQ,
			Max:     effectiveMaxIRQ,
			Status:  irqStatus,
		}
		metricCache["cpu_softirq"] = MetricStatus{
			Current: metrics.SoftIRQ,
			Max:     effectiveMaxSoftIRQ,
			Status:  softirqStatus,
		}
		cacheMutex.Unlock()

		time.Sleep(2 * time.Second)
	}
}

// getWarmupFactor returns a factor between 0.0 and 1.0 based on elapsed time
func getWarmupFactor() float64 {
	elapsed := time.Since(config.startTime)
	if elapsed >= config.Warmup.Duration {
		return 1.0
	}
	return elapsed.Seconds() / config.Warmup.Duration.Seconds()
}
