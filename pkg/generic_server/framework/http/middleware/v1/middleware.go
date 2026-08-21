// Package v1 provides generic HTTP middleware utilities.
package v1

// Middleware defines a function that wraps a service factory to add middleware behavior.
// Svc represents the service factory type being wrapped.
type Middleware[Svc any] func(factory Svc) Svc

// Middlewares is a registry of named middleware functions.
type Middlewares[Svc any] map[string]Middleware[Svc]
