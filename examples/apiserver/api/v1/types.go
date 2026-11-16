// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package v1 defines the API domain types shared by the example HTTP API server
// (examples/apiserver) and its client SDK example (examples/httpclient). Keeping
// them here mirrors real projects such as eir's standalone api/apiserver/v1
// package, so the wire contract lives in exactly one place.
package v1

// User is the API domain model for a user resource, persisted by the store layer
// and returned to clients.
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CreateUserRequest is the request body accepted by the create endpoint
// (POST /v1/users expects {"name": "..."}).
type CreateUserRequest struct {
	Name string `json:"name"`
}
