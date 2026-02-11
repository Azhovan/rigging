package rigging

import (
	"reflect"
	"time"
)

func formatStructuredValue(v reflect.Value, secret bool) any {
	if secret {
		return "***redacted***"
	}

	if !v.IsValid() || (v.Kind() == reflect.Ptr && v.IsNil()) {
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Type() == durationType {
			if dur, ok := v.Interface().(time.Duration); ok {
				return dur.String()
			}
		}
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.String {
			slice := make([]string, v.Len())
			for i := 0; i < v.Len(); i++ {
				slice[i] = v.Index(i).String()
			}
			return slice
		}

		slice := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			slice[i] = v.Index(i).Interface()
		}
		return slice
	case reflect.Struct:
		if v.Type() == timeType {
			if t, ok := v.Interface().(time.Time); ok {
				return t.Format(time.RFC3339)
			}
		}
		return v.Interface()
	default:
		return v.Interface()
	}
}
