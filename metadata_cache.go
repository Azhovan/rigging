package rigging

import (
	"reflect"
	"sync"
)

type structFieldMeta struct {
	index  int
	field  reflect.StructField
	tagCfg tagConfig
}

var structFieldMetaCache sync.Map

func getStructFieldMeta(t reflect.Type) []structFieldMeta {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	if cached, ok := structFieldMetaCache.Load(t); ok {
		if fields, ok := cached.([]structFieldMeta); ok {
			return cloneStructFieldMeta(fields)
		}
	}

	fields := make([]structFieldMeta, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fields = append(fields, structFieldMeta{
			index:  i,
			field:  field,
			tagCfg: parseTag(field.Tag.Get("conf")),
		})
	}

	structFieldMetaCache.Store(t, fields)
	return cloneStructFieldMeta(fields)
}

func cloneStructFieldMeta(fields []structFieldMeta) []structFieldMeta {
	if len(fields) == 0 {
		return nil
	}

	cloned := make([]structFieldMeta, len(fields))
	copy(cloned, fields)
	return cloned
}
