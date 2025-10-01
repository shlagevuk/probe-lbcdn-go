# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

### Building and Running
- `make build` - Build the probe binary to build/probe-lbcdn
- `make run` - Build and run the probe locally
- `make install` - Install binary to $GOPATH/bin

### Testing and Code Quality
- `make test` - Run all tests with verbose output
- `make fmt` - Format Go code using go fmt
- `make vet` - Run go vet for static analysis
- `make lint` - Run both fmt and vet (formatters and linters)
- `make all` - Run complete pipeline: clean, lint, test, build

### Development Workflow
- `make dev` - Run in development mode with auto-restart (requires air)
- `make clean` - Remove build artifacts

### Usage Examples
- `./probe-lbcdn --generate-config` - Create default config file
- `./probe-lbcdn --debug --display` - Run with debug logging and terminal display
- `./probe-lbcdn --config custom.yaml` - Use custom configuration file

## Architecture Overview

This is a lightweight system health probe written in Go that monitors Unix-like systems and provides health status via JSON API.

### Core Components

**Main Application (main.go)**
- HTTP server with /health endpoint
- Concurrent metric collection using goroutines
- Global configuration and metric caching with mutex protection
- Warmup mode support for gradual threshold enforcement

**Metric Collectors** - Each runs in its own goroutine:
- `cpu_metric.go` - CPU usage, IO wait, IRQ, and SoftIRQ percentages from /proc/stat
- `memory_metric.go` - Memory usage from /proc/meminfo
- `disk_metric.go` - Disk space utilization for configured paths
- `network_metric.go` - Network connections and bandwidth monitoring

**Display System (display.go)**
- Terminal output similar to dstat
- Color-coded status indicators (red for KO, normal for OK)
- Real-time metrics display with configurable intervals

### Key Design Patterns

**Concurrent Architecture**: Each metric collector runs as an independent goroutine, allowing:
- Non-blocking metric collection
- Isolated failure handling per metric
- Concurrent data gathering for faster response times

**Thread-Safe Caching**: Global `metricCache` with `cacheMutex` for thread-safe access to metric data.

**Warmup Mode**: Gradual threshold ramping from 0% to 100% over configured duration to prevent false positives during startup.

## Configuration

### Configuration System
The application uses YAML configuration files with command-line overrides:

**Command Line Options:**
- `--config/-c <file>` - Specify config file (default: probe-config.yaml)
- `--generate-config` - Generate default config file and exit
- `--debug/-d` - Enable debug logging with file/line info
- `--display` - Enable terminal metrics display (disabled by default)
- `--help/-h` - Show help

**Configuration Flow:**
1. Load default configuration values
2. Override with YAML file settings (if file exists)
3. Apply command-line flag overrides

### Default Configuration Values
- **Server:** Port :8080
- **Warmup:** Enabled with 60-second duration
- **Thresholds:** CPU 80%, IOWait 20%, IRQ 5%, SoftIRQ 10%, Memory 90%, Disk 95%, Connections 1000
- **Monitoring:** Disk paths [/, /var, /tmp], Network interfaces [eth0, lo]
- **Logging:** File in executable directory, Debug disabled
- **Display:** Disabled by default, 3-second interval

### Configuration Files
- Generate default config: `./probe-lbcdn --generate-config`
- Config file structure organized into logical sections: server, warmup, thresholds, monitoring, logging, display

## Testing

All metric collectors have corresponding test files (*_test.go). Tests use Go's standard testing package.

### Testing Requirements
**CRITICAL: Always run tests after ANY Go code changes**
- Run `make test` immediately after modifying any `.go` file
- Ensure all tests pass before considering changes complete
- Update test files when changing configuration structures or function signatures
- Test failures indicate breaking changes that must be fixed before proceeding

### Test Coverage
- CPU metrics: Unit tests for metric collection and status logic
- Memory metrics: Memory usage calculation and threshold validation  
- Disk metrics: Disk usage monitoring and path sanitization
- Network metrics: Connection counting and bandwidth measurement

## Commit Guidelines

### Commit Message Format
Use conventional commit format for all changes:

```
<type>(<scope>): <description>

<body>

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

**Types:** feat, fix, docs, style, refactor, test, chore
**Scopes:** config, metrics, logging, display, api, build
**Examples:**
- `feat(config): add YAML configuration system with CLI options`
- `fix(metrics): update CPU metric collection to handle edge cases`  
- `test(network): add comprehensive bandwidth measurement tests`
- `docs(readme): update configuration documentation`

### Commit Requirements
- Always run `make test` before committing
- Include scope of changes in commit message
- Describe both what changed and why
- Use imperative mood ("add" not "added")

## API Response Format

The /health endpoint returns JSON with:
- Overall status: "OK" or "KO" 
- Timestamp of the check
- Individual metric details with current value, max threshold, and status
- HTTP 200 for OK, 503 for KO status