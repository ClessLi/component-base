// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package code demonstrates how to define error codes for a project.
// This is an example file showing the recommended error code organization pattern.
//
// The error code design follows the SSSMMM pattern:
//
//	SSS - Service ID (3 digits)
//	MMM - Module ID (3 digits)
//
// Error codes are grouped by services and modules for better organization.
//
// Allowed HTTP status codes:
//
//	StatusOK                  = 200 // RFC 7231, 6.3.1
//	StatusBadRequest          = 400 // RFC 7231, 6.5.1
//	StatusUnauthorized        = 401 // RFC 7235, 3.1
//	StatusForbidden           = 403 // RFC 7231, 6.5.3
//	StatusNotFound            = 404 // RFC 7231, 6.5.4
//	StatusRequestTimeout      = 408 // RFC 7231, 6.5.7
//	StatusConflict            = 409 // RFC 7231, 6.5.8
//	StatusTooManyRequests     = 429 // RFC 6585, 4
//	StatusInternalServerError = 500 // RFC 7231, 6.6.1
//
// Error Code Structure:
//
//	Service ID 10: Common base errors (0base.go)
//	Service ID 11: Proxy claim service (1proxy_claim.go)
//	Service ID 12: Proxy controller service (2proxy_controller.go)
//
// Usage:
//
//	// Generate code and documentation
//	go generate
//
//	// Use in your code
//	err := errors.WithCode("example", ErrNotFound, "resource %s not found", id)
package code // import "github.com/ClessLi/component-base/examples/code"
