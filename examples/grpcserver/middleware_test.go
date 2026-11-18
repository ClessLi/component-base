// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"testing"

	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/peer"
)

// The constants below pin the lines of the methodlog.New calls in middleware.go
// (Echo/Download). The tests verify the recorded "callsite" field points at
// those business call sites; update the constants when middleware.go is edited.
const (
	echoNewLine     = 53
	downloadNewLine = 71
)

func wantCallSite(line int) string {
	return fmt.Sprintf(`"callsite":"grpcserver/middleware.go:%d"`, line)
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
	if strings.Contains(out, `"caller":"grpcserver/middleware.go`) {
		t.Errorf("backend caller must not point at the business file, got: %s", out)
	}
}

// peerContext builds a context carrying gRPC peer information so the
// GRPCFieldExtractor's clientIp extraction is exercised.
func peerContext() context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("198.51.100.9"), Port: 4242},
	})
}

// TestLoggingMiddlewareRecords verifies the logging middleware's records are
// well-formed (level, method, result/error, extracted peer fields, no stray
// error/panic on success) and that the callsite field points at the business
// call line in middleware.go on success, error and panic paths.
// Note: the method field records the interface method value name (e.g.
// DemoService.Echo-fm) because the middleware wraps interface methods.
func TestLoggingMiddlewareRecords(t *testing.T) {
	t.Run("echo success record", func(t *testing.T) {
		captureExampleLog(t)
		demo := loggingMiddleware(NewServiceFactory(NewStoreFactory())).Demo()

		if _, err := demo.Echo(peerContext(), "hello"); err != nil {
			t.Fatalf("Echo: %v", err)
		}
		out := exampleLogOutput()
		assertLogContains(t, out,
			`"level":"INFO"`,
			`DemoService.Echo`,
			`"message":"hello"`,
			`"clientIp":"198.51.100.9:4242"`,
			`"result":"echo: hello"`,
			`"took":`,
			wantCallSite(echoNewLine),
		)
		if strings.Contains(out, `"error":`) || strings.Contains(out, `"panic":`) {
			t.Errorf("success record must not carry error/panic fields, got: %s", out)
		}
		assertInternalCaller(t, out)
	})

	t.Run("download success record", func(t *testing.T) {
		captureExampleLog(t)
		demo := loggingMiddleware(NewServiceFactory(NewStoreFactory())).Demo()

		data, err := demo.Download(peerContext(), sampleKey)
		if err != nil {
			t.Fatalf("Download: %v", err)
		}
		out := exampleLogOutput()
		assertLogContains(t, out,
			`"level":"INFO"`,
			`DemoService.Download`,
			`"key":"greeting"`,
			fmt.Sprintf(`"result":"%d bytes"`, len(data)),
			wantCallSite(downloadNewLine),
		)
		if strings.Contains(out, `"error":`) || strings.Contains(out, `"panic":`) {
			t.Errorf("success record must not carry error/panic fields, got: %s", out)
		}
		assertInternalCaller(t, out)
	})

	t.Run("download error record", func(t *testing.T) {
		captureExampleLog(t)
		demo := loggingMiddleware(NewServiceFactory(NewStoreFactory())).Demo()

		if _, err := demo.Download(context.Background(), "missing"); err == nil {
			t.Fatal("expected not-found error from Download")
		}
		out := exampleLogOutput()
		assertLogContains(t, out,
			`"level":"ERROR"`,
			`DemoService.Download`,
			`"key":"missing"`,
			`"clientIp":"unknown"`,
			`not found`,
			`"result":"0 bytes"`,
			wantCallSite(downloadNewLine),
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
				wantCallSite(echoNewLine),
			)
			assertInternalCaller(t, out)
		}()
		_, _ = loggingMiddleware(panicDemoFactory{}).Demo().Echo(context.Background(), "hi")
		t.Error("expected the panic to propagate out of Do")
	})
}

// panicDemoFactory is a stub factory whose Echo always panics, used to verify
// the panic record's fields and callsite.
type panicDemoFactory struct{}

func (panicDemoFactory) Demo() DemoService { return panicDemoService{} }

type panicDemoService struct{}

func (panicDemoService) Echo(context.Context, string) (string, error)     { panic("boom") }
func (panicDemoService) Download(context.Context, string) ([]byte, error) { return nil, nil }
