package v1

import "reflect"

const (
	// HTTPMethodPost is the HTTP POST method.
	HTTPMethodPost = "POST"
	// HTTPMethodGet is the HTTP GET method.
	HTTPMethodGet = "GET"
	// HTTPMethodPut is the HTTP PUT method.
	HTTPMethodPut = "PUT"
	// HTTPMethodDelete is the HTTP DELETE method.
	HTTPMethodDelete = "DELETE"
)

// NilBody is a type used as a placeholder when no request or response body is needed
type NilBody struct{}

// isNilBody checks if the type parameter RESP is NilBody
func isNilBody[R any]() bool {
	respType := reflect.TypeOf((*R)(nil)).Elem()
	nilType := reflect.TypeOf(NilBody{})
	return respType.String() == nilType.String()
}

type HTTPRequest[REQ any] struct {
	PathVars    map[string]string
	QueryParams map[string][]string
	Body        REQ
}
