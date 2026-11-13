// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	genericserver "github.com/ClessLi/component-base/pkg/generic_server/server"
)

// TestAPIServer demonstrates the full server lifecycle without binding a network
// port: assemble config from options, PrepareRun (which wires store -> service ->
// middleware -> controller through the biz hooks), serve requests through the gin
// router with httptest, then Close to clean up business resources.
func TestAPIServer(t *testing.T) {
	cfg, err := CreateConfigFromOptions(NewOptions())
	if err != nil {
		t.Fatalf("create config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := NewServer(ctx, cfg)
	prepared := srv.PrepareRun()
	defer func() {
		if err := prepared.Close(); err != nil {
			t.Errorf("close server: %v", err)
		}
	}()

	// healthz endpoint is registered by the framework itself.
	doRequest(t, srv, http.MethodGet, "/healthz", nil, http.StatusOK)

	// Create a user.
	rec := doRequest(t, srv, http.MethodPost, "/v1/users", bytes.NewBufferString(`{"name":"alice"}`), http.StatusOK)
	var created User
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created user: %v", err)
	}
	if created.ID != "1" || created.Name != "alice" {
		t.Fatalf("unexpected created user: %+v", created)
	}

	// Get the user back.
	rec = doRequest(t, srv, http.MethodGet, "/v1/users/1", nil, http.StatusOK)
	var got User
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode user: %v", err)
	}
	if got.Name != "alice" {
		t.Fatalf("unexpected user: %+v", got)
	}

	// List users.
	rec = doRequest(t, srv, http.MethodGet, "/v1/users", nil, http.StatusOK)
	var list []*User
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode user list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 user, got %d", len(list))
	}
}

// doRequest sends an in-process request through the server's gin router and
// asserts the response status code.
func doRequest(
	t *testing.T,
	srv *genericserver.GenericAPIServer[ServiceFactory, StoreFactory],
	method, path string,
	body *bytes.Buffer,
	wantStatus int,
) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	rec := httptest.NewRecorder()
	srv.Router.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s: status = %d, want %d, body = %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	return rec
}
