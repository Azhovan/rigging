package rigging

import "reflect"

type walkedField struct {
	fieldPath  string
	keyPath    string
	value      reflect.Value
	provenance *FieldProvenance
	secret     bool
	optional   bool
	set        bool
}

func getStructRootValue(v any) (reflect.Value, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return reflect.Value{}, false
	}

	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return reflect.Value{}, false
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	return rv, true
}

func buildProvenanceMap[T any](cfg *T) map[string]*FieldProvenance {
	provenanceMap := make(map[string]*FieldProvenance)

	prov, _ := GetProvenance(cfg)
	if prov == nil {
		return provenanceMap
	}

	for i := range prov.Fields {
		provenanceMap[prov.Fields[i].FieldPath] = &prov.Fields[i]
	}

	return provenanceMap
}

func resolveKeyPath(fieldName string, tagCfg tagConfig, keyPathPrefix string, prov *FieldProvenance) string {
	if prov != nil && prov.KeyPath != "" {
		return prov.KeyPath
	}

	if tagCfg.name != "" {
		return tagCfg.name
	}

	keyPath := deriveFieldKey(fieldName)
	if keyPathPrefix != "" {
		keyPath = keyPathPrefix + "." + keyPath
	}
	return keyPath
}

func walkFlatFields(v reflect.Value, fieldPathPrefix, keyPathPrefix string, provenanceMap map[string]*FieldProvenance, inheritedSecret bool, visit func(w walkedField)) {
	t := v.Type()
	for _, meta := range getStructFieldMeta(t) {
		field := meta.field
		fieldValue := v.Field(meta.index)
		tagCfg := meta.tagCfg

		fieldPath := field.Name
		if fieldPathPrefix != "" {
			fieldPath = fieldPathPrefix + "." + field.Name
		}

		var prov *FieldProvenance
		if p, ok := provenanceMap[fieldPath]; ok {
			prov = p
		}

		isSecret := shouldRedactField(tagCfg, prov, inheritedSecret)
		keyPath := resolveKeyPath(field.Name, tagCfg, keyPathPrefix, prov)

		if fieldValue.Kind() == reflect.Struct && field.Type != timeType {
			if isOptionalType(field.Type) {
				setField := fieldValue.FieldByName("Set")
				valueField := fieldValue.FieldByName("Value")
				isSet := setField.IsValid() && setField.Bool() && valueField.IsValid()

				w := walkedField{
					fieldPath:  fieldPath,
					keyPath:    keyPath,
					provenance: prov,
					secret:     isSecret,
					optional:   true,
					set:        isSet,
				}
				if isSet {
					w.value = valueField
				}

				visit(w)
				continue
			}

			nestedKeyPrefix := keyPath
			if tagCfg.prefix != "" {
				nestedKeyPrefix = tagCfg.prefix
			}

			walkFlatFields(fieldValue, fieldPath, nestedKeyPrefix, provenanceMap, isSecret, visit)
			continue
		}

		visit(walkedField{
			fieldPath:  fieldPath,
			keyPath:    keyPath,
			value:      fieldValue,
			provenance: prov,
			secret:     isSecret,
			set:        true,
		})
	}
}
