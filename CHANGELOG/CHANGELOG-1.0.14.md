<a name="v1.0.14"></a>

## [v1.0.14] - 2026-11-18

### Features

- **generic_server:** add a configurable generic HTTP server framework with out-of-the-box service assembly and
  lifecycle management (c83eacb)
    - Lifecycle management: full `NewAPIServer` → `PrepareRun` → `Run` → `Close` chain with POSIX signal driven graceful
      shutdown
    - Store/Service/Controller install hooks for business dependency injection, and a builder-pattern middleware
      registry for extensible middleware chains
    - Built-in healthz, version and metrics handlers, with structured logging and error wrapping throughout the flow
    - Ship the `grpc_health/v1` health-check client SDK alongside for reusable health probing (c83eacb, 6e23b57)
- **generic_server:** add project-aware logger options with dependency injection support so logging configuration is
  assembled uniformly with the project context (578960a)
- **generic_server:** complete the generic gRPC server framework and add apiserver (HTTP) and grpcserver (gRPC)
  five-layer reference examples (6e23b57)
    - transport/v1: add `StreamHandler` / `ServeStream` / `StreamSendMsg` streaming utilities for server-side chunked
      streaming
    - Add gRPC serving options and wire chunk size configuration into service hooks
    - Examples integrate methodlog structured method logging and serve as assembly templates for business services
- **client-sdk:** add the generic gRPC client SDK `grpc/v1` with transparent aggregation of streaming responses (
  342d435)
    - Domain-typed `Client` / `ClientBuilder`, injected `Encoder` / `Decoder`, `Endpoint`, `Middleware` / `Chain` and
      metadata helpers
    - `StreamRecvMsg` transparently aggregates server-streaming chunks into a unary response, so callers need not handle
      streaming details
    - Round-trip tests against examples/grpcserver (unary Echo + streaming Download)
- **client-sdk:** add layered httpclient/grpcclient examples and migrate `User` / `CreateUserRequest` into
  examples/apiserver/api/v1 as the shared API contract, eliminating duplicated domain types between client and server (
  b0024ea)
- **methodlog:** lazy logger resolution, new `ResultWithError` / `WithLogger` APIs and a `callsite` log field, fully
  backward compatible with the public API (16f84d0)
    - Remove the one-time binding of level loggers at init and resolve them lazily per `Do`, so the log output
      destination follows host `log.Init` re-initialization (behavior fix)
    - Add `ResultWithError(builder func(err error) string)` so the result-message closure reads the final error of `Do`
      directly, without assigning to an outer captured variable inside `Action`
    - Add the `WithLogger(logger kitlog.Logger)` Option to inject a fixed logger for a single `Do`, enabling output
      capture in tests
    - Add a `callsite` field to every log record pinpointing the business call site (short form
      `package-dir/file.go:line`), and document that the backend `caller` field refers to the component-internal
      location
    - Drop dead code `getLimitResult` / `limit` and the file methodlog.go; add middleware log and callsite contract
      tests for apiserver/grpcserver

### Bug Fixes

- **pkg:** populate the missing Namespace from `errors.Coder` when writing error responses, so clients can identify
  which service namespace produced the error (3f82c55)

### Code Refactoring

- **generic_server:** optimize the HTTP server lifecycle and health-check logic, and replace `gin.New` with
  `gin.Default` to include the built-in Logger/Recovery middleware (a5ed560, 53ebb61)
    - Improve the shutdown flow with error aggregation and context cancellation, add exponential backoff to health-check
      retries, and fix health-check address parsing for 0.0.0.0 binding
    - Add server state constants for lifecycle tracking, prevent startup after a shutdown signal is received, and
      enhance business resource cleanup during graceful shutdown and run-options validation error messages
    - 53ebb61 also generates the commit-message scope convention document docs/devel/zh-CN/scope.md
- **generic_server:** decouple the registration control flow and streamline the gRPC server SDK; business hooks only
  declare registrations while the framework owns the control flow (cb06610)
    - Drive all five layers in `initBiz` via `InitEndpoints` / `InitTransport` hooks, with chunk size injected from
      configuration
    - Collapse the service_register generator chain into `GenerateServiceRegisters` and drop the unused scheme type
    - Health status transitions (NOT_SERVING → mount → SERVING) are managed by the framework; polish the grpcserver
      example as a reference template with unified English comments
- **generic_server:** build the gRPC server via the fluent completed-config style (
  `cfg.Complete().NewGRPCServer(ctx, bizHooks)`), aligning with the HTTP `NewAPIServer` assembly chain; contains
  breaking API adjustments, see the BREAKING CHANGE section below (5a66b36)

### BREAKING CHANGE

gRPC server construction API adjustments (5a66b36):

1. `server.NewGRPCServer` is no longer a package-level function; it becomes a value-receiver method on
   `CompletedGRPCServerConfig`. Build the server via `cfg.Complete().NewGRPCServer(ctx, hooks)`.
2. `GRPCServerConfig`, `CompletedGRPCServerConfig` and the gRPC-only options (`GRPCServerRunOptions`,
   `GRPCServingOptions`, `RAOptions`) widen their type parameters from `[Svc, Store]` to `[Txp, Ep, Svc, Store]`; the
   shared Secure/InsecureServingOptions stay `[Svc, Store]`, so the HTTP path is untouched.
3. The `ApplyToGRPCConfig` methods on `SecureServingOptions` / `InsecureServingOptions` are replaced by the
   package-level generic functions `ApplySecureServingToGRPCConfig` / `ApplyInsecureServingToGRPCConfig` (Txp/Ep
   inferred from the config argument).

The code migration example is as follows:

Before:

```go
import (
genericoptions "github.com/ClessLi/component-base/pkg/generic_server/options"
genericserver "github.com/ClessLi/component-base/pkg/generic_server/server"
)

// The config carries only the Svc/Store type parameters
cfg := genericserver.NewGRPCServerConfig[ServiceFactory, StoreFactory]()
// Shared serving options are applied to the gRPC config via a method
if err := opts.InsecureServingOptions.ApplyToGRPCConfig(cfg); err != nil {
return err
}
// Package-level construction; Txp/Ep must be specified explicitly
srv := genericserver.NewGRPCServer[Factory, DemoEndpoints, ServiceFactory, StoreFactory](ctx, cfg.Complete(), bizHooks)
```

After:

```go
import (
genericoptions "github.com/ClessLi/component-base/pkg/generic_server/options"
genericserver "github.com/ClessLi/component-base/pkg/generic_server/server"
)

// The config carries the four Txp/Ep/Svc/Store type parameters
cfg := genericserver.NewGRPCServerConfig[Factory, DemoEndpoints, ServiceFactory, StoreFactory]()
// Shared serving options are applied to the gRPC config via a package-level generic function
if err := genericoptions.ApplyInsecureServingToGRPCConfig(opts.InsecureServingOptions, cfg); err != nil {
return err
}
// Fluent construction; type parameters are inferred from the config
srv := cfg.Complete().NewGRPCServer(ctx, bizHooks)
```

Impact boundary: all the gRPC APIs above were introduced within this release cycle (v1.0.13 does not contain the gRPC
server framework), so users upgrading from v1.0.13 are not affected by this breaking adjustment; the note only concerns
consumers who tracked intermediate states of the develop branch during this cycle.

[v1.0.14]: https://github.com/ClessLi/component-base/compare/v1.0.13...v1.0.14