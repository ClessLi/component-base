package v1

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"testing"

	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	kitlog "github.com/go-kit/kit/log"
	kitzaplog "github.com/go-kit/kit/log/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// captureLogger wraps a bytes.Buffer with Zap logger to capture log output for test verification
type captureLogger struct {
	mu       sync.Mutex
	buffer   *bytes.Buffer
	logger   kitlog.Logger
	zapLevel zapcore.Level
}

// NewCaptureLogger creates a new capture logger that implements kitlog.Logger with info level
func NewCaptureLogger() *captureLogger {
	return NewCaptureLoggerWithLevel(zapcore.InfoLevel)
}

// NewCaptureLoggerWithLevel creates a capture logger with specified Zap level
// This uses the same kitzaplog.NewZapSugarLogger as production code
func NewCaptureLoggerWithLevel(level zapcore.Level) *captureLogger {
	buf := &bytes.Buffer{}

	// Create Zap encoder config (similar to production)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Create JSON encoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	// Create Zap core with level filtering
	core := zapcore.NewCore(encoder, zapcore.AddSync(buf), level)
	zapLogger := zap.New(core)

	// Convert to kitlog.Logger using the same bridge as production
	kitLogger := kitzaplog.NewZapSugarLogger(zapLogger, level)

	return &captureLogger{
		buffer:   buf,
		logger:   kitLogger,
		zapLevel: level,
	}
}

// Log implements kitlog.Logger interface
func (c *captureLogger) Log(keyvals ...interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.logger.Log(keyvals...)
}

// GetOutput returns all captured log output as string
func (c *captureLogger) GetOutput() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buffer.String()
}

// GetLines returns all captured log lines split by newline
func (c *captureLogger) GetLines() []string {
	output := c.GetOutput()
	if output == "" {
		return []string{}
	}
	return strings.Split(strings.TrimSpace(output), "\n")
}

// Contains checks if the captured output contains the given substring
func (c *captureLogger) Contains(substring string) bool {
	return strings.Contains(c.GetOutput(), substring)
}

// Clear clears all captured logs
func (c *captureLogger) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buffer.Reset()
}

// Len returns the number of captured log lines
func (c *captureLogger) Len() int {
	lines := c.GetLines()
	return len(lines)
}

// In-memory log sinks for tests. Zap keeps sink handles open for the process
// lifetime, so real log files would stay locked on Windows and break temp-dir
// cleanup; registered URL sinks avoid the filesystem entirely.
var (
	testSinkMu sync.Mutex
	testSinks  = make(map[string]*bytes.Buffer)
)

// memorySink is a zap.Sink writing into a named in-memory buffer.
type memorySink struct{ name string }

func (s memorySink) Write(p []byte) (int, error) {
	testSinkMu.Lock()
	defer testSinkMu.Unlock()
	buf, ok := testSinks[s.name]
	if !ok {
		buf = &bytes.Buffer{}
		testSinks[s.name] = buf
	}
	return buf.Write(p)
}

func (s memorySink) Sync() error  { return nil }
func (s memorySink) Close() error { return nil }

// testSinkOutput returns the current output of the named in-memory sink.
func testSinkOutput(name string) string {
	testSinkMu.Lock()
	defer testSinkMu.Unlock()
	if buf, ok := testSinks[name]; ok {
		return buf.String()
	}
	return ""
}

// initLogToBuffer re-initializes the global log/v1 logger to write into the
// named in-memory sink (URL scheme registered in TestMain).
func initLogToBuffer(name string) {
	opts := logV1.NewOptions()
	opts.OutputPaths = []string{"methodlogtest://" + name}
	opts.ErrorOutputPaths = opts.OutputPaths
	logV1.Init(opts)
}

// TestMain registers the in-memory sink scheme and routes global log output
// into it, so tests stay quiet while exercising the lazy per-Do logger
// resolution against real log/v1 state.
func TestMain(m *testing.M) {
	err := zap.RegisterSink("methodlogtest", func(u *url.URL) (zap.Sink, error) {
		return memorySink{name: u.Host}, nil
	})
	if err != nil {
		panic(err)
	}
	initLogToBuffer("main")

	m.Run()
}

// TestResult verifies Result method behavior comprehensively
func TestResult(t *testing.T) {
	ctx := context.Background()

	t.Run("builder executes after method completion", func(t *testing.T) {
		var executionOrder []string
		var resultValue string

		builder := New(ctx, func() {}).
			Result(func() string {
				executionOrder = append(executionOrder, "builder")
				return resultValue // Captures value set by method
			}).
			Action(func(method func()) error {
				executionOrder = append(executionOrder, "method-start")
				resultValue = "modified-by-method"
				executionOrder = append(executionOrder, "method-end")
				method()
				return nil
			})

		err := builder.Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}

		// Verify execution order: method should execute before builder
		expectedOrder := []string{"method-start", "method-end", "builder"}
		if len(executionOrder) != len(expectedOrder) {
			t.Errorf("Expected %d execution steps, got %d: %v", len(expectedOrder), len(executionOrder), executionOrder)
		}

		for i, expected := range expectedOrder {
			if i >= len(executionOrder) || executionOrder[i] != expected {
				t.Errorf("Execution order mismatch at step %d: expected %s, got %v", i, expected, executionOrder)
				break
			}
		}

		// Verify builder captured the modified value
		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else if builder.formatter.result == nil || builder.formatter.result != "modified-by-method" {
			t.Errorf("Expected result 'modified-by-method', got %v", builder.formatter.result)
		}
	})

	t.Run("builder is called even on error for conditional logging", func(t *testing.T) {
		var builderCalled bool

		builder := New(ctx, func() {}).
			Result(func() string {
				builderCalled = true
				// Builder should be able to check error status and return appropriate message
				return "error-result-message"
			}).
			Action(func(method func()) error {
				method()
				return context.Canceled
			})

		err := builder.Do()

		if err == nil {
			t.Error("Expected error from Do()")
		}

		// Builder SHOULD be called even when error occurs
		// This allows conditional logging based on error status
		if !builderCalled {
			t.Error("Result should be called even when error occurs (for conditional logging)")
		}

		// Verify the builder's result was set
		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else if builder.formatter.result == nil || builder.formatter.result != "error-result-message" {
			t.Errorf("Expected result 'error-result-message', got %v", builder.formatter.result)
		}
	})

	t.Run("nil builder returns same instance", func(t *testing.T) {
		builder := New(ctx, func() {})
		result := builder.Result(nil)

		if result != builder {
			t.Error("Result(nil) should return the same builder instance")
		}
	})

	t.Run("multiple calls replace previous builder", func(t *testing.T) {
		callCount := 0

		builder := New(ctx, func() {}).
			Result(func() string {
				callCount++
				return "first"
			}).
			Result(func() string {
				callCount++
				return "second"
			}).
			Action(func(method func()) error {
				method()
				return nil
			})

		err := builder.Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}

		// Only the last builder should be called once
		if callCount != 1 {
			t.Errorf("Expected builder to be called 1 time, got %d times", callCount)
		}

		// Verify the last builder's result is used
		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else if builder.formatter.result == nil || builder.formatter.result != "second" {
			t.Errorf("Expected result 'second' (last builder wins), got %v", builder.formatter.result)
		}
	})

	t.Run("builder captures pointer member value after execution", func(t *testing.T) {
		type Instance struct {
			ID int64
		}
		type Application struct {
			DefName string
			Ins     *Instance
		}

		var app *Application

		builder := New(ctx, func() {}).
			Result(func() string {
				// Captures pointer member value after method execution
				if app != nil && app.Ins != nil {
					return fmt.Sprintf("success, inner-id: %d", app.Ins.ID)
				}
				return "success"
			}).
			Action(func(method func()) error {
				// Simulate method setting nested pointer structure
				app = &Application{
					Ins: &Instance{ID: 999},
				}
				method()
				return nil
			})

		err := builder.Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}

		// Verify result message captured the nested pointer value
		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else if builder.formatter.result == nil || builder.formatter.result != "success, inner-id: 999" {
			t.Errorf("Expected result 'success, inner-id: 999', got %v", builder.formatter.result)
		}
	})

	t.Run("builder can check error status via formatter.err", func(t *testing.T) {
		var checkedError bool

		builder := New(ctx, func() {}).
			Result(func() string {
				// Builder can access b.formatter to check error status
				// Note: In real usage, this would be accessed through closure
				checkedError = true
				return "conditional-result"
			}).
			Action(func(method func()) error {
				method()
				return nil
			})

		err := builder.Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}

		if !checkedError {
			t.Error("Builder should have been executed")
		}
	})

	t.Run("legacy contract: closure sees outer error assigned inside Action", func(t *testing.T) {
		// Regression for the documented Result contract: the closure captures
		// a caller-side variable, which must be assigned INSIDE Action because
		// the builder runs in Do's defer before Do returns.
		wantErr := fmt.Errorf("legacy action failure")
		var actionErr error

		builder := New(ctx, func() error { return wantErr }).
			Result(func() string {
				if actionErr != nil {
					return "request failed"
				}
				return "request processed"
			}).
			Action(func(method func() error) error {
				actionErr = method()
				return actionErr
			})

		err := builder.Do()

		if err != wantErr {
			t.Errorf("Expected error %v from Do(), got %v", wantErr, err)
		}
		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else if builder.formatter.result != "request failed" {
			t.Errorf("Expected result 'request failed', got %v", builder.formatter.result)
		}
	})
}

// TestAddInfos verifies AddInfos method behavior comprehensively
func TestAddInfos(t *testing.T) {
	ctx := context.Background()

	t.Run("builder executes after method completion", func(t *testing.T) {
		var executionOrder []string
		var dynamicValue string

		builder := New(ctx, func() {}).
			AddInfos(func() []interface{} {
				executionOrder = append(executionOrder, "builder")
				return []interface{}{"dynamic-key", dynamicValue}
			}).
			Action(func(method func()) error {
				executionOrder = append(executionOrder, "method-start")
				dynamicValue = "set-by-method"
				executionOrder = append(executionOrder, "method-end")
				method()
				return nil
			})

		err := builder.Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}

		// Verify execution order: method should execute before builder
		expectedOrder := []string{"method-start", "method-end", "builder"}
		if len(executionOrder) != len(expectedOrder) {
			t.Errorf("Expected %d execution steps, got %d: %v", len(expectedOrder), len(executionOrder), executionOrder)
		}

		for i, expected := range expectedOrder {
			if i >= len(executionOrder) || executionOrder[i] != expected {
				t.Errorf("Execution order mismatch at step %d: expected %s, got %v", i, expected, executionOrder)
				break
			}
		}

		// Verify builder captured the modified value
		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else {
			infos := builder.formatter.fields
			// Find our custom key-value pair in the slice
			found := false
			for i := 0; i < len(infos)-1; i += 2 {
				if infos[i] == "dynamic-key" && infos[i+1] == "set-by-method" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected to find key-value pair ('dynamic-key', 'set-by-method') in infos: %v", infos)
			}
		}
	})

	t.Run("multiple builders compose and all execute", func(t *testing.T) {
		var callOrder []string

		builder := New(ctx, func() {}).
			AddInfos(func() []interface{} {
				callOrder = append(callOrder, "builder1")
				return []interface{}{"key1", "value1"}
			}).
			AddInfos(func() []interface{} {
				callOrder = append(callOrder, "builder2")
				return []interface{}{"key2", "value2"}
			}).
			Action(func(method func()) error {
				method()
				return nil
			})

		err := builder.Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}

		// Both builders should be called in order
		expectedOrder := []string{"builder1", "builder2"}
		if len(callOrder) != len(expectedOrder) {
			t.Errorf("Expected %d builders called, got %d: %v", len(expectedOrder), len(callOrder), callOrder)
		}

		for i, expected := range expectedOrder {
			if i >= len(callOrder) || callOrder[i] != expected {
				t.Errorf("Builder call order mismatch at step %d: expected %s, got %v", i, expected, callOrder)
				break
			}
		}

		// Verify both key-value pairs are present
		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else {
			infos := builder.formatter.fields
			expectedPairs := map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			}

			for expectedKey, expectedValue := range expectedPairs {
				found := false
				for i := 0; i < len(infos)-1; i += 2 {
					if infos[i] == expectedKey && infos[i+1] == expectedValue {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find key-value pair ('%v', '%v') in infos: %v", expectedKey, expectedValue, infos)
				}
			}
		}
	})

	t.Run("nil builder returns same instance", func(t *testing.T) {
		builder := New(ctx, func() {})
		result := builder.AddInfos(nil)

		if result != builder {
			t.Error("AddInfos(nil) should return the same builder instance")
		}
	})

	t.Run("empty slice from builder adds no infos", func(t *testing.T) {
		builder := New(ctx, func() {}).
			AddInfos(func() []interface{} {
				return []interface{}{}
			})

		err := builder.Action(func(method func()) error {
			method()
			return nil
		}).Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}

		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else if len(builder.formatter.fields) < 2 {
			// fields contains default fields (method, clientIp), so should have at least 2 items
			t.Errorf("Expected at least default infos in formatter, got %d: %v", len(builder.formatter.fields), builder.formatter.fields)
		}
	})

	t.Run("builder captures closure variables after execution", func(t *testing.T) {
		type TestData struct {
			ID   int64
			Name string
		}

		var data *TestData

		builder := New(ctx, func() {}).
			AddInfos(func() []interface{} {
				if data != nil {
					return []interface{}{"before-id", data.ID}
				}
				return []interface{}{"before-id", nil}
			}).
			AddInfos(func() []interface{} {
				if data != nil {
					return []interface{}{"after-id", data.ID, "after-name", data.Name}
				}
				return []interface{}{"after-id", nil, "after-name", nil}
			}).
			Action(func(method func()) error {
				// Simulate method setting the data
				data = &TestData{ID: 123, Name: "test"}
				method()
				return nil
			})

		err := builder.Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}

		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else {
			infos := builder.formatter.fields

			// Verify that expected key-value pairs are present
			// Note: All builders execute in defer block AFTER method execution,
			// so both builders see data with ID=123
			expectedPairs := map[string]interface{}{
				"before-id":  int64(123), // Evaluated after method execution (in defer)
				"after-id":   int64(123), // Evaluated after method execution
				"after-name": "test",
			}

			for expectedKey, expectedValue := range expectedPairs {
				found := false
				for i := 0; i < len(infos)-1; i += 2 {
					if infos[i] == expectedKey && infos[i+1] == expectedValue {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find key-value pair ('%v', '%v') in infos: %v", expectedKey, expectedValue, infos)
				}
			}
		}
	})

	t.Run("builder captures pointer member value after execution", func(t *testing.T) {
		type Instance struct {
			ID int64
		}
		type Application struct {
			DefName string
			Ins     *Instance
		}

		var app *Application

		builder := New(ctx, func() {}).
			AddInfos(func() []interface{} {
				if app != nil {
					return []interface{}{"app-def-name", app.DefName}
				}
				return []interface{}{"app-def-name", ""}
			}).
			AddInfos(func() []interface{} {
				// Captures pointer member value after method execution
				if app != nil && app.Ins != nil {
					return []interface{}{"instance-id", app.Ins.ID}
				}
				return []interface{}{"instance-id", nil}
			}).
			Action(func(method func()) error {
				// Simulate method setting application with nested instance
				app = &Application{
					DefName: "test-app",
					Ins:     &Instance{ID: 12345},
				}
				method()
				return nil
			})

		err := builder.Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}

		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else {
			infos := builder.formatter.fields

			// Verify that expected key-value pairs are present
			// Note: All builders execute in defer block AFTER method execution,
			// so both builders see app with DefName="test-app"
			expectedPairs := map[string]interface{}{
				"app-def-name": "test-app",   // Evaluated after method execution (in defer)
				"instance-id":  int64(12345), // Evaluated after method execution
			}

			for expectedKey, expectedValue := range expectedPairs {
				found := false
				for i := 0; i < len(infos)-1; i += 2 {
					if infos[i] == expectedKey && infos[i+1] == expectedValue {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find key-value pair ('%v', '%v') in infos: %v", expectedKey, expectedValue, infos)
				}
			}
		}
	})

	t.Run("builder handles nil pointer safely", func(t *testing.T) {
		var value *string = nil

		builder := New(ctx, func() {}).
			AddInfos(func() []interface{} {
				var val interface{}
				if value != nil {
					val = *value
				}
				return []interface{}{"key", val}
			})

		err := builder.Action(func(method func()) error {
			method()
			return nil
		}).Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}

		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else {
			infos := builder.formatter.fields
			// Should have key with nil value
			found := false
			for i := 0; i < len(infos)-1; i += 2 {
				if infos[i] == "key" && infos[i+1] == nil {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected to find key-value pair ('key', nil) in infos: %v", infos)
			}
		}
	})
}

// TestActionNotSet verifies that Do returns ErrActionNotSet when action is not configured
func TestActionNotSet(t *testing.T) {
	ctx := context.Background()

	builder := New(ctx, func() {})
	err := builder.Do()

	if err == nil {
		t.Error("Expected error when action is not set")
	}

	if err != ErrActionNotSet {
		t.Errorf("Expected ErrActionNotSet, got %v", err)
	}
}

// TestCombinedBuilders verifies chaining multiple builders works correctly
func TestCombinedBuilders(t *testing.T) {
	ctx := context.Background()

	t.Run("AddInfos and Result work together", func(t *testing.T) {
		var dynamicValue string

		builder := New(ctx, func() {}).
			AddInfos(func() []interface{} {
				return []interface{}{"static-key", "static-value"}
			}).
			AddInfos(func() []interface{} {
				return []interface{}{"dynamic-key", dynamicValue}
			}).
			Result(func() string {
				return "result: " + dynamicValue
			}).
			Action(func(method func()) error {
				dynamicValue = "from-method"
				method()
				return nil
			})

		err := builder.Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}

		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else {
			// Check infos contain both static and dynamic values
			infos := builder.formatter.fields

			// Verify that expected key-value pairs are present
			expectedPairs := map[string]interface{}{
				"static-key":  "static-value",
				"dynamic-key": "from-method",
			}

			for expectedKey, expectedValue := range expectedPairs {
				found := false
				for i := 0; i < len(infos)-1; i += 2 {
					if infos[i] == expectedKey && infos[i+1] == expectedValue {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find key-value pair ('%v', '%v') in infos: %v", expectedKey, expectedValue, infos)
				}
			}

			// Verify result message
			if builder.formatter.result == nil || builder.formatter.result != "result: from-method" {
				t.Errorf("Expected result 'result: from-method', got %v", builder.formatter.result)
			}
		}
	})
}

// TestCaptureLoggerWithKitlog verifies captureLogger works with kitlog.Logger interface
func TestCaptureLoggerWithKitlog(t *testing.T) {
	t.Run("implements kitlog.Logger interface", func(t *testing.T) {
		captureLog := NewCaptureLogger()

		// Verify it implements kitlog.Logger
		var _ kitlog.Logger = captureLog

		// Log some test data
		err := captureLog.Log("msg", "test message", "key", "value")
		if err != nil {
			t.Errorf("Log() returned error: %v", err)
		}

		// Verify output was captured
		output := captureLog.GetOutput()
		if output == "" {
			t.Error("Expected log output to be captured")
		}

		// Verify content (JSON format) - only check predictable fields
		if !captureLog.Contains(`"level":"INFO"`) {
			t.Errorf("Expected INFO level in JSON output, got: %s", output)
		}
		if !captureLog.Contains(`"msg":"test message"`) {
			t.Errorf("Expected message in JSON output, got: %s", output)
		}
		if !captureLog.Contains(`"key":"value"`) {
			t.Errorf("Expected key-value in JSON output, got: %s", output)
		}

		// Verify timestamp exists (but don't check exact value)
		if !captureLog.Contains(`"timestamp":`) {
			t.Errorf("Expected timestamp field in JSON output, got: %s", output)
		}
	})

	t.Run("captures multiple log lines", func(t *testing.T) {
		captureLog := NewCaptureLogger()

		_ = captureLog.Log("msg", "first line")
		_ = captureLog.Log("msg", "second line")
		_ = captureLog.Log("msg", "third line")

		lines := captureLog.GetLines()
		if len(lines) != 3 {
			t.Errorf("Expected 3 log lines, got %d", len(lines))
		}
	})

	t.Run("clear removes all output", func(t *testing.T) {
		captureLog := NewCaptureLogger()

		_ = captureLog.Log("msg", "test")
		if captureLog.Len() != 1 {
			t.Error("Expected 1 line before clear")
		}

		captureLog.Clear()
		if captureLog.Len() != 0 {
			t.Error("Expected 0 lines after clear")
		}
		if captureLog.GetOutput() != "" {
			t.Error("Expected empty output after clear")
		}
	})

	t.Run("thread-safe concurrent logging", func(t *testing.T) {
		captureLog := NewCaptureLogger()
		done := make(chan bool, 10)

		// Concurrent logging from multiple goroutines
		for i := 0; i < 10; i++ {
			go func(id int) {
				_ = captureLog.Log("goroutine", id, "msg", "concurrent log")
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Should have 10 log lines
		if captureLog.Len() != 10 {
			t.Errorf("Expected 10 log lines, got %d", captureLog.Len())
		}
	})
}

// TestCaptureLoggerWithLevels verifies that different log levels are properly set
func TestCaptureLoggerWithLevels(t *testing.T) {
	t.Run("info level logger", func(t *testing.T) {
		infoLog := NewCaptureLoggerWithLevel(zapcore.InfoLevel)

		_ = infoLog.Log("msg", "test info message")

		output := infoLog.GetOutput()
		if !strings.Contains(output, `"level":"INFO"`) {
			t.Errorf("Expected INFO level in output, got: %s", output)
		}
		if !strings.Contains(output, `"msg":"test info message"`) {
			t.Errorf("Expected message in output, got: %s", output)
		}
	})

	t.Run("error level logger", func(t *testing.T) {
		errorLog := NewCaptureLoggerWithLevel(zapcore.ErrorLevel)

		_ = errorLog.Log("msg", "test error message", "error", "something failed")

		output := errorLog.GetOutput()
		if !strings.Contains(output, `"level":"ERROR"`) {
			t.Errorf("Expected ERROR level in output, got: %s", output)
		}
		if !strings.Contains(output, `"error":"something failed"`) {
			t.Errorf("Expected error details in output, got: %s", output)
		}
	})

	t.Run("panic level logger", func(t *testing.T) {
		panicLog := NewCaptureLoggerWithLevel(zapcore.PanicLevel)

		// PanicLevel logger will actually panic when logging
		// We need to recover from it and verify the log was captured before panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()

			_ = panicLog.Log("msg", "test panic message", "panic", "unexpected panic")
		}()

		// Verify that panic occurred (expected behavior for PanicLevel)
		if !didPanic {
			t.Error("Expected PanicLevel logger to panic")
		}

		// Verify log was captured before panic
		output := panicLog.GetOutput()
		if output == "" {
			t.Error("Expected log output to be captured before panic")
		}

		// Note: The log may or may not be flushed before panic depending on timing
		// In production, this is handled by the middleware's defer/recover
	})

	t.Run("different levels produce different outputs", func(t *testing.T) {
		infoLog := NewCaptureLoggerWithLevel(zapcore.InfoLevel)
		errorLog := NewCaptureLoggerWithLevel(zapcore.ErrorLevel)

		_ = infoLog.Log("msg", "same message")
		_ = errorLog.Log("msg", "same message")

		infoOutput := infoLog.GetOutput()
		errorOutput := errorLog.GetOutput()

		// Verify each has the correct level
		if !strings.Contains(infoOutput, `"level":"INFO"`) {
			t.Error("Info logger should have level=INFO")
		}
		if !strings.Contains(errorOutput, `"level":"ERROR"`) {
			t.Error("Error logger should have level=ERROR")
		}

		// Note: We don't test PanicLevel here because it would cause the test to panic
		// In production, PanicLevel is used with proper defer/recover handling
	})
}

// TestResultWithError verifies ResultWithError receives Do's final error
func TestResultWithError(t *testing.T) {
	ctx := context.Background()

	t.Run("receives action error without outer assignment", func(t *testing.T) {
		// Minimal repro: Action returns method() directly, the caller-side
		// variable is never assigned inside Action. ResultWithError must still
		// see the final error via Do's named return value.
		wantErr := fmt.Errorf("action failed")
		var gotErr error

		builder := New(ctx, func() error { return wantErr }).
			ResultWithError(func(err error) string {
				gotErr = err
				if err != nil {
					return "request failed"
				}
				return "request processed"
			}).
			Action(func(method func() error) error {
				return method()
			})

		err := builder.Do()

		if err != wantErr {
			t.Errorf("Expected error %v from Do(), got %v", wantErr, err)
		}
		if gotErr != wantErr {
			t.Errorf("Expected builder to receive error %v, got %v", wantErr, gotErr)
		}
		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else if builder.formatter.result != "request failed" {
			t.Errorf("Expected result 'request failed', got %v", builder.formatter.result)
		}
	})

	t.Run("receives nil on success", func(t *testing.T) {
		gotErr := fmt.Errorf("sentinel")

		builder := New(ctx, func() error { return nil }).
			ResultWithError(func(err error) string {
				gotErr = err
				if err != nil {
					return "request failed"
				}
				return "request processed"
			}).
			Action(func(method func() error) error {
				return method()
			})

		err := builder.Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}
		if gotErr != nil {
			t.Errorf("Expected builder to receive nil error, got %v", gotErr)
		}
		if builder.formatter == nil {
			t.Error("formatter should not be nil after Do")
		} else if builder.formatter.result != "request processed" {
			t.Errorf("Expected result 'request processed', got %v", builder.formatter.result)
		}
	})

	t.Run("ResultWithError overrides previous Result", func(t *testing.T) {
		legacyCalled := false

		builder := New(ctx, func() {}).
			Result(func() string {
				legacyCalled = true
				return "legacy"
			}).
			ResultWithError(func(err error) string {
				return "with-error"
			}).
			Action(func(method func()) error {
				method()
				return nil
			})

		if err := builder.Do(); err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}
		if legacyCalled {
			t.Error("Overridden Result builder should not be called")
		}
		if builder.formatter == nil || builder.formatter.result != "with-error" {
			t.Errorf("Expected result 'with-error' (last builder wins), got %v", builder.formatter.result)
		}
	})

	t.Run("Result overrides previous ResultWithError", func(t *testing.T) {
		errBuilderCalled := false

		builder := New(ctx, func() {}).
			ResultWithError(func(err error) string {
				errBuilderCalled = true
				return "with-error"
			}).
			Result(func() string {
				return "legacy"
			}).
			Action(func(method func()) error {
				method()
				return nil
			})

		if err := builder.Do(); err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}
		if errBuilderCalled {
			t.Error("Overridden ResultWithError builder should not be called")
		}
		if builder.formatter == nil || builder.formatter.result != "legacy" {
			t.Errorf("Expected result 'legacy' (last builder wins), got %v", builder.formatter.result)
		}
	})

	t.Run("nil builder returns same instance", func(t *testing.T) {
		builder := New(ctx, func() {})
		result := builder.ResultWithError(nil)

		if result != builder {
			t.Error("ResultWithError(nil) should return the same builder instance")
		}
	})
}

// TestDoPanicWithInjectedLogger verifies the panic path: the panic record is
// logged, Do returns nil and ResultWithError receives nil. An Info-level
// injected logger is used because the default PanicLevel zap logger re-panics
// after writing (existing bridge behavior, preserved by this package).
func TestDoPanicWithInjectedLogger(t *testing.T) {
	captureLog := NewCaptureLogger()
	gotErr := fmt.Errorf("sentinel")

	builder := New(context.Background(), func() {}, WithLogger(captureLog)).
		ResultWithError(func(err error) string {
			gotErr = err
			return "panic recovered"
		}).
		Action(func(method func()) error {
			panic("boom")
		})

	err := builder.Do()

	if err != nil {
		t.Errorf("Expected Do() to swallow the panic and return nil, got %v", err)
	}
	if gotErr != nil {
		t.Errorf("Expected ResultWithError to receive nil on panic, got %v", gotErr)
	}
	if builder.formatter == nil {
		t.Fatal("formatter should not be nil after Do")
	}
	if builder.formatter.panicValue != "boom" {
		t.Errorf("Expected panicValue 'boom', got %v", builder.formatter.panicValue)
	}
	if builder.formatter.result != "panic recovered" {
		t.Errorf("Expected result 'panic recovered', got %v", builder.formatter.result)
	}
	// Panic record is logged immediately, then the final entry is emitted
	if !captureLog.Contains(`"panic":"boom"`) {
		t.Errorf("Expected panic record in captured output, got: %s", captureLog.GetOutput())
	}
	if !captureLog.Contains(`"result":"panic recovered"`) {
		t.Errorf("Expected result message in captured output, got: %s", captureLog.GetOutput())
	}
	if captureLog.Len() != 2 {
		t.Errorf("Expected 2 log lines (panic record + final entry), got %d: %s", captureLog.Len(), captureLog.GetOutput())
	}
}

// TestLoggerFollowsLogInit verifies that log output destinations follow the
// host's log.Init re-initialization (minimal repro: stale package-level
// loggers kept writing to the old destination).
func TestLoggerFollowsLogInit(t *testing.T) {
	ctx := context.Background()

	// Restore the quiet main sink for subsequent tests
	defer initLogToBuffer("main")

	initLogToBuffer("dest-a")
	err := New(ctx, func() {}).
		AddInfos(func() []interface{} {
			return []interface{}{"marker", "first-destination"}
		}).
		Action(func(method func()) error {
			method()
			return nil
		}).Do()
	if err != nil {
		t.Fatalf("Do() returned unexpected error: %v", err)
	}

	contentA := testSinkOutput("dest-a")
	if !strings.Contains(contentA, "first-destination") {
		t.Errorf("Expected first log in destination A, got: %s", contentA)
	}

	// Re-initialize logging to another destination; a new Do must follow it
	initLogToBuffer("dest-b")
	err = New(ctx, func() {}).
		AddInfos(func() []interface{} {
			return []interface{}{"marker", "second-destination"}
		}).
		Action(func(method func()) error {
			method()
			return nil
		}).Do()
	if err != nil {
		t.Fatalf("Do() returned unexpected error: %v", err)
	}

	contentB := testSinkOutput("dest-b")
	if !strings.Contains(contentB, "second-destination") {
		t.Errorf("Expected second log in destination B after log.Init, got: %s", contentB)
	}

	contentA = testSinkOutput("dest-a")
	if strings.Contains(contentA, "second-destination") {
		t.Errorf("Second log should not appear in old destination A, got: %s", contentA)
	}
}

// TestWithLogger verifies that an injected logger captures all output of a Do
func TestWithLogger(t *testing.T) {
	ctx := context.Background()

	t.Run("captures result and error fields on failure", func(t *testing.T) {
		captureLog := NewCaptureLogger()
		wantErr := fmt.Errorf("boom error")

		err := New(ctx, func() {}, WithLogger(captureLog)).
			ResultWithError(func(err error) string {
				if err != nil {
					return "request failed"
				}
				return "request processed"
			}).
			Action(func(method func()) error {
				return wantErr
			}).Do()

		if err != wantErr {
			t.Errorf("Expected error %v from Do(), got %v", wantErr, err)
		}
		if !captureLog.Contains(`"result":"request failed"`) {
			t.Errorf("Expected result field in captured output, got: %s", captureLog.GetOutput())
		}
		if !captureLog.Contains(`"error":"boom error"`) {
			t.Errorf("Expected error field in captured output, got: %s", captureLog.GetOutput())
		}
	})

	t.Run("captures result field on success", func(t *testing.T) {
		captureLog := NewCaptureLogger()

		err := New(ctx, func() {}, WithLogger(captureLog)).
			ResultWithError(func(err error) string {
				return "request processed"
			}).
			Action(func(method func()) error {
				method()
				return nil
			}).Do()

		if err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}
		if !captureLog.Contains(`"result":"request processed"`) {
			t.Errorf("Expected result field in captured output, got: %s", captureLog.GetOutput())
		}
		if captureLog.Contains(`"error":`) {
			t.Errorf("Expected no error field on success, got: %s", captureLog.GetOutput())
		}
	})

	t.Run("nil logger falls back to lazy resolution", func(t *testing.T) {
		builder := New(ctx, func() {}, WithLogger(nil)).
			Action(func(method func()) error {
				method()
				return nil
			})

		if err := builder.Do(); err != nil {
			t.Errorf("Do() returned unexpected error: %v", err)
		}
		if builder.formatter == nil {
			t.Fatal("formatter should not be nil after Do")
		}
		if builder.formatter.infoLogger == nil || builder.formatter.errorLogger == nil || builder.formatter.panicLogger == nil {
			t.Error("Level loggers should be lazily resolved when no logger is injected")
		}
	})
}

// TestCallSite verifies the callsite field points to the business call site
// where New is invoked (not to component internals), in short
// package/file.go:line form, on both the normal and the panic path.
func TestCallSite(t *testing.T) {
	ctx := context.Background()

	t.Run("final entry carries caller's file and line", func(t *testing.T) {
		captureLog := NewCaptureLogger()

		_, _, line, ok := runtime.Caller(0)
		builder := New(ctx, func() {}, WithLogger(captureLog)) // must stay on the line right after runtime.Caller(0)
		if !ok {
			t.Fatal("runtime.Caller(0) failed")
		}
		err := builder.Action(func(method func()) error {
			method()
			return nil
		}).Do()
		if err != nil {
			t.Fatalf("Do() returned unexpected error: %v", err)
		}

		expected := fmt.Sprintf(`"callsite":"v1/builder_test.go:%d"`, line+1)
		if !captureLog.Contains(expected) {
			t.Errorf("Expected callsite %s in captured output, got: %s", expected, captureLog.GetOutput())
		}
	})

	t.Run("panic record and final entry both carry callsite", func(t *testing.T) {
		captureLog := NewCaptureLogger()

		_, _, line, ok := runtime.Caller(0)
		builder := New(ctx, func() {}, WithLogger(captureLog)) // must stay on the line right after runtime.Caller(0)
		if !ok {
			t.Fatal("runtime.Caller(0) failed")
		}
		err := builder.Action(func(method func()) error {
			panic("boom")
		}).Do()
		if err != nil {
			t.Fatalf("Expected Do() to swallow the panic and return nil, got %v", err)
		}

		expected := fmt.Sprintf(`"callsite":"v1/builder_test.go:%d"`, line+1)
		if n := strings.Count(captureLog.GetOutput(), expected); n != 2 {
			t.Errorf("Expected callsite %s on both log lines, found %d: %s", expected, n, captureLog.GetOutput())
		}
	})
}

// TestConcurrentDo verifies concurrent Do calls are safe under -race
func TestConcurrentDo(t *testing.T) {
	const workers = 20

	t.Run("with injected logger", func(t *testing.T) {
		captureLog := NewCaptureLogger()
		var wg sync.WaitGroup

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				err := New(context.Background(), func() {}, WithLogger(captureLog)).
					AddInfos(func() []interface{} {
						return []interface{}{"gid", id}
					}).
					ResultWithError(func(err error) string {
						return "concurrent ok"
					}).
					Action(func(method func()) error {
						method()
						return nil
					}).Do()
				if err != nil {
					t.Errorf("Do() returned unexpected error: %v", err)
				}
			}(i)
		}
		wg.Wait()

		if captureLog.Len() != workers {
			t.Errorf("Expected %d log lines, got %d", workers, captureLog.Len())
		}
		if !captureLog.Contains(`"result":"concurrent ok"`) {
			t.Errorf("Expected result message in captured output, got: %s", captureLog.GetOutput())
		}
	})

	t.Run("with lazy resolution", func(t *testing.T) {
		// No log.Init during this test: concurrent Do calls only read the
		// global log state (Init/K() synchronization in log/v1 is out of scope).
		var wg sync.WaitGroup

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := New(context.Background(), func() {}).
					ResultWithError(func(err error) string {
						return "lazy ok"
					}).
					Action(func(method func()) error {
						method()
						return nil
					}).Do()
				if err != nil {
					t.Errorf("Do() returned unexpected error: %v", err)
				}
			}()
		}
		wg.Wait()
	})
}
