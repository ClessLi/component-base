package options

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ClessLi/component-base/pkg/generic_server/logger"

	"github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
)

const (
	flagInfoLevel             = "log.info-level"
	flagErrorLevel            = "log.error-level"
	flagDisableCaller         = "log.disable-caller"
	flagDisableStacktrace     = "log.disable-stacktrace"
	flagFormat                = "log.format"
	flagEnableColor           = "log.enable-color"
	flagInfoOutputPaths       = "log.info-output-paths"
	flagErrorOutputPaths      = "log.error-output-paths"
	flagInnerErrorOutputPaths = "log.inner-error-output-paths"
	flagDevelopment           = "log.development"

	consoleFormat = "console"
	jsonFormat    = "json"

	defaultProjectName      = "GenericServer"
	defaultLogBaseDir       = "logs"
	defaultInfoLogFileName  = "info.log"
	defaultErrorLogFileName = "error.log"
)

// LoggerOptions holds configuration for the logger.
//
// Usage:
//
//	// Option 1: Use default log paths (logs/info.log, logs/error.log)
//	opts := NewLoggerOptions()
//
//	// Option 2: Use project-specific log paths (logs/{projectName}.log, logs/{projectName}_error.log)
//	opts := NewLoggerOptionsWithProject("my-service")
//
//	// Option 3: Customize log paths via CLI flags or config file
//	opts := NewLoggerOptions()
//	opts.AddFlags(flag.CommandLine)
//	flag.Parse()
//
//	// Option 4: Use builder pattern (project name does not affect default log paths)
//	opts := NewLoggerOptions().WithProjectName("my-service")
//	opts.Complete() // Keeps: logs/info.log, logs/error.log (paths unchanged)
//
// Note: WithProjectName() only sets the project name for the logger,
// it does not adjust log paths. Log paths are determined by the constructor used
// or explicitly set via CLI flags/config file.
type LoggerOptions struct {
	projectName           string
	LogBaseDir            string   `json:"log-base-dir"             mapstructure:"log-base-dir"`
	InfoOutputPaths       []string `json:"info-output-paths"        mapstructure:"info-output-paths"`
	ErrorOutputPaths      []string `json:"error-output-paths"       mapstructure:"error-output-paths"`
	InnerErrorOutputPaths []string `json:"inner-error-output-paths" mapstructure:"inner-error-output-paths"`
	InfoLevel             string   `json:"info-level"               mapstructure:"info-level"`
	ErrorLevel            string   `json:"error-level"              mapstructure:"error-level"`
	Format                string   `json:"format"                   mapstructure:"format"`
	DisableCaller         bool     `json:"disable-caller"           mapstructure:"disable-caller"`
	DisableStacktrace     bool     `json:"disable-stacktrace"       mapstructure:"disable-stacktrace"`
	EnableColor           bool     `json:"enable-color"             mapstructure:"enable-color"`
	Development           bool     `json:"development"              mapstructure:"development"`
}

// NewLoggerOptions creates a new LoggerOptions with default log paths.
// This constructor uses default log paths: logs/info.log, logs/error.log
func NewLoggerOptions() *LoggerOptions {
	opts := &LoggerOptions{
		projectName:           defaultProjectName,
		InfoLevel:             zapcore.InfoLevel.String(),
		ErrorLevel:            zapcore.WarnLevel.String(),
		DisableCaller:         false,
		DisableStacktrace:     false,
		Format:                consoleFormat,
		EnableColor:           false,
		Development:           false,
		LogBaseDir:            defaultLogBaseDir,
		InfoOutputPaths:       []string{filepath.Join(defaultLogBaseDir, defaultInfoLogFileName)},
		ErrorOutputPaths:      []string{filepath.Join(defaultLogBaseDir, defaultErrorLogFileName)},
		InnerErrorOutputPaths: []string{"stderr"},
	}

	return opts
}

// NewLoggerOptionsWithProject creates a new LoggerOptions with project-specific configuration.
// This constructor customizes info and error log paths based on the project name.
// Example: logs/{projectName}.log, logs/{projectName}_error.log
func NewLoggerOptionsWithProject(projectName string) *LoggerOptions {
	opts := NewLoggerOptions().WithProjectName(projectName)
	opts.InfoOutputPaths = []string{filepath.Join(defaultLogBaseDir, projectName+".log")}
	opts.ErrorOutputPaths = []string{filepath.Join(defaultLogBaseDir, projectName+"_error.log")}
	return opts
}

// WithProjectName sets the project name for logger options.
// This method does not adjust log paths.
// Returns self for method chaining.
func (o *LoggerOptions) WithProjectName(projectName string) *LoggerOptions {
	o.projectName = projectName
	return o
}

// WithLogBaseDir sets the base log directory for logger options.
// Returns self for method chaining.
func (o *LoggerOptions) WithLogBaseDir(logBaseDir string) *LoggerOptions {
	o.LogBaseDir = logBaseDir
	return o
}

func (o *LoggerOptions) Validate() []error {
	var errs []error
	var zapLevel zapcore.Level
	// validate Info log
	if err := zapLevel.UnmarshalText([]byte(o.InfoLevel)); err != nil {
		errs = append(errs, err)
	}

	format := strings.ToLower(o.Format)
	if format != consoleFormat && format != jsonFormat {
		errs = append(errs, fmt.Errorf("not a valid log format: %q", o.Format))
	}

	// validate Error log
	if err := zapLevel.UnmarshalText([]byte(o.ErrorLevel)); err != nil {
		errs = append(errs, err)
	}

	return errs
}

func (o *LoggerOptions) AddFlags(fs *pflag.FlagSet) {
	// Add log directory flag
	fs.StringVar(&o.LogBaseDir, "log.log-base-dir", o.LogBaseDir, "Base directory for all log files.")

	// Add Info log flags
	fs.StringVar(&o.InfoLevel, flagInfoLevel, o.InfoLevel, "Minimum Info log output `LEVEL`.")
	fs.BoolVar(&o.DisableCaller, flagDisableCaller, o.DisableCaller, "Disable output of caller information in the log.")
	fs.BoolVar(&o.DisableStacktrace, flagDisableStacktrace,
		o.DisableStacktrace, "Disable the log to record a stack trace for all messages at or above panic level.")
	fs.StringVar(&o.Format, flagFormat, o.Format, "Log output `FORMAT`, support plain or json format.")
	fs.BoolVar(&o.EnableColor, flagEnableColor, o.EnableColor, "Enable output ansi colors in plain format logs.")
	fs.StringSliceVar(&o.InfoOutputPaths, flagInfoOutputPaths, o.InfoOutputPaths, "Output paths of Info log.")
	fs.StringSliceVar(&o.InnerErrorOutputPaths, flagInnerErrorOutputPaths, o.InnerErrorOutputPaths, "Inner Error output paths of log.")
	fs.BoolVar(
		&o.Development,
		flagDevelopment,
		o.Development,
		"Development puts the logger in development mode, which changes "+
			"the behavior of DPanicLevel and takes stacktraces more liberally.",
	)

	// Add Error log flags
	fs.StringVar(&o.ErrorLevel, flagErrorLevel, o.ErrorLevel, "Minimum Error log output `LEVEL`.")

	fs.StringSliceVar(&o.ErrorOutputPaths, flagErrorOutputPaths, o.ErrorOutputPaths, "Output paths of Error log.")
}

func (o *LoggerOptions) Complete() error {
	if o.projectName == "" {
		o.projectName = defaultProjectName
	}

	infoLogFileName := defaultInfoLogFileName
	errorLogFileName := defaultErrorLogFileName
	if o.projectName != "" && o.projectName != defaultProjectName {
		infoLogFileName = o.projectName + ".log"
		errorLogFileName = o.projectName + "_error.log"
	}

	if len(o.InfoOutputPaths) == 0 {
		o.InfoOutputPaths = []string{filepath.Join(o.LogBaseDir, infoLogFileName)}
	}

	if len(o.ErrorOutputPaths) == 0 {
		o.ErrorOutputPaths = []string{filepath.Join(o.LogBaseDir, errorLogFileName)}
	}

	return nil
}

func (o *LoggerOptions) ApplyTo(conf *logger.Config) error {
	conf.InfoLogOpts.Format = o.Format
	conf.InfoLogOpts.Name = o.projectName
	conf.InfoLogOpts.ErrorOutputPaths = o.InnerErrorOutputPaths
	conf.InfoLogOpts.Level = o.InfoLevel
	conf.InfoLogOpts.EnableColor = o.EnableColor
	conf.InfoLogOpts.Development = o.Development
	conf.InfoLogOpts.DisableCaller = o.DisableCaller
	conf.InfoLogOpts.DisableStacktrace = o.DisableStacktrace
	conf.InfoLogOpts.OutputPaths = o.InfoOutputPaths

	conf.ErrLogOpts.Format = o.Format
	conf.ErrLogOpts.ErrorOutputPaths = nil
	conf.ErrLogOpts.Level = o.ErrorLevel
	conf.ErrLogOpts.EnableColor = o.EnableColor
	conf.ErrLogOpts.Development = o.Development
	conf.ErrLogOpts.DisableCaller = o.DisableCaller
	conf.ErrLogOpts.DisableStacktrace = o.DisableStacktrace
	conf.ErrLogOpts.OutputPaths = o.ErrorOutputPaths

	return nil
}
