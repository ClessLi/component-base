// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package apiserver demonstrates how to build an HTTP API server on top of
// component-base's generic server (pkg/generic_server/server.GenericAPIServer).
//
// It mirrors the structure of the gjallarhorn apiserver, organized as the same
// five layers used across the framework:
//
//	Transport/Controller layer  -> controller.go   (gin handlers)
//	Endpoint layer              -> service methods invoked by controllers
//	Middleware layer            -> middleware.go   (wraps the service factory)
//	Service layer               -> service.go      (business logic)
//	Store layer                 -> store.go        (in-memory persistence)
//
// The generic server drives the lifecycle through server.APIServerBizHooks
// (implemented in hooks.go):
//
//	InitStore -> InitService -> RegisteredMiddlewares -> InitController -> Startup -> Shutdown
//
// Configuration is assembled from options exactly like gjallarhorn's
// config.CreateConfigFromOptions (see server.go):
//
//	Options -> Validate/Complete -> ApplyTo -> APIServerConfig -> Complete -> NewAPIServer
//
// # Running the server
//
// The package exposes Run(opts) as the production entrypoint. A minimal main:
//
//	package main
//
//	import (
//		"log"
//
//		"github.com/ClessLi/component-base/examples/apiserver"
//	)
//
//	func main() {
//		if err := apiserver.Run(apiserver.NewOptions()); err != nil {
//			log.Fatal(err)
//		}
//	}
//
// Run blocks until a shutdown signal (SIGINT/SIGTERM) triggers graceful shutdown.
//
// # Trying the endpoints
//
//	curl http://127.0.0.1:8080/healthz
//	curl -X POST http://127.0.0.1:8080/v1/users -d '{"name":"alice"}'
//	curl http://127.0.0.1:8080/v1/users/1
//	curl http://127.0.0.1:8080/v1/users
//
// # Testing
//
// The accompanying test boots the server via PrepareRun and exercises the gin
// router with httptest (no network port required):
//
//	go test ./examples/apiserver/...
package apiserver // import "github.com/ClessLi/component-base/examples/apiserver"
