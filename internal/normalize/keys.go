package normalize

import (
	"strings"
	"unicode"
)

// ToLowerDotPath normalizes a key to lowercase dot-separated path.
// Double underscores (__) -> dots, single underscores are preserved.
// Examples: FOO__BAR -> foo.bar, DB_MAX -> db_max, MAX_CONNECTIONS -> max_connections
func ToLowerDotPath(key string) string {
	return strings.ToLower(strings.ReplaceAll(key, "__", "."))
}

// DeriveFieldPath derives a normalized key from a field name.
// It converts Go-style names to snake_case for consistent matching.
// Examples: Host -> host, APIKey -> api_key, MaxConnections -> max_connections
func DeriveFieldPath(fieldName string) string {
	if fieldName == "" {
		return ""
	}

	var b strings.Builder
	runes := []rune(fieldName)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				var next rune
				hasNext := i+1 < len(runes)
				if hasNext {
					next = runes[i+1]
				}

				// Insert separator on lower->upper transitions and acronym boundaries.
				if (unicode.IsLower(prev) || unicode.IsDigit(prev) || (hasNext && unicode.IsLower(next))) && prev != '_' {
					b.WriteByte('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}

		b.WriteRune(unicode.ToLower(r))
	}

	return b.String()
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
