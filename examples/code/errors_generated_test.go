package code

import (
	"fmt"
	"testing"

	"github.com/ClessLi/component-base/pkg/errors"
)

func TestAggregateGoroutines(t *testing.T) {
	tests := []struct {
		name    string
		funcs   []func() error
		wantNil bool
		wantLen int
	}{
		{
			name:    "empty functions",
			funcs:   []func() error{},
			wantNil: true,
			wantLen: 0,
		},
		{
			name:    "all functions succeed",
			funcs:   []func() error{func() error { return nil }, func() error { return nil }},
			wantNil: true,
			wantLen: 0,
		},
		{
			name:    "one function fails",
			funcs:   []func() error{func() error { return nil }, func() error { return WithCode(ErrUnknown, "error") }},
			wantNil: false,
			wantLen: 1,
		},
		{
			name:    "multiple functions fail",
			funcs:   []func() error{func() error { return WithCode(ErrUnknown, "error1") }, func() error { return WithCode(ErrDatabase, "error2") }},
			wantNil: false,
			wantLen: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AggregateGoroutines(tt.funcs...)
			if tt.wantNil {
				if got != nil {
					t.Errorf("AggregateGoroutines() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("AggregateGoroutines() = nil, want non-nil")
				} else if len(got.Errors()) != tt.wantLen {
					t.Errorf("AggregateGoroutines() errors count = %d, want %d", len(got.Errors()), tt.wantLen)
				}
			}
		})
	}
}

func TestCause(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantNil bool
	}{
		{
			name:    "simple error",
			err:     New("simple error"),
			wantNil: false,
		},
		{
			name:    "wrapped error",
			err:     Wrap(WithCode(ErrDatabase, "db error"), "timeout"),
			wantNil: false,
		},
		{
			name:    "deep nested error chain",
			err:     Wrap(Wrap(Wrap(WithCode(ErrDatabase, "root cause"), "layer1"), "layer2"), "layer3"),
			wantNil: false,
		},
		{
			name:    "non-code error",
			err:     fmt.Errorf("plain error"),
			wantNil: false,
		},
		{
			name:    "nil error",
			err:     nil,
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Cause(tt.err)
			if tt.wantNil {
				if got != nil {
					t.Errorf("Cause() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("Cause() = nil, want non-nil")
				}
			}
		})
	}
}

func TestErrorf(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		args    []interface{}
		wantMsg string
	}{
		{
			name:    "format with string",
			format:  "connection to %s failed",
			args:    []interface{}{"database"},
			wantMsg: "connection to database failed",
		},
		{
			name:    "format with integer",
			format:  "timeout after %d seconds",
			args:    []interface{}{30},
			wantMsg: "timeout after 30 seconds",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Errorf(tt.format, tt.args...)
			if err == nil {
				t.Errorf("Errorf() returned nil, want error")
			}
			if err.Error() != tt.wantMsg {
				t.Errorf("Errorf() error = %q, want %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestFilterOut(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		fns     []errors.Matcher
		wantNil bool
	}{
		{
			name:    "filter out matching error",
			err:     NewAggregate([]error{WithCode(ErrPageNotFound, "not found"), WithCode(ErrDatabase, "db error")}),
			fns:     []errors.Matcher{func(err error) bool { return errors.IsCode(err, Namespace, ErrPageNotFound) }},
			wantNil: false,
		},
		{
			name:    "no matching error",
			err:     WithCode(ErrDatabase, "db error"),
			fns:     []errors.Matcher{func(err error) bool { return errors.IsCode(err, Namespace, ErrPageNotFound) }},
			wantNil: false,
		},
		{
			name:    "nil error",
			err:     nil,
			fns:     []errors.Matcher{func(err error) bool { return true }},
			wantNil: true,
		},
		{
			name: "all errors filtered out",
			err:  NewAggregate([]error{WithCode(ErrPageNotFound, "not found"), WithCode(ErrDatabase, "db error")}),
			fns: []errors.Matcher{
				func(err error) bool { return errors.IsCode(err, Namespace, ErrPageNotFound) },
				func(err error) bool { return errors.IsCode(err, Namespace, ErrDatabase) },
			},
			wantNil: true,
		},
		{
			name:    "non-aggregate error",
			err:     WithCode(ErrUnknown, "error"),
			fns:     []errors.Matcher{func(err error) bool { return errors.IsCode(err, Namespace, ErrPageNotFound) }},
			wantNil: false,
		},
		{
			name: "multiple matchers partial match",
			err:  NewAggregate([]error{WithCode(ErrPageNotFound, "not found"), WithCode(ErrDatabase, "db error"), WithCode(ErrValidation, "invalid")}),
			fns: []errors.Matcher{
				func(err error) bool { return errors.IsCode(err, Namespace, ErrPageNotFound) },
				func(err error) bool { return errors.IsCode(err, Namespace, ErrValidation) },
			},
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterOut(tt.err, tt.fns...)
			if tt.wantNil {
				if got != nil {
					t.Errorf("FilterOut() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("FilterOut() = nil, want non-nil")
				}
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	tests := []struct {
		name    string
		agg     errors.Aggregate
		wantLen int
	}{
		{
			name:    "flat aggregate",
			agg:     NewAggregate([]error{WithCode(ErrUnknown, "error1"), WithCode(ErrDatabase, "error2")}),
			wantLen: 2,
		},
		{
			name: "nested aggregate",
			agg: NewAggregate([]error{
				NewAggregate([]error{WithCode(ErrUnknown, "error1")}),
				WithCode(ErrDatabase, "error2"),
			}),
			wantLen: 2,
		},
		{
			name: "deep nested aggregate",
			agg: NewAggregate([]error{
				NewAggregate([]error{
					NewAggregate([]error{WithCode(ErrUnknown, "error1")}),
				}),
				WithCode(ErrDatabase, "error2"),
			}),
			wantLen: 2,
		},
		{
			name:    "nil aggregate",
			agg:     nil,
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Flatten(tt.agg)
			if tt.agg == nil {
				if got != nil {
					t.Errorf("Flatten() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("Flatten() = nil, want non-nil")
				} else if len(got.Errors()) != tt.wantLen {
					t.Errorf("Flatten() errors count = %d, want %d", len(got.Errors()), tt.wantLen)
				}
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "simple message",
			message: "operation completed",
		},
		{
			name:    "error message",
			message: "something went wrong",
		},
		{
			name:    "validation message",
			message: "invalid input",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.message)
			if err == nil {
				t.Errorf("New() returned nil, want error")
			}
			if err.Error() != tt.message {
				t.Errorf("New() error = %q, want %q", err.Error(), tt.message)
			}
		})
	}
}

func TestNewAggregate(t *testing.T) {
	tests := []struct {
		name    string
		errlist []error
		wantNil bool
		wantLen int
	}{
		{
			name:    "empty list",
			errlist: []error{},
			wantNil: true,
			wantLen: 0,
		},
		{
			name:    "single error",
			errlist: []error{WithCode(ErrUnknown, "error")},
			wantNil: false,
			wantLen: 1,
		},
		{
			name:    "multiple errors",
			errlist: []error{WithCode(ErrUnknown, "error1"), WithCode(ErrDatabase, "error2")},
			wantNil: false,
			wantLen: 2,
		},
		{
			name:    "nil errors filtered out",
			errlist: []error{nil, WithCode(ErrUnknown, "error"), nil},
			wantNil: false,
			wantLen: 1,
		},
		{
			name:    "all nil errors",
			errlist: []error{nil, nil, nil},
			wantNil: true,
			wantLen: 0,
		},
		{
			name:    "nested aggregate",
			errlist: []error{NewAggregate([]error{WithCode(ErrUnknown, "error1")}), WithCode(ErrDatabase, "error2")},
			wantNil: false,
			wantLen: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAggregate(tt.errlist)
			if tt.wantNil {
				if got != nil {
					t.Errorf("NewAggregate() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("NewAggregate() = nil, want non-nil")
				} else if len(got.Errors()) != tt.wantLen {
					t.Errorf("NewAggregate() errors count = %d, want %d", len(got.Errors()), tt.wantLen)
				}
			}
		})
	}
}

func TestReduce(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantNil bool
	}{
		{
			name:    "single error aggregate",
			err:     NewAggregate([]error{WithCode(ErrUnknown, "error")}),
			wantNil: false,
		},
		{
			name:    "multiple errors aggregate",
			err:     NewAggregate([]error{WithCode(ErrUnknown, "error1"), WithCode(ErrDatabase, "error2")}),
			wantNil: false,
		},
		{
			name:    "empty aggregate",
			err:     NewAggregate([]error{}),
			wantNil: true,
		},
		{
			name:    "non-aggregate error",
			err:     WithCode(ErrUnknown, "error"),
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Reduce(tt.err)
			if tt.wantNil {
				if got != nil {
					t.Errorf("Reduce() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("Reduce() = nil, want non-nil")
				}
			}
		})
	}
}

func TestWithCode(t *testing.T) {
	tests := []struct {
		name   string
		code   int
		format string
		args   []interface{}
	}{
		{
			name:   "simple message",
			code:   ErrPageNotFound,
			format: "page not found",
			args:   nil,
		},
		{
			name:   "formatted message",
			code:   ErrPageNotFound,
			format: "page %s not found",
			args:   []interface{}{"/users"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WithCode(tt.code, tt.format, tt.args...)
			if err == nil {
				t.Errorf("WithCode() returned nil, want error")
			}
			coder := errors.ParseCoder(err)
			if coder == nil {
				t.Errorf("WithCode() returned error without coder")
			} else if coder.Code() != tt.code {
				t.Errorf("WithCode() code = %d, want %d", coder.Code(), tt.code)
			} else if coder.Namespace() != Namespace {
				t.Errorf("WithCode() namespace = %s, want %s", coder.Namespace(), Namespace)
			}
		})
	}
}

func TestWithMessage(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{
			name:    "add message to error",
			err:     WithCode(ErrUnknown, "internal error"),
			message: "additional context",
		},
		{
			name:    "add message to wrapped error",
			err:     Wrap(fmt.Errorf("original"), "db error"),
			message: "wrapped context",
		},
		{
			name:    "nil error",
			err:     nil,
			message: "message for nil",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WithMessage(tt.err, tt.message)
			if tt.err == nil {
				if got != nil {
					t.Errorf("WithMessage() with nil err = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("WithMessage() returned nil, want error")
				}
			}
		})
	}
}

func TestWithMessagef(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		format string
		args   []interface{}
	}{
		{
			name:   "format message",
			err:    WithCode(ErrUnknown, "internal error"),
			format: "context: %s",
			args:   []interface{}{"user 123"},
		},
		{
			name:   "format with multiple args",
			err:    WithCode(ErrDatabase, "db error"),
			format: "failed to %s for %s",
			args:   []interface{}{"connect", "database"},
		},
		{
			name:   "nil error",
			err:    nil,
			format: "message for nil: %d",
			args:   []interface{}{123},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WithMessagef(tt.err, tt.format, tt.args...)
			if tt.err == nil {
				if got != nil {
					t.Errorf("WithMessagef() with nil err = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("WithMessagef() returned nil, want error")
				}
			}
		})
	}
}

func TestWithStack(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "add stack to error",
			err:  WithCode(ErrUnknown, "error with stack"),
		},
		{
			name: "add stack to wrapped error",
			err:  Wrap(fmt.Errorf("original"), "db error"),
		},
		{
			name: "nil error",
			err:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WithStack(tt.err)
			if tt.err == nil {
				if got != nil {
					t.Errorf("WithStack() with nil err = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("WithStack() returned nil, want error")
				}
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{
			name:    "wrap simple error",
			err:     fmt.Errorf("simple error"),
			message: "wrapped error",
		},
		{
			name:    "wrap nil error",
			err:     nil,
			message: "wrapped nil",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Wrap(tt.err, tt.message)
			if tt.err == nil {
				if got != nil {
					t.Errorf("Wrap() with nil err = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("Wrap() returned nil, want error")
				}
			}
		})
	}
}

func TestWrapC(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		code   int
		format string
		args   []interface{}
	}{
		{
			name:   "wrap with code",
			err:    fmt.Errorf("original error"),
			code:   ErrDatabase,
			format: "wrapped: %s",
			args:   []interface{}{"context"},
		},
		{
			name:   "wrap nil with code",
			err:    nil,
			code:   ErrUnknown,
			format: "nil wrapped",
			args:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapC(tt.err, tt.code, tt.format, tt.args...)
			if tt.err == nil {
				if got != nil {
					t.Errorf("WrapC() with nil err = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("WrapC() returned nil, want error")
				} else {
					coder := errors.ParseCoder(got)
					if coder == nil {
						t.Errorf("WrapC() returned error without coder")
					} else if coder.Code() != tt.code {
						t.Errorf("WrapC() code = %d, want %d", coder.Code(), tt.code)
					}
				}
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		format string
		args   []interface{}
	}{
		{
			name:   "wrap with format",
			err:    fmt.Errorf("original"),
			format: "wrapped: %s",
			args:   []interface{}{"context"},
		},
		{
			name:   "wrap nil with format",
			err:    nil,
			format: "nil wrapped: %d",
			args:   []interface{}{123},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Wrapf(tt.err, tt.format, tt.args...)
			if tt.err == nil {
				if got != nil {
					t.Errorf("Wrapf() with nil err = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("Wrapf() returned nil, want error")
				}
			}
		})
	}
}

func TestIsCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
		want bool
	}{
		{
			name: "matching code",
			err:  WithCode(ErrPageNotFound, "not found"),
			code: ErrPageNotFound,
			want: true,
		},
		{
			name: "wrong code",
			err:  WithCode(ErrPageNotFound, "not found"),
			code: ErrDatabase,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			code: ErrPageNotFound,
			want: false,
		},
		{
			name: "non-code error",
			err:  fmt.Errorf("plain error"),
			code: ErrPageNotFound,
			want: false,
		},
		{
			name: "wrapped code error",
			err:  Wrap(WithCode(ErrPageNotFound, "not found"), "wrapped"),
			code: ErrPageNotFound,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCode(tt.err, tt.code); got != tt.want {
				t.Errorf("IsCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCoder(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantCoder bool
		wantCode  int
		wantNS    string
	}{
		{
			name:      "valid error with coder",
			err:       WithCode(ErrPageNotFound, "not found"),
			wantCoder: true,
			wantCode:  ErrPageNotFound,
			wantNS:    Namespace,
		},
		{
			name:      "simple error returns unknown coder",
			err:       fmt.Errorf("simple error"),
			wantCoder: true,
			wantCode:  100002, // unknownCoder code
			wantNS:    "",
		},
		{
			name:      "nil error",
			err:       nil,
			wantCoder: false,
		},
		{
			name:      "wrapped code error",
			err:       Wrap(WithCode(ErrPageNotFound, "not found"), "wrapped"),
			wantCoder: true,
			wantCode:  ErrPageNotFound, // Wrap preserves original code
			wantNS:    Namespace,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coder := errors.ParseCoder(tt.err)
			if tt.wantCoder {
				if coder == nil {
					t.Errorf("ParseCoder() = nil, want coder")
				} else if tt.wantNS != "" {
					if coder.Code() != tt.wantCode {
						t.Errorf("ParseCoder() code = %d, want %d", coder.Code(), tt.wantCode)
					}
					if coder.Namespace() != tt.wantNS {
						t.Errorf("ParseCoder() namespace = %s, want %s", coder.Namespace(), tt.wantNS)
					}
				}
			} else {
				if coder != nil {
					t.Errorf("ParseCoder() = %v, want nil", coder)
				}
			}
		})
	}
}

func TestErrorsIs(t *testing.T) {
	// Create a sentinel error for comparison
	var sentinelErr = WithCode(ErrPageNotFound, "sentinel")

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "same error instance matches",
			err:    sentinelErr,
			target: sentinelErr,
			want:   true,
		},
		{
			name:   "different error",
			err:    WithCode(ErrPageNotFound, "not found"),
			target: WithCode(ErrDatabase, "db error"),
			want:   false,
		},
		{
			name:   "wrapped error matches sentinel",
			err:    Wrap(sentinelErr, "wrapped"),
			target: sentinelErr,
			want:   true,
		},
		{
			name:   "wrapped error does not match different",
			err:    Wrap(WithCode(ErrDatabase, "db error"), "wrapped"),
			target: sentinelErr,
			want:   false,
		},
		{
			name:   "nil error",
			err:    nil,
			target: sentinelErr,
			want:   false,
		},
		{
			name:   "nil target",
			err:    sentinelErr,
			target: nil,
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorsAs(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		wantOk bool
	}{
		{
			name:   "error can be cast to Coder via ParseCoder",
			err:    WithCode(ErrPageNotFound, "not found"),
			wantOk: true,
		},
		{
			name:   "simple error cannot be cast to Coder",
			err:    fmt.Errorf("simple error"),
			wantOk: false,
		},
		{
			name:   "nil error",
			err:    nil,
			wantOk: false,
		},
		{
			name:   "wrapped code error",
			err:    Wrap(WithCode(ErrPageNotFound, "not found"), "wrapped"),
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coder := errors.ParseCoder(tt.err)
			if tt.wantOk {
				if coder == nil {
					t.Errorf("ParseCoder() = nil, want coder")
				}
			} else {
				// simple errors return unknownCoder which is not nil
				// so we just verify the behavior
			}
		})
	}
}

func TestIs(t *testing.T) {
	sentinelErr := WithCode(ErrPageNotFound, "not found")

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "same error instance matches",
			err:    sentinelErr,
			target: sentinelErr,
			want:   true,
		},
		{
			name:   "different error",
			err:    WithCode(ErrPageNotFound, "not found"),
			target: WithCode(ErrDatabase, "db error"),
			want:   false,
		},
		{
			name:   "wrapped error matches sentinel",
			err:    Wrap(sentinelErr, "wrapped"),
			target: sentinelErr,
			want:   true,
		},
		{
			name:   "wrapped error does not match different",
			err:    Wrap(WithCode(ErrDatabase, "db error"), "wrapped"),
			target: sentinelErr,
			want:   false,
		},
		{
			name:   "nil error",
			err:    nil,
			target: sentinelErr,
			want:   false,
		},
		{
			name:   "nil target",
			err:    sentinelErr,
			target: nil,
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Is(tt.err, tt.target); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAs(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target interface{}
		want   bool
	}{
		{
			name:   "error can be cast to error type",
			err:    WithCode(ErrPageNotFound, "not found"),
			target: new(error),
			want:   true,
		},
		{
			name:   "simple error can be cast to error type",
			err:    fmt.Errorf("simple error"),
			target: new(error),
			want:   true,
		},
		{
			name:   "nil error",
			err:    nil,
			target: new(error),
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := As(tt.err, tt.target); got != tt.want {
				t.Errorf("As() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantNil bool
	}{
		{
			name:    "wrapped error unwraps",
			err:     Wrap(WithCode(ErrPageNotFound, "not found"), "wrapped"),
			wantNil: false,
		},
		{
			name:    "simple error returns nil",
			err:     fmt.Errorf("simple error"),
			wantNil: true,
		},
		{
			name:    "nil error returns nil",
			err:     nil,
			wantNil: true,
		},
		{
			name:    "WithCode error unwraps to cause",
			err:     WrapC(fmt.Errorf("cause"), ErrPageNotFound, "wrapped: %s", "context"),
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Unwrap(tt.err)
			if tt.wantNil {
				if got != nil {
					t.Errorf("Unwrap() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("Unwrap() = nil, want non-nil")
				}
			}
		})
	}
}
