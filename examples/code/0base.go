// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package code demonstrates how to define error codes for a project.
// This is an example file showing the recommended error code organization pattern.
//
// Error Code Structure:
// - Format: SSSMMM (SSS=Service ID 3位, MMM=Module ID 3位)
// - Service ID 10: Common base errors (0base.go)
// - Service ID 11: Proxy claim service (1proxy_claim.go)
// - Service ID 12: Proxy controller service (2proxy_controller.go)
//
// Usage:
//
//	// Generate code and documentation
//	go generate
//
//	// Use in your code
//	err := errors.WithCode(code.Namespace, ErrNotFound, "resource %s not found", id)
package code

//go:generate codegen -type=int -fullname=CodeExample -namespace=codeexample -doc-output=../../docs/guide/zh-CN/examples/error_code_generated.md -wrapper

// Common: basic errors.
// Code must start with 100001.
// Refer to https://github.com/ClessLi/component-base/pkg/errors for details.
// Service ID: 10, Module ID: 00
const (
	// ErrSuccess - 200: OK.
	// Reference: https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/errors/success.md
	ErrSuccess int = iota + 100001

	// ErrUnknown - 500: Contact system administrator with error details for investigation.
	// Reference Message: Internal server error occurred, check logs for details
	// Reference: https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/errors/unknown.md
	ErrUnknown

	// ErrBind - 400: Check request body and ensure it matches the expected format.
	// Reference: https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/errors/bind.md
	ErrBind

	// ErrValidation - 400: Check input parameters and ensure they meet validation requirements.
	// Reference Message: Input validation failed, check parameter constraints
	ErrValidation

	// ErrTokenInvalid - 401: Provide a valid authentication token.
	ErrTokenInvalid

	// ErrPageNotFound - 404: Check the URL path and try again.
	ErrPageNotFound

	// ErrRequestTimeout - 408: Retry request or check network connection.
	// Reference Message: Request timeout, check network connectivity and retry
	ErrRequestTimeout

	// ErrInvalidParameter - 400: Check input parameters and ensure they are valid.
	ErrInvalidParameter
)

// Common: database errors.
// Code must start with 100101.
// Service ID: 10, Module ID: 01
const (
	// ErrDatabase - 500: Contact system administrator to resolve database connectivity issue.
	// Reference: https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/errors/database.md
	ErrDatabase int = iota + 100101
)

// Common: authorization and authentication errors.
// Code must start with 100301.
// Service ID: 10, Module ID: 03
const (
	// ErrEncrypt - 401: Provide a valid password that can be encrypted.
	ErrEncrypt int = iota + 100301

	// ErrSignatureInvalid - 401: Provide a request with valid signature.
	ErrSignatureInvalid

	// ErrExpired - 401: Refresh authentication token and try again.
	ErrExpired

	// ErrInvalidAuthHeader - 401: Provide request with valid authorization header.
	ErrInvalidAuthHeader

	// ErrMissingHeader - 401: Provide request with valid Authorization header.
	ErrMissingHeader

	// ErrUserOrPasswordIncorrect - 401: Provide valid username and password credentials.
	ErrUserOrPasswordIncorrect

	// ErrPermissionDenied - 403: Contact system administrator for required permissions.
	ErrPermissionDenied

	// ErrAuthnClientInitFailed - 500: Contact system administrator to initialize authentication client.
	ErrAuthnClientInitFailed

	// ErrAuthClientNotInit - 500: Contact system administrator to initialize authentication and authorization clients.
	ErrAuthClientNotInit

	// ErrConnToAuthServerFailed - 500: Contact system administrator to check authentication server connectivity.
	ErrConnToAuthServerFailed
)

// Common: encode/decode errors.
// Code must start with 100401.
// Service ID: 10, Module ID: 04
const (
	// ErrEncodingFailed - 500: Check data format and ensure it can be properly encoded.
	// Reference: https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/errors/encoding.md
	ErrEncodingFailed int = iota + 100401

	// ErrDecodingFailed - 500: Check data format and ensure it can be properly decoded.
	ErrDecodingFailed

	// ErrInvalidJSON - 500: Provide data in valid JSON format.
	ErrInvalidJSON

	// ErrEncodingJSON - 500: Check JSON data and ensure it can be properly encoded.
	ErrEncodingJSON

	// ErrDecodingJSON - 500: Check JSON data and ensure it can be properly decoded.
	ErrDecodingJSON

	// ErrInvalidYaml - 500: Provide data in valid YAML format.
	ErrInvalidYaml

	// ErrEncodingYaml - 500: Check YAML data and ensure it can be properly encoded.
	ErrEncodingYaml

	// ErrDecodingYaml - 500: Check YAML data and ensure it can be properly decoded.
	ErrDecodingYaml
)
