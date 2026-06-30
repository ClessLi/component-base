package v1

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
