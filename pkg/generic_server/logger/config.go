package logger

import (
	"path/filepath"

	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
)

// Config holds the configuration for creating a logger.
//
// It contains separate options for Info-level and Error-level log outputs,
// allowing independent configuration of log levels, output paths, and formatting.
type Config struct {
	// InfoLogOpts holds configuration for Info-level log output.
	InfoLogOpts *logV1.Options
	// ErrLogOpts holds configuration for Error-level log output.
	ErrLogOpts *logV1.Options
}

// NewConfig creates a new Config with default options for both Info and Error logs.
func NewConfig() *Config {
	return &Config{
		InfoLogOpts: logV1.NewOptions(),
		ErrLogOpts:  logV1.NewOptions(),
	}
}

// CompletedConfig is a completed configuration ready to create a logger.
//
// It wraps Config and provides the NewLogger method to initialize
// the logger with the configured options.
type CompletedConfig struct {
	*Config
}

// Complete transforms Config into a CompletedConfig for logger creation.
func (c *Config) Complete() CompletedConfig {
	return CompletedConfig{c}
}

// NewLogger creates and initializes a new Logger instance.
//
// It performs the following initialization steps:
//  1. Creates log directories for all output paths (excluding stdout/stderr).
//  2. Initializes the underlying logging system with Info and Error log options.
//
// Returns an error if directory creation or logger initialization fails.
func (c CompletedConfig) NewLogger() (*Logger, error) {
	return &Logger{
		initFunc: func() error {
			// Create log directories for Info and Error outputs
			if err := createLogDirs(c.InfoLogOpts.OutputPaths); err != nil {
				return err
			}
			if err := createLogDirs(c.ErrLogOpts.OutputPaths); err != nil {
				return err
			}

			logV1.Init(c.InfoLogOpts, c.ErrLogOpts)

			return nil
		},
		flush: func() {
			logV1.Flush()
		},
	}, nil
}

// createLogDirs creates directories for all output paths, skipping stdout/stderr.
func createLogDirs(outputPaths []string) error {
	for _, outputPath := range outputPaths {
		if outputPath == "stdout" || outputPath == "stderr" {
			continue
		}
		if err := createLogDir(filepath.Dir(outputPath)); err != nil {
			return err
		}
	}
	return nil
}
