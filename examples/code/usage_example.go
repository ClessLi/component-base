// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package code demonstrates how to use error codes and wrapper functions.
package code

import (
	"fmt"

	"github.com/ClessLi/component-base/pkg/errors"
)

// ExampleWithCode demonstrates creating errors with code using wrapper function.
//
// Note: The output shows the error code's external message (Ext field), not the
// custom message passed to WithCode. Use %+v to see full error details.
//
// Example output:
//
//	Check the URL path and try again
//	Error details: user 12345 not found - #0 (codeexample:100006) Check the URL path and try again
func ExampleWithCode() {
	// Create error with auto-injected namespace
	err := WithCode(ErrPageNotFound, "user %s not found", "12345")
	fmt.Println(err)

	// Print detailed error information
	fmt.Printf("Error details: %+v\n", err)

	// Output:
	// Check the URL path and try again
	// Error details: user 12345 not found - #0 (codeexample:100006) Check the URL path and try again
}

// ExampleNew demonstrates creating simple errors.
//
// Note: New creates a plain error without error code, similar to fmt.Errorf.
// It records a stack trace at the point it was called.
//
// Example output:
//
//	invalid email format
func ExampleNew() {
	err := New("invalid email format")
	fmt.Println(err)

	// Output:
	// invalid email format
}

// ExampleErrorf demonstrates creating formatted errors.
//
// Note: Errorf creates a plain formatted error without error code, similar to fmt.Errorf.
// It records a stack trace at the point it was called.
//
// Example output:
//
//	connection timeout after 30 seconds
func ExampleErrorf() {
	err := Errorf("connection timeout after %d seconds", 30)
	fmt.Println(err)

	// Output:
	// connection timeout after 30 seconds
}

// ExampleWrap demonstrates wrapping errors with context.
//
// Note: Wrap annotates the original error with a new message and adds a stack trace.
// The original error is preserved in the error chain.
//
// Example output:
//
//	failed to connect to database
func ExampleWrap() {
	originalErr := fmt.Errorf("network timeout")
	err := Wrap(originalErr, "failed to connect to database")
	fmt.Println(err)

	// Output:
	// failed to connect to database
}

// ExampleWrapf demonstrates wrapping errors with formatted message.
//
// Note: Wrapf annotates the original error with a formatted message and adds a stack trace.
// The original error is preserved in the error chain.
//
// Example output:
//
//	failed to write data to /var/log
func ExampleWrapf() {
	originalErr := fmt.Errorf("disk full")
	err := Wrapf(originalErr, "failed to write data to %s", "/var/log")
	fmt.Println(err)

	// Output:
	// failed to write data to /var/log
}

// ExampleWrapC demonstrates wrapping errors with code and namespace.
//
// Note: The output shows the error code's external message (Ext field), not the
// custom message passed to WrapC. The namespace is automatically injected.
//
// Example output:
//
//	Provide a valid authentication token
func ExampleWrapC() {
	originalErr := fmt.Errorf("authentication failed")
	err := WrapC(originalErr, ErrTokenInvalid, "invalid token for user %s", "admin")
	fmt.Println(err)

	// Output:
	// Provide a valid authentication token
}

// ExampleWithStack demonstrates adding stack trace to error.
//
// Note: The output shows the error code's external message (Ext field). The stack
// trace is included when using %+v format for detailed debugging.
//
// Example output:
//
//	Contact system administrator with error details for investigation
func ExampleWithStack() {
	err := WithCode(ErrUnknown, "something went wrong")
	err = WithStack(err)
	fmt.Println(err)

	// Output:
	// Contact system administrator with error details for investigation
}

// ExampleWithMessage demonstrates adding message to error.
//
// Note: Unlike other functions, WithMessage replaces the external message with the
// provided message, making it suitable for user-facing error messages.
//
// Example output:
//
//	failed to parse request body
func ExampleWithMessage() {
	err := WithCode(ErrBind, "invalid JSON")
	err = WithMessage(err, "failed to parse request body")
	fmt.Println(err)

	// Output:
	// failed to parse request body
}

// ExampleWithMessagef demonstrates adding formatted message to error.
//
// Note: Unlike other functions, WithMessagef replaces the external message with the
// formatted message, making it suitable for user-facing error messages.
//
// Example output:
//
//	failed to encode JSON response
func ExampleWithMessagef() {
	err := WithCode(ErrEncodingFailed, "encoding error")
	err = WithMessagef(err, "failed to encode %s response", "JSON")
	fmt.Println(err)

	// Output:
	// failed to encode JSON response
}

// ExampleCause demonstrates retrieving the root cause of error.
//
// Note: Cause returns the underlying cause of the error chain. For wrapped errors,
// this returns the original error that started the chain.
//
// Example output:
//
//	disk full
func ExampleCause() {
	originalErr := fmt.Errorf("disk full")
	err := WrapC(originalErr, ErrDatabase, "failed to save data")
	cause := Cause(err)
	fmt.Println(cause)

	// Output:
	// disk full
}

// ExampleErrorChain demonstrates error chain handling.
//
// Note: Shows how errors are chained and how Cause retrieves the root cause.
// Wrap preserves the original error in the chain.
//
// Example output:
//
//	Root cause: database connection failed
//	Full error: service unavailable
func ExampleErrorChain() {
	// Create a chain of errors
	err := New("database connection failed")
	err = Wrap(err, "timeout after 30s")
	err = Wrap(err, "service unavailable")

	// Get the root cause
	cause := Cause(err)
	fmt.Printf("Root cause: %v\n", cause)
	fmt.Printf("Full error: %v\n", err)

	// Output:
	// Root cause: database connection failed
	// Full error: service unavailable
}

// ExampleMultipleErrors demonstrates handling multiple error codes.
//
// Note: Each error displays its external message (Ext field), which is the user-safe
// message mapped to the error code.
//
// Example output:
//
//	Error: OK
//	Error: Check the URL path and try again
//	Error: Check input parameters and ensure they meet validation requirements
//	Error: Contact system administrator to resolve database connectivity issue
func ExampleMultipleErrors() {
	errors := []error{
		WithCode(ErrSuccess, "operation completed"),
		WithCode(ErrPageNotFound, "resource not found"),
		WithCode(ErrValidation, "invalid input"),
		WithCode(ErrDatabase, "connection failed"),
	}

	for _, err := range errors {
		fmt.Printf("Error: %v\n", err)
	}

	// Output:
	// Error: OK
	// Error: Check the URL path and try again
	// Error: Check input parameters and ensure they meet validation requirements
	// Error: Contact system administrator to resolve database connectivity issue
}

// ExampleErrorWithReference demonstrates error with reference URL.
//
// Note: Shows how errors display their external messages. The reference URL is
// stored internally and can be accessed via errors.ParseCoder().
//
// Example output:
//
//	Error: Contact system administrator with error details for investigation
//	Error: Check request body and ensure it matches the expected format
func ExampleErrorWithReference() {
	// Error with custom reference message
	err := WithCode(ErrUnknown, "internal server error occurred")
	fmt.Printf("Error: %v\n", err)

	// Error with reference URL (from code definition)
	err = WithCode(ErrBind, "request body parsing failed")
	fmt.Printf("Error: %v\n", err)

	// Output:
	// Error: Contact system administrator with error details for investigation
	// Error: Check request body and ensure it matches the expected format
}

// ExampleNamespace demonstrates namespace isolation.
//
// Note: The namespace is automatically injected from the code package and ensures
// error codes are isolated across different projects.
//
// Example output:
//
//	Namespace: codeexample
//	Error namespace: codeexample
//	Error code: 100001
func ExampleNamespace() {
	// Namespace is automatically injected from code package
	fmt.Printf("Namespace: %s\n", Namespace)

	// Create error with namespace
	err := WithCode(ErrSuccess, "success")

	// Parse the error to verify namespace is correctly injected
	coder := errors.ParseCoder(err)
	if coder != nil {
		fmt.Printf("Error namespace: %s\n", coder.Namespace())
		fmt.Printf("Error code: %d\n", coder.Code())
	}

	// Output:
	// Namespace: codeexample
	// Error namespace: codeexample
	// Error code: 100001
}

// ExampleErrorDetails demonstrates extracting detailed error information.
//
// This example shows how to extract all error details including:
// - Namespace: The error code namespace
// - Code: The numeric error code
// - HTTPStatus: The HTTP status code
// - Ext: The external (user-safe) error message
// - Reference: The reference URL for documentation
func ExampleErrorDetails() {
	// Create an error
	err := WithCode(ErrPageNotFound, "resource %s not found", "/api/users")

	// Parse the error to get details
	coder := errors.ParseCoder(err)
	if coder != nil {
		fmt.Printf("Namespace: %s\n", coder.Namespace())
		fmt.Printf("Code: %d\n", coder.Code())
		fmt.Printf("HTTPStatus: %d\n", coder.HTTPStatus())
		fmt.Printf("Ext: %s\n", coder.String())
		fmt.Printf("Reference: %s\n", coder.Reference())
	}

	// Output:
	// Namespace: codeexample
	// Code: 100006
	// HTTPStatus: 404
	// Ext: Check the URL path and try again
	// Reference: https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/examples/error_code_generated.md#L22
}

// ExampleAllErrorCodes demonstrates all defined error codes.
//
// Note: This example iterates through all defined error codes and displays their
// external messages. Useful for verifying all error codes are properly registered.
//
// Example output:
//
//	Code 100001: OK
//	Code 100002: Contact system administrator with error details for investigation
//	Code 100003: Check request body and ensure it matches the expected format
//	... (and more error codes)
func ExampleAllErrorCodes() {
	// Basic errors
	testErrorCode(ErrSuccess, "OK")
	testErrorCode(ErrUnknown, "unknown error")
	testErrorCode(ErrBind, "bind error")
	testErrorCode(ErrValidation, "validation error")
	testErrorCode(ErrTokenInvalid, "invalid token")
	testErrorCode(ErrPageNotFound, "page not found")
	testErrorCode(ErrRequestTimeout, "request timeout")
	testErrorCode(ErrInvalidParameter, "invalid parameter")

	// Database errors
	testErrorCode(ErrDatabase, "database error")

	// Auth errors
	testErrorCode(ErrEncrypt, "encrypt error")
	testErrorCode(ErrSignatureInvalid, "invalid signature")
	testErrorCode(ErrExpired, "token expired")
	testErrorCode(ErrInvalidAuthHeader, "invalid auth header")
	testErrorCode(ErrMissingHeader, "missing header")
	testErrorCode(ErrUserOrPasswordIncorrect, "incorrect credentials")
	testErrorCode(ErrPermissionDenied, "permission denied")
	testErrorCode(ErrAuthnClientInitFailed, "auth client init failed")
	testErrorCode(ErrAuthClientNotInit, "auth client not init")
	testErrorCode(ErrConnToAuthServerFailed, "auth server connection failed")

	// Encode/Decode errors
	testErrorCode(ErrEncodingFailed, "encoding failed")
	testErrorCode(ErrDecodingFailed, "decoding failed")
	testErrorCode(ErrInvalidJSON, "invalid JSON")
	testErrorCode(ErrEncodingJSON, "JSON encoding failed")
	testErrorCode(ErrDecodingJSON, "JSON decoding failed")
	testErrorCode(ErrInvalidYaml, "invalid YAML")
	testErrorCode(ErrEncodingYaml, "YAML encoding failed")
	testErrorCode(ErrDecodingYaml, "YAML decoding failed")
}

// testErrorCode is a helper function to test error code creation.
func testErrorCode(code int, description string) {
	err := WithCode(code, "%s", description)
	fmt.Printf("Code %d: %v\n", code, err)
}

// ExampleNewAggregate demonstrates creating an aggregate of errors.
//
// Note: Multiple errors are combined into a single Aggregate interface. The output
// shows all external messages joined together in brackets.
//
// Example output:
//
//	[Check the URL path and try again, Contact system administrator to resolve database connectivity issue]
func ExampleNewAggregate() {
	// Create multiple errors
	err1 := WithCode(ErrPageNotFound, "page /users not found")
	err2 := WithCode(ErrDatabase, "connection timeout")

	// Aggregate them into a single error
	agg := NewAggregate([]error{err1, err2})
	if agg != nil {
		fmt.Println(agg)
	}

	// Output:
	// [Check the URL path and try again, Contact system administrator to resolve database connectivity issue]
}

// ExampleFilterOut demonstrates filtering errors from an aggregate.
//
// Note: FilterOut removes errors matching specified criteria from the aggregate.
// The remaining errors are returned as a new aggregate.
//
// Example output:
//
//	Filtered: Contact system administrator to resolve database connectivity issue
func ExampleFilterOut() {
	// Create multiple errors
	err1 := WithCode(ErrPageNotFound, "page not found")
	err2 := WithCode(ErrDatabase, "connection timeout")

	// Aggregate them
	agg := NewAggregate([]error{err1, err2})

	// Filter out ErrPageNotFound errors
	filtered := FilterOut(agg, func(err error) bool {
		return errors.IsCode(err, Namespace, ErrPageNotFound)
	})

	fmt.Printf("Filtered: %v\n", filtered)

	// Output:
	// Filtered: Contact system administrator to resolve database connectivity issue
}

// ExampleFlatten demonstrates flattening nested aggregates.
//
// Example output:
//
//	Flattened count: 3
func ExampleFlatten() {
	// Create nested aggregates
	err1 := WithCode(ErrPageNotFound, "not found")
	err2 := WithCode(ErrDatabase, "db error")
	err3 := WithCode(ErrValidation, "invalid input")

	// Create nested structure
	inner := NewAggregate([]error{err1, err2})
	outer := NewAggregate([]error{inner, err3})

	// Flatten the nested aggregate
	flattened := Flatten(outer)
	fmt.Printf("Flattened count: %d\n", len(flattened.Errors()))

	// Output:
	// Flattened count: 3
}

// ExampleReduce demonstrates reducing an aggregate to a single error.
//
// Example output:
//
//	Reduced: OK
func ExampleReduce() {
	// Create single error aggregate
	err := WithCode(ErrSuccess, "success")
	agg := NewAggregate([]error{err})

	// Reduce to single error
	reduced := Reduce(agg)
	fmt.Printf("Reduced: %v\n", reduced)

	// Output:
	// Reduced: OK
}

// ExampleAggregateGoroutines demonstrates aggregating errors from goroutines.
//
// Example output:
//
//	Aggregated errors collected
func ExampleAggregateGoroutines() {
	// Run multiple functions in parallel
	agg := AggregateGoroutines(
		func() error {
			return WithCode(ErrPageNotFound, "not found")
		},
		func() error {
			return nil // Success
		},
		func() error {
			return WithCode(ErrDatabase, "db error")
		},
	)

	if agg != nil {
		fmt.Println("Aggregated errors collected")
	} else {
		fmt.Println("No errors")
	}

	// Output:
	// Aggregated errors collected
}
