package v1

import (
	"context"
	"reflect"
	"runtime"
	"time"

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
func (f *formatter) getLogger() kitlog.Logger {
	if f.panicValue != nil {
		return panicLogger
	}
	if f.err != nil {
		return errorLogger
	}
	return infoLogger
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
		f.writeLog(panicLogger, "panic", r)
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

// newFormatter creates a new formatter instance with method name and configured options.
// It applies field extractors to populate initial log fields from context.
func newFormatter(ctx context.Context, method interface{}, opts ...Option) *formatter {
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

	// Apply options
	for _, opt := range opts {
		opt(f)
	}

	// Execute field extractors to populate fields
	for _, extractor := range f.fieldExtractors {
		f.fields = append(f.fields, extractor(ctx)...)
	}

	return f
}
