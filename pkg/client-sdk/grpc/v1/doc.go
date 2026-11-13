// Package v1 provides a generic gRPC client SDK mirroring the HTTP client SDK
// (pkg/client-sdk/http/v1). It exposes a domain-typed, unary-only Client[REQ, RESP]
// built through a fluent ClientBuilder with composable middlewares.
//
// Protocol encoding is injected (Encoder/Decoder) rather than automatic, matching
// the gRPC server framework's generic contracts. Server-streaming RPCs are handled
// transparently by the transport layer: StreamRecvMsg aggregates all chunks into a
// complete payload, which the endpoint forwards to the caller as a single unary
// response, so client users consume the service layer without stream awareness.
//
// Channel-based streaming (a channel-driven receive primitive and a service-layer
// streaming client) is planned; see the TODO(channel-transport) markers.
package v1
