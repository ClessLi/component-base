package v1

import (
	kitlog "github.com/go-kit/kit/log"
)

// Option defines a function that configures the formatter.
// Options are applied during formatter initialization.
type Option func(*formatter)

// WithFieldExtractor adds a custom field extractor to the formatter.
// Multiple extractors can be composed and are executed in order.
//
// Example:
//
//	methodlog.New(ctx, handler,
//	    methodlog.WithFieldExtractor(methodlog.HTTPFieldExtractor),
//	    methodlog.WithFieldExtractor(customExtractor),
//	).Do()
func WithFieldExtractor(extractor FieldExtractor) Option {
	return func(f *formatter) {
		if extractor != nil {
			f.fieldExtractors = append(f.fieldExtractors, extractor)
		}
	}
}

// WithLogger injects a fixed logger used for all log output of this Do,
// including the panic record, overriding the lazy per-Do resolution from
// the global log state. Useful for capturing log output in tests.
// A nil logger is ignored and lazy resolution applies.
//
// Example:
//
//	methodlog.New(ctx, handler, methodlog.WithLogger(captureLogger)).Do()
func WithLogger(logger kitlog.Logger) Option {
	return func(f *formatter) {
		if logger != nil {
			f.logger = logger
		}
	}
}
