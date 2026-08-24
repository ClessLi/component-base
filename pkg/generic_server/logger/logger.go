package logger

import (
	"os"

	"github.com/ClessLi/component-base/pkg/errors"
)

// Logger provides a wrapper for logger initialization and flushing.
//
// It encapsulates the initialization logic (including directory creation
// and logger setup) and provides a Flush method to ensure all buffered
// log entries are written before the application exits.
type Logger struct {
	initFunc func() error
	flush    func()
}

// Init initializes the logger by creating log directories and setting up
// the underlying logging system. It should be called once during application startup.
func (l *Logger) Init() error {
	if l.initFunc == nil {
		return errors.New("logger init function is nil")
	}
	return l.initFunc()
}

// Flush ensures all buffered log entries are written to their destinations.
// It should be called before the application exits to prevent log loss.
func (l *Logger) Flush() {
	if l.flush == nil {
		return
	}
	l.flush()
}

// createLogDir creates the log directory if it does not exist.
// It returns an error if the path exists but is not a directory.
func createLogDir(dirpath string) error {
	if info, err := os.Stat(dirpath); os.IsNotExist(err) {
		return os.MkdirAll(dirpath, 0755)
	} else if err != nil {
		return err
	} else if !info.IsDir() {
		return errors.Errorf("The path '%s' exists, and it is not a directory", dirpath)
	}

	return nil
}
