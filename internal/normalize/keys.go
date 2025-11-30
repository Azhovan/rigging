package normalize

import (
	"strings"
	"unicode"
)

// ToLowerDotPath normalizes a key to lowercase dot-separated path.
// Double underscores (__) → dots, single underscores preserved.
// Examples: FOO__BAR → foo.bar, DB_MAX → db_max
func ToLowerDotPath(key string) string {
	normalized := strings.ReplaceAll(key, "__", ".")
	return strings.ToLower(normalized)
}

// DeriveFieldPath lowercases the first letter of a field name.
// Examples: Host → host, APIKey → aPIKey
func DeriveFieldPath(fieldName string) string {
	if fieldName == "" {
		return ""
	}

	runes := []rune(fieldName)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// ApplyPrefix combines prefix with key: prefix.key or key if prefix is empty.
func ApplyPrefix(prefix, key string) string {
	if prefix == "" {
		return key
	}
	if key == "" {
		return prefix
	}
	return prefix + "." + key
}
