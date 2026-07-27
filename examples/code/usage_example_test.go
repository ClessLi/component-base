package code

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// captureOutput captures stdout from a function
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestExampleWithCode(t *testing.T) {
	output := captureOutput(ExampleWithCode)
	// The output includes full stack trace with file path and function name
	// Use regex to match since line numbers may change
	if !bytes.Contains([]byte(output), []byte("Check the URL path and try again")) ||
		!bytes.Contains([]byte(output), []byte("user 12345 not found")) ||
		!bytes.Contains([]byte(output), []byte("codeexample:100006")) {
		t.Errorf("ExampleWithCode() output = %q, missing expected content", output)
	}
}

func TestExampleNew(t *testing.T) {
	output := captureOutput(ExampleNew)
	expected := "invalid email format\n"
	if output != expected {
		t.Errorf("ExampleNew() output = %q, want %q", output, expected)
	}
}

func TestExampleErrorf(t *testing.T) {
	output := captureOutput(ExampleErrorf)
	expected := "connection timeout after 30 seconds\n"
	if output != expected {
		t.Errorf("ExampleErrorf() output = %q, want %q", output, expected)
	}
}

func TestExampleWrap(t *testing.T) {
	output := captureOutput(ExampleWrap)
	expected := "failed to connect to database\n"
	if output != expected {
		t.Errorf("ExampleWrap() output = %q, want %q", output, expected)
	}
}

func TestExampleWrapf(t *testing.T) {
	output := captureOutput(ExampleWrapf)
	expected := "failed to write data to /var/log\n"
	if output != expected {
		t.Errorf("ExampleWrapf() output = %q, want %q", output, expected)
	}
}

func TestExampleWrapC(t *testing.T) {
	output := captureOutput(ExampleWrapC)
	expected := "Provide a valid authentication token\n"
	if output != expected {
		t.Errorf("ExampleWrapC() output = %q, want %q", output, expected)
	}
}

func TestExampleWithStack(t *testing.T) {
	output := captureOutput(ExampleWithStack)
	expected := "Contact system administrator with error details for investigation\n"
	if output != expected {
		t.Errorf("ExampleWithStack() output = %q, want %q", output, expected)
	}
}

func TestExampleWithMessage(t *testing.T) {
	output := captureOutput(ExampleWithMessage)
	expected := "failed to parse request body\n"
	if output != expected {
		t.Errorf("ExampleWithMessage() output = %q, want %q", output, expected)
	}
}

func TestExampleWithMessagef(t *testing.T) {
	output := captureOutput(ExampleWithMessagef)
	expected := "failed to encode JSON response\n"
	if output != expected {
		t.Errorf("ExampleWithMessagef() output = %q, want %q", output, expected)
	}
}

func TestExampleCause(t *testing.T) {
	output := captureOutput(ExampleCause)
	expected := "disk full\n"
	if output != expected {
		t.Errorf("ExampleCause() output = %q, want %q", output, expected)
	}
}

func TestExampleErrorChain(t *testing.T) {
	output := captureOutput(ExampleErrorChain)
	expected := "Root cause: database connection failed\nFull error: service unavailable\n"
	if output != expected {
		t.Errorf("ExampleErrorChain() output = %q, want %q", output, expected)
	}
}

func TestExampleMultipleErrors(t *testing.T) {
	output := captureOutput(ExampleMultipleErrors)
	expected := "Error: OK\nError: Check the URL path and try again\nError: Check input parameters and ensure they meet validation requirements\nError: Contact system administrator to resolve database connectivity issue\n"
	if output != expected {
		t.Errorf("ExampleMultipleErrors() output = %q, want %q", output, expected)
	}
}

func TestExampleErrorWithReference(t *testing.T) {
	output := captureOutput(ExampleErrorWithReference)
	expected := "Error: Contact system administrator with error details for investigation\nError: Check request body and ensure it matches the expected format\n"
	if output != expected {
		t.Errorf("ExampleErrorWithReference() output = %q, want %q", output, expected)
	}
}

func TestExampleNamespace(t *testing.T) {
	output := captureOutput(ExampleNamespace)
	expected := "Namespace: codeexample\nError namespace: codeexample\nError code: 100001\n"
	if output != expected {
		t.Errorf("ExampleNamespace() output = %q, want %q", output, expected)
	}
}

func TestExampleErrorDetails(t *testing.T) {
	output := captureOutput(ExampleErrorDetails)
	expected := "Namespace: codeexample\nCode: 100006\nHTTPStatus: 404\nExt: Check the URL path and try again\nReference: https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/examples/error_code_generated.md#L22\n"
	if output != expected {
		t.Errorf("ExampleErrorDetails() output = %q, want %q", output, expected)
	}
}

func TestExampleAllErrorCodes(t *testing.T) {
	output := captureOutput(ExampleAllErrorCodes)
	// Verify that output contains expected patterns
	if output == "" {
		t.Error("ExampleAllErrorCodes() output is empty, expected error code outputs")
	}
	// Check a few key error codes are present
	expectedPatterns := []string{
		"Code 100001: OK",
		"Code 100002: Contact system administrator with error details for investigation",
		"Code 100003: Check request body and ensure it matches the expected format",
	}
	for _, pattern := range expectedPatterns {
		if !bytes.Contains([]byte(output), []byte(pattern)) {
			t.Errorf("ExampleAllErrorCodes() output missing pattern %q", pattern)
		}
	}
}

func TestExampleNewAggregate(t *testing.T) {
	output := captureOutput(ExampleNewAggregate)
	expected := "[Check the URL path and try again, Contact system administrator to resolve database connectivity issue]\n"
	if output != expected {
		t.Errorf("ExampleNewAggregate() output = %q, want %q", output, expected)
	}
}

func TestExampleFilterOut(t *testing.T) {
	output := captureOutput(ExampleFilterOut)
	expected := "Filtered: Contact system administrator to resolve database connectivity issue\n"
	if output != expected {
		t.Errorf("ExampleFilterOut() output = %q, want %q", output, expected)
	}
}

func TestExampleFlatten(t *testing.T) {
	output := captureOutput(ExampleFlatten)
	expected := "Flattened count: 3\n"
	if output != expected {
		t.Errorf("ExampleFlatten() output = %q, want %q", output, expected)
	}
}

func TestExampleReduce(t *testing.T) {
	output := captureOutput(ExampleReduce)
	expected := "Reduced: OK\n"
	if output != expected {
		t.Errorf("ExampleReduce() output = %q, want %q", output, expected)
	}
}

func TestExampleAggregateGoroutines(t *testing.T) {
	output := captureOutput(ExampleAggregateGoroutines)
	expected := "Aggregated errors collected\n"
	if output != expected {
		t.Errorf("ExampleAggregateGoroutines() output = %q, want %q", output, expected)
	}
}
