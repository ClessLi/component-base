package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/ClessLi/component-base/pkg/utils"
	"github.com/marmotedu/errors"
)

//var EncodeRequest = http_transport.EncodeRequestFunc(func(ctx context.Context, request *http.Request, i interface{}) error {
//
//})

// findMapInterfaceError recursively finds map[interface{}]interface{} types in the given value
func findMapInterfaceError(v interface{}, path string) []string {
	var issues []string

	if v == nil {
		return issues
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return issues
	}

	switch rv.Kind() {
	case reflect.Ptr:
		if rv.IsNil() {
			return issues
		}
		return findMapInterfaceError(rv.Elem().Interface(), path)
	case reflect.Interface:
		return findMapInterfaceError(rv.Elem().Interface(), path)
	case reflect.Struct:
		for i := 0; i < rv.NumField(); i++ {
			field := rv.Type().Field(i)
			fieldValue := rv.Field(i)

			fieldPath := fmt.Sprintf("%s.%s", path, field.Name)
			issues = append(issues, findMapInterfaceError(fieldValue.Interface(), fieldPath)...)
		}
	case reflect.Slice:
		for i := 0; i < rv.Len(); i++ {
			elem := rv.Index(i)
			elemPath := fmt.Sprintf("%s[%d]", path, i)
			issues = append(issues, findMapInterfaceError(elem.Interface(), elemPath)...)
		}
	case reflect.Map:
		// Check the map's key and value types
		keyType := rv.Type().Key()
		elemType := rv.Type().Elem()

		if keyType.Kind() == reflect.Interface && elemType.Kind() == reflect.Interface {
			issues = append(issues, fmt.Sprintf("FOUND map[interface{}]interface{} at path: %s, type: %s", path, rv.Type().String()))
			return issues // Found the problematic type, stop further traversal
		}

		for _, key := range rv.MapKeys() {
			mapValue := rv.MapIndex(key)
			keyStr := fmt.Sprintf("%v", key.Interface()) // Use string representation as path part
			mapPath := fmt.Sprintf("%s[%s]", path, keyStr)
			issues = append(issues, findMapInterfaceError(mapValue.Interface(), mapPath)...)
		}
	}

	return issues
}

func EncodeRequest[REQ any](ctx context.Context, request *http.Request, req HTTPRequest[REQ]) error {
	//// Print request
	//fmt.Printf("DEBUG: Request: %+v\n", req)

	// Parse path parameters in URL path with format :paramName
	// Example: /v1/:appname/:attr-group-name/attr -> /v1/myapp/default-group/attr
	path := request.URL.Path

	// Regular expression to match path parameters in format :paramName, supporting letters, digits, underscores and hyphens
	re := regexp.MustCompile(`:([a-zA-Z0-9_\-]+)`)

	// Find all matching path parameters
	pathMatches := re.FindAllStringSubmatch(path, -1)

	for _, match := range pathMatches {
		if len(match) < 2 {
			continue
		}

		// Get parameter name (without colon)
		paramName := match[1]

		// Get the corresponding value from req.PathVars
		pathParam, exists := req.PathVars[paramName]
		if !exists {
			return errors.Errorf("required path parameter '%s' not found in request map", paramName)
		}

		if pathParam == "" {
			return errors.Errorf("path parameter '%s' has empty value", paramName)
		}

		// Replace parameter in path
		placeholder := ":" + paramName
		path = strings.ReplaceAll(path, placeholder, pathParam)
	}

	// Update request URL path
	request.URL.Path = path

	// Add query parameters to request URL
	query := request.URL.Query()
	for key, values := range req.QueryParams {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	// Update the request URL with the new query string
	request.URL.RawQuery = query.Encode()

	// Add body to request
	// Note: Body can be nil for certain operations (e.g., List with default QueryOptions)
	// Double nil-check mechanism ensures safety:
	// 1. isNilBody[REQ](): Compile-time type check (excludes NilBody special case)
	// 2. utils.IsNil(req.Body): Runtime value check (handles nil pointer values)
	shouldMarshalBody := !isNilBody[REQ]() && !utils.IsNil(req.Body)
	if shouldMarshalBody {
		//// Print request body type and structure for debugging
		//fmt.Printf("DEBUG: Request body type: %T\n", req.Body)
		//fmt.Printf("DEBUG: Request body value: %+v\n", req.Body)
		//
		//// Check for map[interface{}]interface{} types that cause JSON marshaling issues
		//issues := findMapInterfaceError(req.Body, "req.Body")
		//if len(issues) > 0 {
		//	fmt.Printf("DEBUG: Found %d map[interface{}]interface{} issues:\n", len(issues))
		//	for _, issue := range issues {
		//		fmt.Printf("  - %s\n", issue)
		//	}
		//}

		bodydata, err := json.Marshal(req.Body)
		if err != nil {
			//// Print detailed error information
			//fmt.Printf("DEBUG: JSON marshal error: %v\n", err)
			//fmt.Printf("DEBUG: Error type: %T\n", err)
			return errors.Wrapf(err, "failed to marshal request body(%T)", req.Body)
		}
		//// Print body data
		//fmt.Printf("DEBUG: Request body data: %s\n", string(bodydata))
		request.Body = io.NopCloser(strings.NewReader(string(bodydata)))
	}
	// else: skip body for nil QueryOptions or other nil body scenarios

	return nil
}
