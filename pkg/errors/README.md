# Errors Package

## Overview

This package is cloned from [`github.com/marmotedu/errors`](https://github.com/marmotedu/errors) and optimized for local
use within the `component-base` project.

## Origin

- **Source Repository**: github.com/marmotedu/errors
- **License**: See [LICENSE](LICENSE) file for original license terms
- **Purpose**: Provides simple error handling primitives with context, stack traces, and error code support

## Why Cloned

This package was cloned from the original `github.com/marmotedu/errors` repository to:

1. **Reduce External Dependencies**: Minimize reliance on external packages for core error handling functionality
2. **Enable Local Optimizations**: Allow project-specific modifications without affecting upstream
3. **Improve Build Performance**: Avoid network fetches for this critical dependency
4. **Maintain Control**: Full control over error handling behavior and evolution

## Features

- **Error Wrapping**: Add context to errors while preserving the original error
- **Stack Traces**: Automatic stack trace capture at error creation points
- **Error Codes**: Register and use custom error codes with HTTP status mappings
- **Error Causes**: Retrieve the root cause from wrapped error chains
- **Formatted Output**: Support for detailed error formatting with stack traces

## Usage

### Basic Error Creation

```go
import "github.com/ClessLi/component-base/pkg/errors"

// Create a new error
err := errors.New("something went wrong")

// Create error with formatting
err := errors.Errorf("failed to process %s: %d", name, count)
```

### Error Wrapping

```go
// Wrap an error with context
err := errors.Wrap(originalErr, "read failed")

// Wrap with formatting
err := errors.Wrapf(originalErr, "failed to read %s", filename)
```

### Error Code Registration

```go
// Define a custom error code
type MyErrorCode struct {
    code int
    http int
    msg  string
}

func (e MyErrorCode) Code() int         { return e.code }
func (e MyErrorCode) HTTPStatus() int   { return e.http }
func (e MyErrorCode) String() string    { return e.msg }
func (e MyErrorCode) Reference() string { return "http://docs.example.com/errors" }

// Register the error code
errors.Register(MyErrorCode{
    code: 10001,
    http: http.StatusBadRequest,
    msg:  "Invalid request parameter",
})
```

### Retrieving Error Cause

```go
switch err := errors.Cause(err).(type) {
case *MyError:
    // handle specific error type
default:
    // unknown error
}
```

### Formatted Error Output

```go
// Standard output
fmt.Printf("%v", err)

// Detailed output with stack trace
fmt.Printf("%+v", err)
```

## API Reference

### Core Functions

- `New(message string) error` - Create a new error
- `Errorf(format string, args ...interface{}) error` - Create formatted error
- `Wrap(err error, message string) error` - Wrap error with context
- `Wrapf(err error, format string, args ...interface{}) error` - Wrap with formatted context
- `WithMessage(err error, message string) error` - Add message without stack trace
- `WithStack(err error) error` - Add stack trace to error
- `Cause(err error) error` - Retrieve root cause
- `Is(err, target error) bool` - Check error equality (Go 1.13+)
- `As(err error, target interface{}) bool` - Type assertion (Go 1.13+)

### Error Code Functions

- `Register(coder Coder)` - Register custom error code
- `MustRegister(coder Coder)` - Register with duplicate check
- `GetCode(code int) Coder` - Retrieve registered code
- `ParseCoder(err error) Coder` - Extract code from error

## Differences from Original

This cloned version may include the following optimizations:

- Project-specific error code conventions
- Enhanced error message formatting
- Performance improvements
- Bug fixes not yet merged upstream

## License

This package retains the license from the original `github.com/marmotedu/errors` repository. See the [LICENSE](LICENSE)
file for details.

## Migration

If you were previously using `github.com/marmotedu/errors`, simply update your imports:

```go
// Old import
import "github.com/marmotedu/errors"

// New import
import "github.com/ClessLi/component-base/pkg/errors"
```

The API remains compatible with the original package.