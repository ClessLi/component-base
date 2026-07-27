package errors

import (
	"fmt"
	"net/http"
	"sync"
)

var (
	unknownCoder defaultCoder = defaultCoder{1, http.StatusInternalServerError, "An internal server error occurred", "http://github.com/ClessLi/component-base/pkg/errors/README.md", "base"}
)

// Coder defines an interface for an error code detail information.
//
// Each project should implement this interface for their error codes,
// specifying a unique namespace to isolate error codes across projects.
type Coder interface {
	// HTTPStatus returns the HTTP status that should be used for the associated error code.
	HTTPStatus() int

	// String returns the external (user) facing error text.
	String() string

	// Reference returns the detail documents for user.
	Reference() string

	// Code returns the integer code of the coder.
	Code() int

	// Namespace returns the namespace that this error code belongs to.
	Namespace() string
}

// defaultCoder is the default implementation of the Coder interface.
//
// Projects can use this struct directly to define their error codes:
//
//	var ErrNotFound = defaultCoder{
//	    C:    1001,
//	    HTTP: 404,
//	    Ext:  "Resource not found",
//	    Ref:  "http://docs.example.com/errors/1001",
//	    NS:   "myproject",
//	}
type defaultCoder struct {
	// C refers to the integer code of the ErrCode.
	C int

	// HTTP status that should be used for the associated error code.
	HTTP int

	// External (user) facing error text.
	Ext string

	// Ref specify the reference document.
	Ref string

	// NS specifies the namespace that this error code belongs to.
	NS string
}

// Code returns the integer code of the coder.
func (coder defaultCoder) Code() int {
	return coder.C
}

// String implements stringer. String returns the external error message,
// if any.
func (coder defaultCoder) String() string {
	return coder.Ext
}

// HTTPStatus returns the associated HTTP status code, if any. Otherwise,
// returns 500.
func (coder defaultCoder) HTTPStatus() int {
	if coder.HTTP == 0 {
		return 500
	}

	return coder.HTTP
}

// Reference returns the reference document.
func (coder defaultCoder) Reference() string {
	return coder.Ref
}

// Namespace returns the namespace that this error code belongs to.
func (coder defaultCoder) Namespace() string {
	return coder.NS
}

// DefaultCoder creates a new Coder with the specified parameters.
// This is a convenience constructor for defining error codes that implement the Coder interface.
//
// Parameters:
//   - code: the integer error code (e.g., 1001, 1002)
//   - httpStatus: the HTTP status code (e.g., 400, 404, 500)
//   - message: the user-facing error message describing how to resolve the issue
//   - ref: optional reference document URL for detailed error information
//   - namespace: the namespace that identifies which project this error code belongs to
//
// Example:
//
//	var ErrNotFound = errors.DefaultCoder(
//	    1001,
//	    404,
//	    "Resource not found. Check the identifier and try again",
//	    "http://docs.example.com/errors/1001",
//	    "myproject",
//	)
//
//	func init() {
//	    errors.Register(ErrNotFound)
//	}
func DefaultCoder(code int, httpStatus int, message string, ref string, namespace string) Coder {
	return defaultCoder{
		C:    code,
		HTTP: httpStatus,
		Ext:  message,
		Ref:  ref,
		NS:   namespace,
	}
}

// Global registries
var (
	nsCodes     = map[string]map[int]Coder{} // namespace → (code → Coder)
	registryMux = &sync.RWMutex{}
)

// Register registers a user defined error code.
// The namespace is extracted from coder.Namespace().
// It will panic if the code already exists in the same namespace or namespace is empty.
//
// Example:
//
//	func init() {
//	    errors.Register(myprojectNotFoundCoder)
//	}
func Register(coder Coder) {
	code := coder.Code()
	if code == 0 {
		panic("code `0` is reserved by `github.com/ClessLi/component-base/pkg/errors` as unknownCode error code")
	}

	namespace := coder.Namespace()
	if namespace == "" {
		panic("coder must have a non-empty namespace")
	}

	registryMux.Lock()
	defer registryMux.Unlock()

	// Ensure namespace exists
	if _, ok := nsCodes[namespace]; !ok {
		nsCodes[namespace] = make(map[int]Coder)
	}

	// Enforce namespace uniqueness
	if existing, ok := nsCodes[namespace][code]; ok {
		panic(fmt.Sprintf(
			"code %d already registered in namespace '%s': '%s'",
			code, namespace, existing.String(),
		))
	}

	// Register
	nsCodes[namespace][code] = coder
}

// ParseCoder parses any error into Coder.
// nil error will return nil directly.
// Non-withCode error will be parsed as unknownCoder.
//
// The Coder is looked up by the error's namespace and code, allowing
// cross-project error code resolution.
func ParseCoder(err error) Coder {
	if err == nil {
		return nil
	}

	if v, ok := err.(*withCode); ok {
		if coder := findCoderByCodeInNamespace(v.code, v.namespace); coder != nil {
			return coder
		}
	}

	return unknownCoder
}

// findCoderByCodeInNamespace searches for a coder by code in a specific namespace.
// Returns nil if not found.
func findCoderByCodeInNamespace(code int, namespace string) Coder {
	registryMux.RLock()
	defer registryMux.RUnlock()

	if nsMap, ok := nsCodes[namespace]; ok {
		return nsMap[code]
	}
	return nil
}

// IsCode reports whether any error in err's chain contains the given error code
// within the specified namespace.
//
// This allows cross-project error checking by verifying both namespace and code:
//
//	if errors.IsCode(err, "myproject", ErrNotFound) {
//	    // handle myproject not found error
//	}
func IsCode(err error, namespace string, code int) bool {
	if v, ok := err.(*withCode); ok {
		if v.namespace == namespace && v.code == code {
			return true
		}

		if v.cause != nil {
			return IsCode(v.cause, namespace, code)
		}

		return false
	}

	return false
}

// GetErrorNamespace returns the namespace of an error.
// Returns "unknown" if the error is not a withCode error.
//
// This is useful for identifying which project an error originated from:
//
//	ns := errors.GetErrorNamespace(err)
//	switch ns {
//	case "myproject":
//	    // handle myproject error
//	case "otherproject":
//	    // handle otherproject error
//	}
func GetErrorNamespace(err error) string {
	if v, ok := err.(*withCode); ok {
		return v.namespace
	}
	return "unknown"
}

func init() {
	// Register unknownCoder
	Register(unknownCoder)
}
