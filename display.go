package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// displayMetrics shows a dstat-like terminal output of current metrics
func displayMetrics(config Config) {
	// Print header
	printHeader()

	ticker := time.NewTicker(config.Display.Interval)
	defer ticker.Stop()

	lineCount := 0
	for range ticker.C {
		// Reprint header every 20 lines
		if lineCount%20 == 0 && lineCount > 0 {
			fmt.Println()
			printHeader()
		}

		cacheMutex.RLock()
		metrics := make(map[string]MetricStatus)
		for k, v := range metricCache {
			metrics[k] = v
		}
		cacheMutex.RUnlock()

		printMetricLine(metrics)
		lineCount++
	}
}

// printHeader prints the column headers
func printHeader() {
	fmt.Println(strings.Repeat("-", 110))
	fmt.Printf("%-8s | %-6s %-6s %-6s %-6s | %-6s | %-10s | %-6s | %-15s\n",
		"TIME", "CPU%", "IOWT%", "IRQ%", "SIRQ%", "MEM%", "DISK%", "CONN", "NET(B/s)")
	fmt.Println(strings.Repeat("-", 110))
}

// printMetricLine prints a single line of metrics
func printMetricLine(metrics map[string]MetricStatus) {
	timestamp := time.Now().Format("15:04:05")

	// Get CPU metrics
	cpuUsage := getMetricValue(metrics, "cpu_usage")
	iowait := getMetricValue(metrics, "cpu_iowait")
	irq := getMetricValue(metrics, "cpu_irq")
	softirq := getMetricValue(metrics, "cpu_softirq")

	// Get memory metric
	memory := getMetricValue(metrics, "memory")

	// Get average disk usage across all monitored paths
	diskAvg := getAverageDiskUsage(metrics)

	// Get network connections
	connections := getMetricValue(metrics, "network_connections")

	// Get total network bandwidth across all interfaces
	totalBandwidth := getTotalBandwidth(metrics)

	// Print the line with color coding for status
	fmt.Printf("%-8s | ", timestamp)
	printColoredValue(cpuUsage, getMetricStatus(metrics, "cpu_usage"), 6)
	fmt.Print(" ")
	printColoredValue(iowait, getMetricStatus(metrics, "cpu_iowait"), 6)
	fmt.Print(" ")
	printColoredValue(irq, getMetricStatus(metrics, "cpu_irq"), 6)
	fmt.Print(" ")
	printColoredValue(softirq, getMetricStatus(metrics, "cpu_softirq"), 6)
	fmt.Print(" | ")
	printColoredValue(memory, getMetricStatus(metrics, "memory"), 6)
	fmt.Print(" | ")
	fmt.Printf("%-10s", fmt.Sprintf("%.1f%%", diskAvg))
	fmt.Print(" | ")
	fmt.Printf("%-6.0f", connections)
	fmt.Print(" | ")
	fmt.Printf("%-15s", formatBandwidth(totalBandwidth))
	fmt.Println()
}

// getMetricValue safely retrieves a metric value
func getMetricValue(metrics map[string]MetricStatus, key string) float64 {
	if metric, exists := metrics[key]; exists {
		return metric.Current
	}
	return 0.0
}

// getMetricStatus safely retrieves a metric status
func getMetricStatus(metrics map[string]MetricStatus, key string) string {
	if metric, exists := metrics[key]; exists {
		return metric.Status
	}
	return "OK"
}

// getAverageDiskUsage calculates average disk usage across all monitored paths
func getAverageDiskUsage(metrics map[string]MetricStatus) float64 {
	total := 0.0
	count := 0
	for key, metric := range metrics {
		if strings.HasPrefix(key, "disk_") {
			total += metric.Current
			count++
		}
	}
	if count == 0 {
		return 0.0
	}
	return total / float64(count)
}

// getTotalBandwidth sums bandwidth across all network interfaces
func getTotalBandwidth(metrics map[string]MetricStatus) float64 {
	total := 0.0
	for key, metric := range metrics {
		if strings.HasPrefix(key, "network_") && strings.HasSuffix(key, "_bandwidth") {
			total += metric.Current
		}
	}
	return total
}

// printColoredValue prints a value with color based on status
func printColoredValue(value float64, status string, width int) {
	valueStr := fmt.Sprintf("%*.1f", width, value)

	// ANSI color codes
	if status == "KO" {
		// Red for KO
		fmt.Printf("\033[31m%s\033[0m", valueStr)
	} else {
		// Normal color for OK
		fmt.Print(valueStr)
	}
}

// clearScreen clears the terminal screen
func clearScreen() {
	fmt.Print("\033[H\033[2J")
	os.Stdout.Sync()
}
