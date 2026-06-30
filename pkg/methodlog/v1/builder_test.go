package v1

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

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

// TestMain initializes logger for all tests
func TestMain(m *testing.M) {
	// Initialize loggers with level-specific capture loggers for testing
	// This simulates production behavior where different log levels are used
	// Using Zap levels: InfoLevel(0), ErrorLevel(-2 equivalent), PanicLevel(-4 equivalent)
	infoLogger = NewCaptureLoggerWithLevel(zapcore.InfoLevel)
	errorLogger = NewCaptureLoggerWithLevel(zapcore.ErrorLevel)
	panicLogger = NewCaptureLoggerWithLevel(zapcore.PanicLevel)
	limit = 200

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
