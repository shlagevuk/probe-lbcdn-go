package main

import (
	"io"
	"log"
	"os"
)

// setupLogging configures the logging system based on configuration
func setupLogging(config Config) (*os.File, error) {
	var logFile *os.File
	var err error

	// Open log file if specified
	if config.Logging.File != "" {
		logFile, err = os.OpenFile(config.Logging.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Warning: Failed to open log file %s: %v. Logging to stderr only.", config.Logging.File, err)
		}
	}

	// Configure log output
	if logFile != nil {
		// Log to both file and stderr
		multiWriter := io.MultiWriter(os.Stderr, logFile)
		log.SetOutput(multiWriter)
	} else {
		// Log to stderr only
		log.SetOutput(os.Stderr)
	}

	// Set log flags based on debug mode
	if config.Logging.Debug {
		// Include date, time, and file info for debug mode
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Printf("Debug logging enabled")
	} else {
		// Standard format for production
		log.SetFlags(log.LstdFlags)
	}

	return logFile, nil
}

// logDebug logs a message only if debug mode is enabled
func logDebug(config Config, format string, args ...interface{}) {
	if config.Logging.Debug {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// logInfo logs an informational message
func logInfo(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

// logWarning logs a warning message
func logWarning(format string, args ...interface{}) {
	log.Printf("[WARNING] "+format, args...)
}

// logError logs an error message
func logError(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}
