package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the probe
type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`

	Warmup struct {
		Enabled  bool          `yaml:"enabled"`
		Duration time.Duration `yaml:"duration"`
	} `yaml:"warmup"`

	Thresholds struct {
		MaxCPU         float64 `yaml:"max_cpu"`
		MaxIOWait      float64 `yaml:"max_iowait"`
		MaxIRQ         float64 `yaml:"max_irq"`
		MaxSoftIRQ     float64 `yaml:"max_softirq"`
		MaxMemory      float64 `yaml:"max_memory"`
		MaxDisk        float64 `yaml:"max_disk"`
		MaxConnections float64 `yaml:"max_connections"`
	} `yaml:"thresholds"`

	Monitoring struct {
		DiskPaths         []string `yaml:"disk_paths"`
		NetworkInterfaces []string `yaml:"network_interfaces"`
	} `yaml:"monitoring"`

	Logging struct {
		File  string `yaml:"file"`
		Debug bool   `yaml:"debug"`
	} `yaml:"logging"`

	Display struct {
		Enabled  bool          `yaml:"enabled"`
		Interval time.Duration `yaml:"interval"`
	} `yaml:"display"`

	// Runtime fields (not in YAML)
	startTime time.Time `yaml:"-"`
}

// CommandLineFlags holds parsed command line arguments
type CommandLineFlags struct {
	ConfigFile     string
	GenerateConfig bool
	Debug          bool
	Display        bool
	Help           bool
}

// getDefaultConfig returns a configuration with default values
func getDefaultConfig() Config {
	exePath, err := os.Executable()
	if err != nil {
		exePath = "."
	}
	exeDir := filepath.Dir(exePath)
	defaultLogFile := filepath.Join(exeDir, "probe.log")

	config := Config{
		startTime: time.Now(),
	}

	config.Server.Port = ":8080"

	config.Warmup.Enabled = true
	config.Warmup.Duration = 60 * time.Second

	config.Thresholds.MaxCPU = 80.0
	config.Thresholds.MaxIOWait = 20.0
	config.Thresholds.MaxIRQ = 5.0
	config.Thresholds.MaxSoftIRQ = 10.0
	config.Thresholds.MaxMemory = 90.0
	config.Thresholds.MaxDisk = 95.0
	config.Thresholds.MaxConnections = 1000.0

	config.Monitoring.DiskPaths = []string{"/", "/var", "/tmp"}
	config.Monitoring.NetworkInterfaces = []string{"eth0", "lo"}

	config.Logging.File = defaultLogFile
	config.Logging.Debug = false

	config.Display.Enabled = false
	config.Display.Interval = 3 * time.Second

	return config
}

// parseCommandLineFlags parses command line arguments
func parseCommandLineFlags() CommandLineFlags {
	flags := CommandLineFlags{}

	flag.StringVar(&flags.ConfigFile, "config", "probe-config.yaml", "Path to YAML configuration file")
	flag.StringVar(&flags.ConfigFile, "c", "probe-config.yaml", "Path to YAML configuration file (short)")
	flag.BoolVar(&flags.GenerateConfig, "generate-config", false, "Generate default configuration file and exit")
	flag.BoolVar(&flags.Debug, "debug", false, "Enable debug logging")
	flag.BoolVar(&flags.Debug, "d", false, "Enable debug logging (short)")
	flag.BoolVar(&flags.Display, "display", false, "Enable terminal metrics display")
	flag.BoolVar(&flags.Help, "help", false, "Show help")
	flag.BoolVar(&flags.Help, "h", false, "Show help (short)")

	flag.Parse()

	return flags
}

// showHelp displays command line usage information
func showHelp() {
	fmt.Printf("probe-lbcdn-go - System Health Probe\n\n")
	fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
	fmt.Printf("Options:\n")
	flag.PrintDefaults()
	fmt.Printf("\nExamples:\n")
	fmt.Printf("  %s --generate-config          # Generate default config file\n", os.Args[0])
	fmt.Printf("  %s --config myconfig.yaml     # Use custom config file\n", os.Args[0])
	fmt.Printf("  %s --debug --display          # Run with debug logs and terminal display\n", os.Args[0])
}

// generateConfigFile creates a default configuration file
func generateConfigFile(filename string) error {
	config := getDefaultConfig()

	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Generated default configuration file: %s\n", filename)
	return nil
}

// loadConfig loads configuration from file and applies command line overrides
func loadConfig(flags CommandLineFlags) (Config, error) {
	config := getDefaultConfig()

	// Try to load from file if it exists
	if _, err := os.Stat(flags.ConfigFile); err == nil {
		data, err := os.ReadFile(flags.ConfigFile)
		if err != nil {
			return config, fmt.Errorf("failed to read config file: %w", err)
		}

		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return config, fmt.Errorf("failed to parse config file: %w", err)
		}

		fmt.Printf("Loaded configuration from: %s\n", flags.ConfigFile)
	} else {
		fmt.Printf("Config file not found, using defaults: %s\n", flags.ConfigFile)
	}

	// Apply command line overrides
	if flags.Debug {
		config.Logging.Debug = true
	}
	if flags.Display {
		config.Display.Enabled = true
	}

	return config, nil
}
