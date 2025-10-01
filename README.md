# probe-lbcdn-go

A lightweight health probe for Unix-like systems that monitors system resources and reports operational status via JSON API.

## Overview

This probe monitors various system metrics and compares them against configured maximum thresholds. It serves a simple JSON response indicating whether the system is operating within acceptable parameters:

- **OK**: All metrics are within their maximum thresholds
- **KO**: One or more metrics have exceeded their configured limits

## Features

- **Concurrent Metric Collection**: Each metric is gathered in its own goroutine for optimal performance
- **YAML Configuration**: Flexible configuration system with command-line overrides
- **Configurable Thresholds**: Set maximum values for each monitored resource
- **Warmup Mode**: Gradually ramp up threshold limits on startup instead of immediately enforcing full limits
- **Debug Logging**: Enhanced logging with file/line information for troubleshooting
- **Terminal Display**: Optional real-time metrics display with color-coded status
- **JSON API**: Simple HTTP endpoint returning health status
- **Unix-focused**: Designed for Linux and Unix-like operating systems

## Monitored Metrics

- **CPU usage** - User, system, IOWait, IRQ, and SoftIRQ percentages
- **Memory usage** - Available memory percentage
- **Disk space utilization** - Per-path disk usage monitoring
- **Network connections** - Active TCP connection count
- **Network bandwidth** - Per-interface traffic monitoring

## Warmup Mode

When enabled, warmup mode starts with thresholds at 0% and gradually increases them to 100% over a configured time period. This prevents false KO states during application startup or system recovery phases.

## Architecture

Each metric collector runs as an independent goroutine, allowing:
- Non-blocking metric collection
- Easy addition of new metrics
- Isolated failure handling per metric
- Concurrent data gathering for faster response times

## API Response Format

```json
{
  "status": "OK|KO",
  "timestamp": "2025-09-30T12:00:00Z",
  "metrics": {
    "cpu": {"current": 45.2, "max": 80.0, "status": "OK"},
    "memory": {"current": 62.5, "max": 90.0, "status": "OK"}
  }
}
```

## Quick Start

### 1. Build the probe
```bash
make build
```

### 2. Generate configuration file
```bash
./build/probe-lbcdn --generate-config
```

### 3. Run the probe
```bash
# With default settings
./build/probe-lbcdn

# With debug logging and terminal display
./build/probe-lbcdn --debug --display

# With custom configuration file
./build/probe-lbcdn --config myconfig.yaml
```

## Command Line Options

```bash
# Configuration
--config, -c <file>     Path to YAML configuration file (default: probe-config.yaml)
--generate-config       Generate default configuration file and exit

# Logging and Display  
--debug, -d             Enable debug logging with file/line information
--display               Enable terminal metrics display (disabled by default)

# Help
--help, -h              Show help and usage information
```

## Configuration

The probe uses a hierarchical configuration system:

1. **Default values** - Built-in sensible defaults
2. **YAML configuration file** - Override defaults with file settings
3. **Command-line flags** - Final overrides for specific options

### Configuration File Structure

Generate a default configuration file to see all available options:
```bash
./build/probe-lbcdn --generate-config
```

The configuration file is organized into sections:
- **server**: HTTP server settings (port)
- **warmup**: Warmup mode configuration (enabled, duration)
- **thresholds**: Maximum values for each metric (CPU, memory, disk, etc.)
- **monitoring**: Paths and interfaces to monitor
- **logging**: Log file location and debug mode
- **display**: Terminal display settings

## Development Status

This project is under active development.
