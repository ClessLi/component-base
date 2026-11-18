package v1

import (
	"context"
	"reflect"
	"runtime"
	"time"

	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	kitlog "github.com/go-kit/kit/log"
)

// formatter holds the state and configuration for log output.
// It manages log fields, timing, error/panic state, and result messages.
type formatter struct {
	fields          []interface{}    // fields contains log key-value pairs
	fieldExtractors []FieldExtractor // fieldExtractors are functions to extract fields from context
	result          interface{}      // result holds the operation result message
	err             error            // err holds any execution error
	panicValue      any              // panicValue holds recovered panic value
	startTime       time.Time        // startTime records when execution began
	logger          kitlog.Logger    // logger is an optional injected logger (via WithLogger) used for all levels
	infoLogger      kitlog.Logger    // infoLogger handles normal operation logs, resolved per Do
	errorLogger     kitlog.Logger    // errorLogger handles error-level logs, resolved per Do
	panicLogger     kitlog.Logger    // panicLogger handles panic-level logs, resolved per Do
}

// addInfos appends additional key-value pairs to the log fields.
func (f *formatter) addInfos(infos ...interface{}) *formatter {
	f.fields = append(f.fields, infos...)
	return f
}

// writeLog outputs a log message with timing information to the specified logger.
func (f *formatter) writeLog(logger kitlog.Logger, infos ...interface{}) {
	if !f.startTime.IsZero() {
		infos = append(infos, "took", time.Since(f.startTime))
	}
	logger.Log(append(f.fields, infos...)...)
}

// getLogger determines the appropriate logger based on current state.
// Priority: panic > error > info.
// Loggers are resolved once per Do by resolveLoggers, so output destinations
// follow the host's log.Init re-initialization.
func (f *formatter) getLogger() kitlog.Logger {
	if f.panicValue != nil {
		return f.panicLogger
	}
	if f.err != nil {
		return f.errorLogger
	}
	return f.infoLogger
}

// resolveLoggers binds the level-specific loggers for this formatter.
// An injected logger (via WithLogger) overrides lazy resolution for all levels.
// Otherwise loggers are fetched from the current global log state on each call,
// so a host log.Init after package load takes effect on subsequent Do calls.
// Note: log/v1 Init writes the global logger under a mutex while K()/V() read
// it without locking; hosts should complete log.Init before concurrent logging.
func (f *formatter) resolveLoggers() {
	if f.logger != nil {
		f.infoLogger = f.logger
		f.errorLogger = f.logger
		f.panicLogger = f.logger
		return
	}
	f.infoLogger = logV1.K()
	// Set errorLogger to ERROR level to ensure error logs are always recorded
	// Note: V(-2) means zapcore.ErrorLevel (4), because V() uses -1 * level
	f.errorLogger = logV1.V(-2).KitLogger()
	// Set panicLogger to PANIC level to ensure panic logs are always recorded
	// Note: V(-4) means zapcore.PanicLevel (4), because V() uses -1 * level
	f.panicLogger = logV1.V(-4).KitLogger()
}

// setResult sets the operation result message.
func (f *formatter) setResult(result interface{}) {
	f.result = result
}

// setErr sets the execution error.
func (f *formatter) setErr(err error) {
	f.err = err
}

// setStartTime records the execution start time.
func (f *formatter) setStartTime(begin time.Time) {
	f.startTime = begin
}

// recoverPanic captures panic value and logs it immediately.
func (f *formatter) recoverPanic(r any) {
	if r != nil {
		f.panicValue = r
		// Log panic immediately for quick detection
		f.writeLog(f.panicLogger, "panic", r)
	}
}

// emit outputs the final operation result with all accumulated fields.
// Should be called at the end of the deferred function.
func (f *formatter) emit() {
	var infos []interface{}

	// Add result info if exists
	if f.result != nil {
		infos = append(infos, "result", f.result)
	}

	// Add error info if exists
	if f.err != nil {
		infos = append(infos, "error", f.err)
	}

	// Output with appropriate logger level
	f.writeLog(f.getLogger(), infos...)
}

// newFormatter creates a new formatter instance with method name, the business
// call site captured in New, and configured options.
// It applies field extractors to populate initial log fields from context.
func newFormatter(ctx context.Context, method interface{}, callsite string, opts ...Option) *formatter {
	methodName := ""
	if method != nil {
		methodValue := reflect.ValueOf(method)
		if methodValue.Kind() == reflect.Func {
			methodName = runtime.FuncForPC(methodValue.Pointer()).Name()
		} else {
			methodName = reflect.TypeOf(method).String()
		}
	}

	f := &formatter{
		fields: []interface{}{
			"method", methodName,
		},
		startTime: time.Time{},
	}

	// Record the business call site under a key distinct from the backend's
	// own "caller" field (which points inside this package) to avoid duplicate
	// keys in structured output.
	if callsite != "" {
		f.fields = append(f.fields, "callsite", callsite)
	}

	// Apply options
	for _, opt := range opts {
		opt(f)
	}

	// Resolve level-specific loggers once per Do (after options so that
	// WithLogger injection takes precedence over lazy resolution)
	f.resolveLoggers()

	// Execute field extractors to populate fields
	for _, extractor := range f.fieldExtractors {
		f.fields = append(f.fields, extractor(ctx)...)
	}

	return f
}
