package errors

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		err  string
		want error
	}{
		{"", fmt.Errorf("")},
		{"foo", fmt.Errorf("foo")},
		{"foo", New("foo")},
		{"string with format specifiers: %v", errors.New("string with format specifiers: %v")},
	}

	for _, tt := range tests {
		got := New(tt.err)
		if got.Error() != tt.want.Error() {
			t.Errorf("New.Error(): got: %q, want %q", got, tt.want)
		}
	}
}

func TestWrapNil(t *testing.T) {
	got := Wrap(nil, "no error")
	if got != nil {
		t.Errorf("Wrap(nil, \"no error\"): got %#v, expected nil", got)
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		err     error
		message string
		want    string
	}{
		{io.EOF, "read error", "read error"},
		{Wrap(io.EOF, "read error"), "client error", "client error"},
	}

	for _, tt := range tests {
		got := Wrap(tt.err, tt.message).Error()
		if got != tt.want {
			t.Errorf("Wrap(%v, %q): got: %v, want %v", tt.err, tt.message, got, tt.want)
		}
	}
}

type nilError struct{}

func (nilError) Error() string { return "nil error" }

func TestCause(t *testing.T) {
	x := New("error")
	tests := []struct {
		err  error
		want error
	}{{
		// nil error is nil
		err:  nil,
		want: nil,
	}, {
		// explicit nil error is nil
		err:  (error)(nil),
		want: nil,
	}, {
		// typed nil is nil
		err:  (*nilError)(nil),
		want: (*nilError)(nil),
	}, {
		// uncaused error is unaffected
		err:  io.EOF,
		want: io.EOF,
	}, {
		// caused error returns cause
		err:  Wrap(io.EOF, "ignored"),
		want: io.EOF,
	}, {
		err:  x, // return from errors.New
		want: x,
	}, {
		WithMessage(nil, "whoops"),
		nil,
	}, {
		WithMessage(io.EOF, "whoops"),
		io.EOF,
	}, {
		WithStack(nil),
		nil,
	}, {
		WithStack(io.EOF),
		io.EOF,
	}}

	for i, tt := range tests {
		got := Cause(tt.err)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("test %d: got %#v, want %#v", i+1, got, tt.want)
		}
	}
}

func TestWrapfNil(t *testing.T) {
	got := Wrapf(nil, "no error")
	if got != nil {
		t.Errorf("Wrapf(nil, \"no error\"): got %#v, expected nil", got)
	}
}

func TestWrapf(t *testing.T) {
	tests := []struct {
		err     error
		message string
		want    string
	}{
		{io.EOF, "read error", "read error"},
		{Wrapf(io.EOF, "read error without format specifiers"), "client error", "client error"},
		{Wrapf(io.EOF, "read error with %d format specifier", 1), "client error", "client error"},
	}

	for _, tt := range tests {
		got := Wrapf(tt.err, "%s", tt.message).Error()
		if got != tt.want {
			t.Errorf("Wrapf(%v, %q): got: %v, want %v", tt.err, tt.message, got, tt.want)
		}
	}
}

func TestErrorf(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{Errorf("read error without format specifiers"), "read error without format specifiers"},
		{Errorf("read error with %d format specifier", 1), "read error with 1 format specifier"},
	}

	for _, tt := range tests {
		got := tt.err.Error()
		if got != tt.want {
			t.Errorf("Errorf(%v): got: %q, want %q", tt.err, got, tt.want)
		}
	}
}

func TestWithStackNil(t *testing.T) {
	got := WithStack(nil)
	if got != nil {
		t.Errorf("WithStack(nil): got %#v, expected nil", got)
	}
}

func TestWithStack(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{io.EOF, "EOF"},
		{WithStack(io.EOF), "EOF"},
	}

	for _, tt := range tests {
		got := WithStack(tt.err).Error()
		if got != tt.want {
			t.Errorf("WithStack(%v): got: %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestWithMessageNil(t *testing.T) {
	got := WithMessage(nil, "no error")
	if got != nil {
		t.Errorf("WithMessage(nil, \"no error\"): got %#v, expected nil", got)
	}
}

func TestWithMessage(t *testing.T) {
	tests := []struct {
		err     error
		message string
		want    string
	}{
		{io.EOF, "read error", "read error"},
		{WithMessage(io.EOF, "read error"), "client error", "client error"},
	}

	for _, tt := range tests {
		got := WithMessage(tt.err, tt.message).Error()
		if got != tt.want {
			t.Errorf("WithMessage(%v, %q): got: %q, want %q", tt.err, tt.message, got, tt.want)
		}
	}
}

func TestWithMessagefNil(t *testing.T) {
	got := WithMessagef(nil, "no error")
	if got != nil {
		t.Errorf("WithMessage(nil, \"no error\"): got %#v, expected nil", got)
	}
}

func TestWithMessagef(t *testing.T) {
	tests := []struct {
		err     error
		message string
		want    string
	}{
		{io.EOF, "read error", "read error"},
		{WithMessagef(io.EOF, "read error without format specifier"), "client error", "client error"},
		{WithMessagef(io.EOF, "read error with %d format specifier", 1), "client error", "client error"},
	}

	for _, tt := range tests {
		got := WithMessagef(tt.err, "%s", tt.message).Error()
		if got != tt.want {
			t.Errorf("WithMessage(%v, %q): got: %q, want %q", tt.err, tt.message, got, tt.want)
		}
	}
}

func TestWithCode(t *testing.T) {
	tests := []struct {
		code     int
		message  string
		wantType string
		wantCode int
	}{
		{ConfigurationNotValid, "ConfigurationNotValid error", "*withCode", ConfigurationNotValid},
	}

	for _, tt := range tests {
		got := WithCode("test", tt.code, "%s", tt.message)
		err, ok := got.(*withCode)
		if !ok {
			t.Errorf("WithCode(%v, %q): error type got: %T, want %s", tt.code, tt.message, got, tt.wantType)
		}

		if err.code != tt.wantCode {
			t.Errorf("WithCode(%v, %q): got: %v, want %v", tt.code, tt.message, err.code, tt.wantCode)
		}
	}
}

func TestWithCodef(t *testing.T) {
	tests := []struct {
		code       int
		format     string
		args       string
		wantType   string
		wantCode   int
		wangString string
	}{
		{ConfigurationNotValid, "Configuration %s", "failed", "*withCode", ConfigurationNotValid, `ConfigurationNotValid error`},
	}

	for _, tt := range tests {
		got := WithCode("test", tt.code, tt.format, tt.args)
		err, ok := got.(*withCode)
		if !ok {
			t.Errorf("WithCode(%v, %q %q): error type got: %T, want %s", tt.code, tt.format, tt.args, got, tt.wantType)
		}

		if err.code != tt.wantCode {
			t.Errorf("WithCode(%v, %q %q): got: %v, want %v", tt.code, tt.format, tt.args, err.code, tt.wantCode)
		}

		if got.Error() != tt.wangString {
			t.Errorf("WithCode(%v, %q %q): got: %v, want %v", tt.code, tt.format, tt.args, got.Error(), tt.wangString)
		}
	}
}

// errors.New, etc values are not expected to be compared by value
// but the change in errors#27 made them incomparable. Assert that
// various kinds of errors have a functional equality operator, even
// if the result of that equality is always false.
func TestErrorEquality(t *testing.T) {
	vals := []error{
		nil,
		io.EOF,
		errors.New("EOF"),
		New("EOF"),
		Errorf("EOF"),
		Wrap(io.EOF, "EOF"),
		Wrapf(io.EOF, "EOF%d", 2),
		WithMessage(nil, "whoops"),
		WithMessage(io.EOF, "whoops"),
		WithStack(io.EOF),
		WithStack(nil),
	}

	for i := range vals {
		for j := range vals {
			_ = vals[i] == vals[j] // mustn't panic
		}
	}
}

func TestParseCoder(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantHTTPCode  int
		wantString    string
		wantCode      int
		wantReference string
		wantNamespace string
	}{
		{
			name:          "simple error returns unknownCoder",
			err:           fmt.Errorf("yes error"),
			wantHTTPCode:  500,
			wantString:    "An internal server error occurred",
			wantCode:      1,
			wantReference: "http://github.com/ClessLi/component-base/pkg/errors/README.md",
			wantNamespace: "base",
		},
		{
			name:          "withCode error returns registered coder",
			err:           WithCode("test", ConfigurationNotValid, "internal error message"),
			wantHTTPCode:  500,
			wantString:    "ConfigurationNotValid error",
			wantCode:      1000,
			wantReference: "",
			wantNamespace: "test",
		},
		{
			name:          "nil error returns nil",
			err:           nil,
			wantHTTPCode:  0,
			wantString:    "",
			wantCode:      0,
			wantReference: "",
			wantNamespace: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coder := ParseCoder(tt.err)
			if tt.err == nil {
				if coder != nil {
					t.Errorf("ParseCoder(nil) = %v, want nil", coder)
				}
				return
			}
			if coder.HTTPStatus() != tt.wantHTTPCode {
				t.Errorf("HTTPStatus() = %d, want %d", coder.HTTPStatus(), tt.wantHTTPCode)
			}
			if coder.String() != tt.wantString {
				t.Errorf("String() = %q, want %q", coder.String(), tt.wantString)
			}
			if coder.Code() != tt.wantCode {
				t.Errorf("Code() = %d, want %d", coder.Code(), tt.wantCode)
			}
			if coder.Reference() != tt.wantReference {
				t.Errorf("Reference() = %q, want %q", coder.Reference(), tt.wantReference)
			}
			if coder.Namespace() != tt.wantNamespace {
				t.Errorf("Namespace() = %q, want %q", coder.Namespace(), tt.wantNamespace)
			}
		})
	}
}

func TestWithCodeIs(t *testing.T) {
	err1 := WithCode("test", ConfigurationNotValid, "error 1")
	err2 := WithCode("test", ConfigurationNotValid, "error 2")
	err3 := WithCode("other", ConfigurationNotValid, "error 3")
	err4 := WithCode("test", ErrInvalidJSON, "error 4")

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "same code and namespace matches",
			err:    err1,
			target: err2,
			want:   true,
		},
		{
			name:   "different namespace does not match",
			err:    err1,
			target: err3,
			want:   false,
		},
		{
			name:   "different code does not match",
			err:    err1,
			target: err4,
			want:   false,
		},
		{
			name:   "non-withCode target does not match",
			err:    err1,
			target: fmt.Errorf("simple error"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := err1.(*withCode).Is(tt.target)
			if got != tt.want {
				t.Errorf("withCode.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithCodeAs(t *testing.T) {
	err := WithCode("test", ConfigurationNotValid, "error message")

	var target Coder
	got := err.(*withCode).As(&target)
	if !got {
		t.Errorf("withCode.As() = false, want true")
	}
	if target == nil {
		t.Errorf("target should be set to coder")
	} else if target.Code() != ConfigurationNotValid {
		t.Errorf("target.Code() = %d, want %d", target.Code(), ConfigurationNotValid)
	}
	if target.Namespace() != "test" {
		t.Errorf("target.Namespace() = %q, want %q", target.Namespace(), "test")
	}

	// Test with non-Coder target
	var simpleErr error
	got2 := err.(*withCode).As(&simpleErr)
	if got2 {
		t.Errorf("withCode.As() for non-Coder target = true, want false")
	}
}

func TestAggregateGoroutines(t *testing.T) {
	tests := []struct {
		name    string
		funcs   []func() error
		wantNil bool
		wantLen int
	}{
		{
			name:    "empty functions returns nil",
			funcs:   []func() error{},
			wantNil: true,
			wantLen: 0,
		},
		{
			name:    "all functions succeed returns nil",
			funcs:   []func() error{func() error { return nil }, func() error { return nil }},
			wantNil: true,
			wantLen: 0,
		},
		{
			name:    "one function fails returns aggregate",
			funcs:   []func() error{func() error { return nil }, func() error { return fmt.Errorf("error 1") }},
			wantNil: false,
			wantLen: 1,
		},
		{
			name:    "all functions fail returns aggregate",
			funcs:   []func() error{func() error { return fmt.Errorf("error 1") }, func() error { return fmt.Errorf("error 2") }},
			wantNil: false,
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AggregateGoroutines(tt.funcs...)
			if tt.wantNil && got != nil {
				t.Errorf("AggregateGoroutines() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("AggregateGoroutines() = nil, want aggregate")
			}
			if got != nil && len(got.Errors()) != tt.wantLen {
				t.Errorf("AggregateGoroutines().Errors() = %d errors, want %d", len(got.Errors()), tt.wantLen)
			}
		})
	}
}

func TestFilterOut(t *testing.T) {
	err1 := fmt.Errorf("error 1")
	err2 := fmt.Errorf("error 2")
	agg := NewAggregate([]error{err1, err2})

	matchErr1 := func(err error) bool {
		return err.Error() == "error 1"
	}

	tests := []struct {
		name string
		err  error
		fns  []Matcher
		want error
	}{
		{
			name: "nil error returns nil",
			err:  nil,
			fns:  []Matcher{matchErr1},
			want: nil,
		},
		{
			name: "non-aggregate matching error returns error",
			err:  err2,
			fns:  []Matcher{matchErr1},
			want: err2,
		},
		{
			name: "non-aggregate non-matching error returns nil",
			err:  err1,
			fns:  []Matcher{matchErr1},
			want: nil,
		},
		{
			name: "aggregate with matching error filtered",
			err:  agg,
			fns:  []Matcher{matchErr1},
			want: NewAggregate([]error{err2}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterOut(tt.err, tt.fns...)
			if got == nil && tt.want != nil {
				t.Errorf("FilterOut() = nil, want %v", tt.want)
			}
			if got != nil && tt.want == nil {
				t.Errorf("FilterOut() = %v, want nil", got)
			}
			if got != nil && tt.want != nil {
				if got.Error() != tt.want.Error() {
					t.Errorf("FilterOut() = %q, want %q", got.Error(), tt.want.Error())
				}
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	err1 := fmt.Errorf("error 1")
	err2 := fmt.Errorf("error 2")
	err3 := fmt.Errorf("error 3")

	nestedAgg := NewAggregate([]error{err2, err3})
	topAgg := NewAggregate([]error{err1, nestedAgg})

	tests := []struct {
		name    string
		agg     Aggregate
		wantLen int
	}{
		{
			name:    "nil aggregate returns nil",
			agg:     nil,
			wantLen: 0,
		},
		{
			name:    "flat aggregate stays same",
			agg:     NewAggregate([]error{err1, err2}),
			wantLen: 2,
		},
		{
			name:    "nested aggregate flattens",
			agg:     topAgg,
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Flatten(tt.agg)
			if tt.agg == nil && got != nil {
				t.Errorf("Flatten(nil) = %v, want nil", got)
			}
			if got != nil && len(got.Errors()) != tt.wantLen {
				t.Errorf("Flatten().Errors() = %d, want %d", len(got.Errors()), tt.wantLen)
			}
		})
	}
}

func TestReduce(t *testing.T) {
	err1 := fmt.Errorf("error 1")
	singleAgg := NewAggregate([]error{err1})
	multiAgg := NewAggregate([]error{err1, fmt.Errorf("error 2")})

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "nil error returns nil",
			err:  nil,
			want: nil,
		},
		{
			name: "non-aggregate returns same",
			err:  err1,
			want: err1,
		},
		{
			name: "single-item aggregate returns first item",
			err:  singleAgg,
			want: err1,
		},
		{
			name: "multi-item aggregate returns same",
			err:  multiAgg,
			want: multiAgg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Reduce(tt.err)
			if got == nil && tt.want != nil {
				t.Errorf("Reduce() = nil, want %v", tt.want)
			}
			if got != nil && tt.want == nil {
				t.Errorf("Reduce() = %v, want nil", got)
			}
			if got != nil && tt.want != nil {
				if got.Error() != tt.want.Error() {
					t.Errorf("Reduce() = %q, want %q", got.Error(), tt.want.Error())
				}
			}
		})
	}
}
