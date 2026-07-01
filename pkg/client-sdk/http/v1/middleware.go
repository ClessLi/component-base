package v1

// Middleware wraps an Endpoint and returns a new Endpoint
type Middleware[REQ any, RESP any] func(Endpoint[REQ, RESP]) Endpoint[REQ, RESP]

// Chain applies multiple middlewares in order (first middleware wraps outermost)
func Chain[REQ any, RESP any](middlewares ...Middleware[REQ, RESP]) Middleware[REQ, RESP] {
	return func(next Endpoint[REQ, RESP]) Endpoint[REQ, RESP] {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}
