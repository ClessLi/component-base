package v1

import (
	"context"
	"net"

	"google.golang.org/grpc/peer"
)

// FieldExtractor defines a function that extracts log fields from context.
// It enables protocol-specific field extraction (HTTP, gRPC, etc.).
type FieldExtractor func(ctx context.Context) []interface{}

// HTTPFieldExtractor extracts HTTP-specific fields from context.
// It retrieves client IP from context (set by middleware using WithClientIP).
//
// Usage:
//
//	methodlog.New(ctx, handler, methodlog.WithFieldExtractor(methodlog.HTTPFieldExtractor)).Do()
var HTTPFieldExtractor = FieldExtractor(func(ctx context.Context) []interface{} {
	return []interface{}{
		"clientIp", getHTTPClientIP(ctx),
		// Add more HTTP-specific fields as needed:
		// "userAgent", getUserAgent(ctx),
		// "requestId", getRequestId(ctx),
	}
})

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// Context keys for HTTP-specific values.
const (
	clientIPKey contextKey = "clientIP"
)

// getHTTPClientIP retrieves the client IP from context.
// Returns "unknown" if no IP is set.
func getHTTPClientIP(ctx context.Context) string {
	// Try to get client IP from context (set by middleware)
	if ip, ok := ctx.Value(clientIPKey).(string); ok && ip != "" {
		return ip
	}
	return "unknown"
}

// WithClientIP returns a context with the client IP stored.
// Use this in HTTP middleware to propagate client IP to the logger.
//
// Example:
//
//	ctx = methodlog.WithClientIP(r.Context(), "192.168.1.1")
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey, ip)
}

// GRPCFieldExtractor extracts gRPC-specific fields from context.
// It retrieves client IP from gRPC peer information.
//
// Usage:
//
//	methodlog.New(ctx, handler, methodlog.WithFieldExtractor(methodlog.GRPCFieldExtractor)).Do()
var GRPCFieldExtractor = FieldExtractor(func(ctx context.Context) []interface{} {
	return []interface{}{
		"clientIp", getGRPCClientIP(ctx),
		// Add more gRPC-specific fields as needed:
		// "peer", getPeerInfo(ctx),
		// "method", getGRPCMethod(ctx),
	}
})

// getGRPCClientIP retrieves the client IP from gRPC peer context.
// Returns "unknown" if peer information is not available.
func getGRPCClientIP(ctx context.Context) string {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return "unknown"
	}
	if pr.Addr == net.Addr(nil) {
		return "unknown"
	}

	return pr.Addr.String()
}
