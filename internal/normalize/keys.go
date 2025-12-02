package normalize

import (
	"strings"
	"unicode"
)

// ToLowerDotPath normalizes a key to lowercase dot-separated path.
// Double underscores (__) → dots, all other underscores stripped.
// Examples: FOO__BAR → foo.bar, DB_MAX → dbmax, MAX_CONNECTIONS → maxconnections
func ToLowerDotPath(key string) string {
	// First convert double underscores to dots
	normalized := strings.ReplaceAll(key, "__", ".")
	// Then strip all remaining single underscores
	normalized = strings.ReplaceAll(normalized, "_", "")
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
