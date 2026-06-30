package utils

import "reflect"

// IsNil checks if a generic type is nil using reflection
func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	return v.Kind() == reflect.Ptr && v.IsNil()
}

// IsPointer checks if a generic type is a pointer using reflection
func IsPointer(i interface{}) bool {
	return reflect.TypeOf(i).Kind() == reflect.Pointer
}

func IsPointerType[A any]() bool {
	var a A
	return IsPointer(a)
}

// IsNullOrZeroValue checks if a value is nil, empty string, or zero value
func IsNullOrZeroValue(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return v == ""
	case int, int8, int16, int32, int64:
		return v == 0
	case uint, uint8, uint16, uint32, uint64:
		return v == 0
	case float32, float64:
		return v == 0.0
	case bool:
		return !v // false is considered zero value for bool
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		// For other types, check if it's the zero value of its type
		return value == reflect.Zero(reflect.TypeOf(value)).Interface()
	}
}
