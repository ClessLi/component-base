// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package apiserver

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"

	apiv1 "github.com/ClessLi/component-base/examples/apiserver/api/v1"
	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	methodlog "github.com/ClessLi/component-base/pkg/methodlog/v1"
	"go.uber.org/zap"
)

// The constants below pin the lines of the methodlog.New calls in middleware.go
// (Get/List/Create). The tests verify the recorded "callsite" field points at
// those business call sites; update the constants when middleware.go is edited.
const (
	getNewLine    = 48
	listNewLine   = 69
	createNewLine = 84
)

func wantCallSite(line int) string {
	return fmt.Sprintf(`"callsite":"apiserver/middleware.go:%d"`, line)
}

// In-memory log sink: zap keeps file sinks open for the process lifetime, so
// tests capture global log/v1 output through a registered URL sink instead of
// real files (which would stay locked on Windows).
var (
	exampleLogOnce sync.Once
	exampleLogMu   sync.Mutex
	exampleLogBuf  bytes.Buffer
)

type exampleLogSink struct{}

func (exampleLogSink) Write(p []byte) (int, error) {
	exampleLogMu.Lock()
	defer exampleLogMu.Unlock()
	return exampleLogBuf.Write(p)
}
func (exampleLogSink) Sync() error  { return nil }
func (exampleLogSink) Close() error { return nil }

// captureExampleLog routes global log/v1 output (JSON format for stable
// assertions) into an in-memory buffer and restores default logging on cleanup.
func captureExampleLog(t *testing.T) {
	t.Helper()
	exampleLogOnce.Do(func() {
		err := zap.RegisterSink("examplelogtest", func(*url.URL) (zap.Sink, error) {
			return exampleLogSink{}, nil
		})
		if err != nil {
			panic(err)
		}
	})
	exampleLogMu.Lock()
	exampleLogBuf.Reset()
	exampleLogMu.Unlock()

	opts := logV1.NewOptions()
	opts.Format = "json"
	opts.OutputPaths = []string{"examplelogtest://buf"}
	opts.ErrorOutputPaths = opts.OutputPaths
	logV1.Init(opts)
	t.Cleanup(func() { logV1.Init(logV1.NewOptions()) })
}

func exampleLogOutput() string {
	exampleLogMu.Lock()
	defer exampleLogMu.Unlock()
	return exampleLogBuf.String()
}

func assertLogContains(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("expected log to contain %s, got: %s", want, out)
		}
	}
}

// assertInternalCaller checks the backend's own caller field stays inside the
// logging component (never the business file); business localization relies on
// the callsite field instead.
func assertInternalCaller(t *testing.T, out string) {
	t.Helper()
	if !strings.Contains(out, `"caller":"`) {
		t.Errorf("expected backend caller field in log, got: %s", out)
	}
	if strings.Contains(out, `"caller":"apiserver/middleware.go`) {
		t.Errorf("backend caller must not point at the business file, got: %s", out)
	}
}

// TestLoggingMiddlewareRecords verifies the logging middleware's records are
// well-formed (level, method, result/error, extracted fields, no stray
// error/panic on success) and that the callsite field points at the business
// call line in middleware.go on success, error and panic paths.
// Note: the method field records the interface method value name (e.g.
// UserService.Get-fm) because the middleware wraps interface methods.
func TestLoggingMiddlewareRecords(t *testing.T) {
	t.Run("create success record", func(t *testing.T) {
		captureExampleLog(t)
		ctx := methodlog.WithClientIP(context.Background(), "203.0.113.7")
		users := loggingMiddleware(NewServiceFactory(NewStoreFactory())).Users()

		if _, err := users.Create(ctx, "alice"); err != nil {
			t.Fatalf("Create: %v", err)
		}
		out := exampleLogOutput()
		assertLogContains(t, out,
			`"level":"INFO"`,
			`UserService.Create`,
			`"name":"alice"`,
			`"clientIp":"203.0.113.7"`,
			`"result":"created user alice (1)"`,
			`"took":`,
			wantCallSite(createNewLine),
		)
		if strings.Contains(out, `"error":`) || strings.Contains(out, `"panic":`) {
			t.Errorf("success record must not carry error/panic fields, got: %s", out)
		}
		assertInternalCaller(t, out)
	})

	t.Run("get success record", func(t *testing.T) {
		captureExampleLog(t)
		ctx := methodlog.WithClientIP(context.Background(), "203.0.113.7")
		users := loggingMiddleware(NewServiceFactory(NewStoreFactory())).Users()
		if _, err := users.Create(ctx, "alice"); err != nil {
			t.Fatalf("Create: %v", err)
		}
		exampleLogMu.Lock()
		exampleLogBuf.Reset() // drop the create record, keep only the get record
		exampleLogMu.Unlock()

		if _, err := users.Get(ctx, "1"); err != nil {
			t.Fatalf("Get: %v", err)
		}
		out := exampleLogOutput()
		assertLogContains(t, out,
			`"level":"INFO"`,
			`UserService.Get`,
			`"id":"1"`,
			`"result":"user alice (1)"`,
			wantCallSite(getNewLine),
		)
		if strings.Contains(out, `"error":`) || strings.Contains(out, `"panic":`) {
			t.Errorf("success record must not carry error/panic fields, got: %s", out)
		}
		assertInternalCaller(t, out)
	})

	t.Run("list success record", func(t *testing.T) {
		captureExampleLog(t)
		users := loggingMiddleware(NewServiceFactory(NewStoreFactory())).Users()

		if _, err := users.List(context.Background()); err != nil {
			t.Fatalf("List: %v", err)
		}
		out := exampleLogOutput()
		assertLogContains(t, out,
			`"level":"INFO"`,
			`UserService.List`,
			`"clientIp":"unknown"`,
			`"result":"0 users"`,
			wantCallSite(listNewLine),
		)
		assertInternalCaller(t, out)
	})

	t.Run("get error record", func(t *testing.T) {
		captureExampleLog(t)
		users := loggingMiddleware(NewServiceFactory(NewStoreFactory())).Users()

		if _, err := users.Get(context.Background(), "404"); err == nil {
			t.Fatal("expected not-found error from Get")
		}
		out := exampleLogOutput()
		assertLogContains(t, out,
			`"level":"ERROR"`,
			`UserService.Get`,
			`"id":"404"`,
			`"error":"user 404 not found"`,
			`"result":"not found"`,
			wantCallSite(getNewLine),
		)
		if strings.Contains(out, `"panic":`) {
			t.Errorf("error record must not carry a panic field, got: %s", out)
		}
		assertInternalCaller(t, out)
	})

	t.Run("panic record keeps business callsite", func(t *testing.T) {
		captureExampleLog(t)
		// The default panic-level logger re-panics after writing the record; zap
		// panics with the log entry message (empty via the go-kit bridge), not
		// the original panic value, so only assert that a re-panic happened.
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected the default panic-level logger to re-panic after writing the record")
				return
			}
			out := exampleLogOutput()
			assertLogContains(t, out,
				`"level":"PANIC"`,
				`"panic":"boom"`,
				wantCallSite(getNewLine),
			)
			assertInternalCaller(t, out)
		}()
		_, _ = loggingMiddleware(panicUserFactory{}).Users().Get(context.Background(), "1")
		t.Error("expected the panic to propagate out of Do")
	})
}

// panicUserFactory is a stub factory whose Get always panics, used to verify
// the panic record's fields and callsite.
type panicUserFactory struct{}

func (panicUserFactory) Users() UserService { return panicUserService{} }

type panicUserService struct{}

func (panicUserService) Get(context.Context, string) (*apiv1.User, error)    { panic("boom") }
func (panicUserService) List(context.Context) ([]*apiv1.User, error)         { return nil, nil }
func (panicUserService) Create(context.Context, string) (*apiv1.User, error) { return nil, nil }
