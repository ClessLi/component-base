// Package v1 provides a fluent API for method execution logging with panic recovery.
// It supports HTTP and gRPC protocol-specific field extraction through functional options.
//
// Basic usage:
//
//	err := methodlog.New(ctx, handler).
//	    Action(func(h Handler) error {
//	        return h.ServeHTTP(w, r)
//	    }).
//	    Result(func() string {
//	        return "request processed"
//	    }).
//	    Do()
//
// With protocol-specific field extraction:
//
//	err := methodlog.New(ctx, handler, methodlog.WithFieldExtractor(methodlog.HTTPFieldExtractor)).
//	    Action(func(h Handler) error {
//	        return h.ServeHTTP(w, r)
//	    }).
//	    Do()
//
// Result closure contract:
// Result/ResultWithError builders run inside Do's defer, BEFORE Do returns to
// the caller. When a closure captures the caller's own variable (e.g. the
// variable assigned by `err = ...Do()`), that assignment has not happened yet,
// so the closure must rely on values assigned INSIDE Action. To branch the
// result message on the final error returned by Do, prefer ResultWithError,
// which receives that error directly:
//
//	err := methodlog.New(ctx, method).
//	    Action(func(m MethodType) error {
//	        return m(args) // no outer assignment needed
//	    }).
//	    ResultWithError(func(err error) string {
//	        if err != nil {
//	            return "request failed"
//	        }
//	        return "request processed"
//	    }).
//	    Do()
//
// Log output destinations follow the host's log.Init: level loggers are
// resolved lazily on each Do. Hosts should complete log.Init before starting
// concurrent logging (log/v1 Init/K() synchronization is out of scope here).
//
// Caller semantics:
// The logging backend's own "caller" field reports a position INSIDE this
// package (the deferred closure in Do), not the business call site, because
// entries are emitted from Do's defer through several wrapper layers. New
// therefore captures the business call site up front and every entry carries
// it as the "callsite" field (package/file.go:line). Prefer "callsite" or
// "method" when locating business code.
package v1

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ClessLi/component-base/pkg/errors"
)

// ErrActionNotSet is returned when attempting to execute without configuring an action
var ErrActionNotSet = errors.New("methodlog: action not set")

// Action defines the signature for middleware action functions
// It receives a method and returns an error if execution fails
type Action[M any] func(method M) error

// infosBuilder defines a function that generates log key-value pairs after method execution
// It enables capturing runtime state via closures
type infosBuilder func() []interface{}

// resultBuilder defines a function that generates result message after method execution
// It allows access to closure-captured variables modified during execution
type resultBuilder func() string

// resultErrBuilder defines a result message builder that receives the final
// error returned by Do (nil on success or when the action panicked).
// It is the internal result slot shared by Result and ResultWithError.
type resultErrBuilder func(err error) string

// executor encapsulates the execution logic and configuration for logging middleware.
// It separates concerns: configuration collection vs. execution orchestration.
type executor[M any] struct {
	action        Action[M]        // action is the function to execute with logging
	infosBuilder  infosBuilder     // infosBuilder generates additional log key-value pairs
	resultBuilder resultErrBuilder // resultBuilder generates the result message, receiving the final error
}

// addInfos appends a builder function that generates log key-value pairs.
// Multiple calls compose builders, all executed in order during defer.
// The builder is invoked after method completion, enabling capture of runtime state.
func (e *executor[M]) addInfos(builder infosBuilder) *executor[M] {
	if builder == nil {
		return e
	}
	originalBuilder := e.infosBuilder
	e.infosBuilder = func() []interface{} {
		if originalBuilder != nil {
			return append(originalBuilder(), builder()...)
		}
		return builder()
	}
	return e
}

// setResult sets the result message builder function.
// Unlike addInfos, multiple calls replace previous builder (last one wins).
// The builder executes in defer block after method completion.
func (e *executor[M]) setResult(builder resultErrBuilder) *executor[M] {
	if builder == nil {
		return e
	}
	e.resultBuilder = builder
	return e
}

// callerSite renders the caller's position at the given runtime.Caller skip
// in short form (package/file.go:line), mirroring zapcore.ShortCallerEncoder.
// It returns an empty string when the position is unavailable.
func callerSite(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	// Keep only the last two path segments; runtime.Caller always uses slashes.
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		if prev := strings.LastIndex(file[:idx], "/"); prev >= 0 {
			file = file[prev+1:]
		}
	}
	return file + ":" + strconv.Itoa(line)
}

// Builder provides a fluent API for constructing logging middleware with panic recovery.
// Generic type M represents the method signature to be wrapped.
//
// Example:
//
//	builder := methodlog.New(ctx, myHandler).
//	    Action(func(h Handler) error {
//	        return h.ServeHTTP(w, r)
//	    }).
//	    Result(func() string {
//	        return "success"
//	    })
//	err := builder.Do()
type Builder[M any] struct {
	ctx       context.Context // ctx is the request context
	method    M               // method is the target method/handler to wrap
	formatter *formatter      // formatter holds log output state
	exec      *executor[M]    // exec holds execution configuration
	opts      []Option        // opts holds formatter configuration options
	callsite  string          // callsite is the business call site captured in New
}

// New creates a new Builder instance with initialized executor.
// Optional opts can be provided to configure field extractors.
// It also captures the caller's position (the business call site) so log
// entries can carry it as the "callsite" field; see package docs on caller
// semantics. If New is wrapped by another helper, the captured position is
// that helper's call site.
func New[M any](ctx context.Context, method M, opts ...Option) *Builder[M] {
	return &Builder[M]{
		ctx:      ctx,
		method:   method,
		exec:     new(executor[M]),
		opts:     opts,
		callsite: callerSite(2), // skip callerSite and New to reach the caller
	}
}

// Do wraps the configured action with logging, panic recovery, and error handling
// Execution flow:
// 1. Record start time
// 2. Execute action (via exec.action)
// 3. In defer: recover panics, set error, execute builders, output log
//
// Returns error from action execution or ErrActionNotSet if action is nil
func (b *Builder[M]) Do() (err error) {
	defer func(begin time.Time) {
		// Initialize formatter with context, method info, call site, and options
		b.formatter = newFormatter(b.ctx, b.method, b.callsite, b.opts...)
		b.formatter.setStartTime(begin)
		b.formatter.recoverPanic(recover())

		// Set error status before executing builders
		// Builders can check b.formatter.err to conditionally skip execution
		if err != nil {
			b.formatter.setErr(err)
		}

		// Execute log info builders if configured
		if b.exec.infosBuilder != nil {
			b.formatter.addInfos(b.exec.infosBuilder()...)
		}

		// Execute result message builder if configured, passing Do's final
		// named return value (nil when the action panicked)
		if b.exec.resultBuilder != nil {
			b.formatter.setResult(b.exec.resultBuilder(err))
		}

		// Apply default result message if none was set and no panic occurred
		if b.formatter.result == nil && b.formatter.panicValue == nil {
			if b.formatter.err == nil {
				b.formatter.setResult("operation completed successfully")
			}
			// Error case: error already logged, no default message needed
		}

		// Output final log entry
		defer b.formatter.emit()
	}(time.Now().Local())

	// Execute the configured action
	if b.exec.action != nil {
		err = b.exec.action(b.method)
		return
	}
	return ErrActionNotSet
}

// Action configures the action function to be executed with logging
// The action receives the target method and should invoke it, returning any error
// Example:
//
//	builder.Action(func(method func(ctx context.Context, id int64) error) error {
//	    return method(ctx, 123)
//	})
func (b *Builder[M]) Action(action Action[M]) *Builder[M] {
	b.exec.action = action
	return b
}

// Result configures a dynamic result message builder
// The builder function executes after method completion in the defer block,
// allowing access to closure-captured variables modified during execution
//
// Contract: the builder runs inside Do's defer, before Do returns. A closure
// capturing the caller's variable assigned by `err = ...Do()` still sees the
// old value (typically nil); captured variables must be assigned INSIDE Action.
// To branch on the final error without such assignment, use ResultWithError.
//
// Note: Multiple calls replace previous builder (last one wins), shared with
// ResultWithError. For conditional logic, check b.formatter.err inside the
// builder if needed
//
// Example:
//
//	var result *SomeType
//	methodlog.New(ctx, method).
//	    Result(func() string {
//	        if result != nil {
//	            return fmt.Sprintf("success, id: %d", result.ID)
//	        }
//	        return "success"
//	    }).
//	    Action(func(m MethodType) error {
//	        result, err = m(args) // assign inside Action so the closure sees it
//	        return err
//	    }).
//	    Do()
func (b *Builder[M]) Result(builder resultBuilder) *Builder[M] {
	if builder == nil {
		return b
	}
	// Adapt to the shared error-aware slot; the final error is ignored
	b.exec.setResult(func(error) string {
		return builder()
	})
	return b
}

// ResultWithError configures a result message builder that receives the final
// error returned by Do. Unlike Result, it does not require assigning captured
// variables inside Action: the builder is invoked in Do's defer with Do's own
// named return value, so `return method()` inside Action is enough.
//
// The error is nil on success and when the action panicked (panics are logged
// separately via the panic logger). Shares the same result slot with Result:
// multiple calls replace the previous builder (last one wins).
//
// Example:
//
//	err := methodlog.New(ctx, method).
//	    Action(func(m MethodType) error {
//	        return m(args)
//	    }).
//	    ResultWithError(func(err error) string {
//	        if err != nil {
//	            return fmt.Sprintf("request failed: %v", err)
//	        }
//	        return "request processed"
//	    }).
//	    Do()
func (b *Builder[M]) ResultWithError(builder func(err error) string) *Builder[M] {
	if builder == nil {
		return b
	}
	b.exec.setResult(builder)
	return b
}

// AddInfos configures builders for dynamic log key-value pairs
// Each builder executes after method completion, capturing runtime state via closures
// Multiple calls compose builders - all execute in order during defer
//
// Use cases:
// 1. Capture values modified during method execution
// 2. Access return values or state changes
// 3. Conditional logging based on execution results
//
// Example - Static values (captured at call time):
//
//	methodlog.New(ctx, method).
//	    AddInfos(func() []interface{} {
//	        return []interface{}{"key", "value"}
//	    }).
//	    Do()
//
// Example - Dynamic values (captured after execution):
//
//	var id int64
//	var name string
//	methodlog.New(ctx, method).
//	    AddInfos(func() []interface{} {
//	        return []interface{}{"id", id, "name", name}
//	    }).
//	    Action(func(m MethodType) error {
//	        id = 123
//	        name = "test"
//	        return m()
//	    }).
//	    Do()
func (b *Builder[M]) AddInfos(builder infosBuilder) *Builder[M] {
	b.exec.addInfos(builder)
	return b
}
