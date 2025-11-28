package normalize

import (
	"strings"
	"unicode"
)

// ToLowerDotPath normalizes a configuration key to a lowercase dot-separated path.
// Double underscores (__) are treated as level separators and converted to dots.
// Single underscores within a level are preserved.
// Examples:
//   - "FOO__BAR" → "foo.bar"
//   - "DB_MAX_CONNECTIONS" → "db_max_connections"
//   - "API__RATE_LIMIT" → "api.rate_limit"
func ToLowerDotPath(key string) string {
	normalized := strings.ReplaceAll(key, "__", ".")
	return strings.ToLower(normalized)
}

// DeriveFieldPath derives a configuration key path from a struct field name.
// It lowercases the first letter of the field name.
// Examples:
//   - "Host" → "host"
//   - "Port" → "port"
//   - "APIKey" → "aPIKey"
func DeriveFieldPath(fieldName string) string {
	if fieldName == "" {
		return ""
	}

	// Convert first rune to lowercase
	runes := []rune(fieldName)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// ApplyPrefix combines a prefix with a key to create a nested configuration path.
// If prefix is empty, returns the key unchanged.
// Otherwise, returns "prefix.key".
// Examples:
//   - ApplyPrefix("database", "host") → "database.host"
//   - ApplyPrefix("", "host") → "host"
//   - ApplyPrefix("api", "rate_limit") → "api.rate_limit"
func ApplyPrefix(prefix, key string) string {
	if prefix == "" {
		return key
	}
	if key == "" {
		return prefix
	}
	return prefix + "." + key
}
