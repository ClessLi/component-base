// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package transport is the transport layer of the HTTP client SDK example. It
// owns the *http.Client and base URL and builds one httpclientv1.Client per
// remote operation, mirroring the eir apiserver client's transport layout.
package transport

import (
	"net/http"
	"sync"
)

// Factory is the entry point of the transport layer, exposing one transport
// per resource. It follows the eir apiserver client convention.
type Factory interface {
	Users() UserTransport
}

// transport implements Factory. Each resource transport is lazily created once
// and cached behind a sync.Once.
type transport struct {
	client  *http.Client
	baseURL string

	onceUser      sync.Once
	singletonUser UserTransport
}

// Compile-time check that transport satisfies the factory contract.
var _ Factory = (*transport)(nil)

func (t *transport) Users() UserTransport {
	t.onceUser.Do(func() {
		if t.singletonUser == nil {
			t.singletonUser = newUserTransport(t)
		}
	})
	return t.singletonUser
}

// NewTransport creates a transport Factory bound to the given HTTP client and
// base URL. A nil client falls back to http.DefaultClient.
func NewTransport(client *http.Client, baseURL string) Factory {
	if client == nil {
		client = http.DefaultClient
	}
	return &transport{
		client:  client,
		baseURL: baseURL,
	}
}
