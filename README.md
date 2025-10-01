# probe-lbcdn-go

A lightweight health probe for Unix-like systems that monitors system resources and reports operational status via JSON API.

## Overview

This probe monitors various system metrics and compares them against configured maximum thresholds. It serves a simple JSON response indicating whether the system is operating within acceptable parameters:

- **OK**: All metrics are within their maximum thresholds
- **KO**: One or more metrics have exceeded their configured limits

## Features

- **Concurrent Metric Collection**: Each metric is gathered in its own goroutine for optimal performance
- **Configurable Thresholds**: Set maximum values for each monitored resource
- **Warmup Mode**: Gradually ramp up threshold limits on startup instead of immediately enforcing full limits
- **JSON API**: Simple HTTP endpoint returning health status
- **Unix-focused**: Designed for Linux and Unix-like operating systems

## Planned Metrics

- CPU usage
- Memory usage
- Disk I/O
- Network connections
- Load average
- Disk space utilization

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

## Usage

```bash
# Start probe with default configuration
./probe-lbcdn-go

# Start with warmup enabled (300 second ramp-up)
./probe-lbcdn-go --warmup --warmup-duration=300

# Configure custom thresholds
./probe-lbcdn-go --max-cpu=80 --max-memory=90 --max-disk=95
```

## Configuration

Configuration can be provided via:
- Command-line flags
- Environment variables
- Configuration file (YAML/JSON)

## Development Status

This project is under active development.
