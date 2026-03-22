package rigging

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
)

var schemaValidationCache sync.Map

func configType[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func cloneFieldErrors(errors []FieldError) []FieldError {
	if len(errors) == 0 {
		return nil
	}

	cloned := make([]FieldError, len(errors))
	copy(cloned, errors)
	return cloned
}

func getSchemaValidationErrors(rootType reflect.Type) []FieldError {
	rootType = dereferenceType(rootType)
	if rootType == nil || rootType.Kind() != reflect.Struct {
		return nil
	}

	if cached, ok := schemaValidationCache.Load(rootType); ok {
		if errors, ok := cached.([]FieldError); ok {
			return cloneFieldErrors(errors)
		}
	}

	errors := validateSchemaType(rootType, "")
	schemaValidationCache.Store(rootType, cloneFieldErrors(errors))
	return cloneFieldErrors(errors)
}

func validateSchemaType(rootType reflect.Type, parentFieldPath string) []FieldError {
	rootType = dereferenceType(rootType)
	if rootType == nil || rootType.Kind() != reflect.Struct {
		return nil
	}

	var errors []FieldError

	for _, meta := range getStructFieldMeta(rootType) {
		field := meta.field
		fieldType := field.Type
		fieldPath := field.Name
		if parentFieldPath != "" {
			fieldPath = parentFieldPath + "." + field.Name
		}

		errors = append(errors, validateTagSchema(fieldType, fieldPath, meta.tagCfg)...)

		if isOptionalType(fieldType) {
			innerType := fieldType.Field(0).Type
			if innerType.Kind() == reflect.Struct && innerType.PkgPath() != "time" {
				errors = append(errors, validateSchemaType(innerType, fieldPath)...)
			}
			continue
		}

		if fieldType.Kind() == reflect.Struct && fieldType.PkgPath() != "time" {
			errors = append(errors, validateSchemaType(fieldType, fieldPath)...)
		}
	}

	return errors
}

func validateTagSchema(fieldType reflect.Type, fieldPath string, tags tagConfig) []FieldError {
	var errors []FieldError

	for _, msg := range tags.parseErrors {
		errors = append(errors, FieldError{
			FieldPath: fieldPath,
			Code:      ErrCodeInvalidTag,
			Message:   msg,
		})
	}

	errors = append(errors, validateMinMaxTagSchema(fieldType, fieldPath, tags)...)
	return errors
}

func validateMinMaxTagSchema(fieldType reflect.Type, fieldPath string, tags tagConfig) []FieldError {
	if fieldType == nil {
		return nil
	}

	if isOptionalType(fieldType) {
		fieldType = fieldType.Field(0).Type
	}

	var errors []FieldError

	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if tags.min != "" {
			if _, err := strconv.ParseInt(tags.min, 10, 64); err != nil {
				errors = append(errors, FieldError{
					FieldPath: fieldPath,
					Code:      ErrCodeInvalidTag,
					Message:   fmt.Sprintf("invalid min constraint %q: must be a valid integer", tags.min),
				})
			}
		}
		if tags.max != "" {
			if _, err := strconv.ParseInt(tags.max, 10, 64); err != nil {
				errors = append(errors, FieldError{
					FieldPath: fieldPath,
					Code:      ErrCodeInvalidTag,
					Message:   fmt.Sprintf("invalid max constraint %q: must be a valid integer", tags.max),
				})
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if tags.min != "" {
			if _, err := strconv.ParseUint(tags.min, 10, 64); err != nil {
				errors = append(errors, FieldError{
					FieldPath: fieldPath,
					Code:      ErrCodeInvalidTag,
					Message:   fmt.Sprintf("invalid min constraint %q: must be a valid unsigned integer", tags.min),
				})
			}
		}
		if tags.max != "" {
			if _, err := strconv.ParseUint(tags.max, 10, 64); err != nil {
				errors = append(errors, FieldError{
					FieldPath: fieldPath,
					Code:      ErrCodeInvalidTag,
					Message:   fmt.Sprintf("invalid max constraint %q: must be a valid unsigned integer", tags.max),
				})
			}
		}
	case reflect.Float32, reflect.Float64:
		if tags.min != "" {
			if _, err := strconv.ParseFloat(tags.min, 64); err != nil {
				errors = append(errors, FieldError{
					FieldPath: fieldPath,
					Code:      ErrCodeInvalidTag,
					Message:   fmt.Sprintf("invalid min constraint %q: must be a valid number", tags.min),
				})
			}
		}
		if tags.max != "" {
			if _, err := strconv.ParseFloat(tags.max, 64); err != nil {
				errors = append(errors, FieldError{
					FieldPath: fieldPath,
					Code:      ErrCodeInvalidTag,
					Message:   fmt.Sprintf("invalid max constraint %q: must be a valid number", tags.max),
				})
			}
		}
	case reflect.String:
		if tags.min != "" {
			if _, err := strconv.Atoi(tags.min); err != nil {
				errors = append(errors, FieldError{
					FieldPath: fieldPath,
					Code:      ErrCodeInvalidTag,
					Message:   fmt.Sprintf("invalid min constraint %q: must be a valid integer", tags.min),
				})
			}
		}
		if tags.max != "" {
			if _, err := strconv.Atoi(tags.max); err != nil {
				errors = append(errors, FieldError{
					FieldPath: fieldPath,
					Code:      ErrCodeInvalidTag,
					Message:   fmt.Sprintf("invalid max constraint %q: must be a valid integer", tags.max),
				})
			}
		}
	}

	return errors
}

func dereferenceType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
