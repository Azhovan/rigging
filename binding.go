package rigging

import (
	"strings"
)

// tagConfig holds parsed directives from a struct field's `conf` tag.
type tagConfig struct {
	env      string   // Environment variable name (env:VAR_NAME)
	name     string   // Custom key path (name:custom.path)
	prefix   string   // Prefix for nested structs (prefix:foo)
	defValue string   // Default value (default:value)
	min      string   // Minimum constraint (min:N)
	max      string   // Maximum constraint (max:M)
	oneof    []string // Allowed values (oneof:a,b,c)
	required bool     // Field is required (required or required:true)
	secret   bool     // Field is secret (secret or secret:true)
	hasDefault bool   // Whether a default directive was present
}

// parseTag parses a `conf` struct tag into a structured tagConfig.
// Tag format: "directive1:value1,directive2:value2,..."
// Boolean directives can omit `:true` (e.g., "required" == "required:true")
func parseTag(tag string) tagConfig {
	cfg := tagConfig{}
	
	if tag == "" {
		return cfg
	}
	
	// Parse directives manually to handle oneof values that contain commas
	directives := splitDirectives(tag)
	
	for _, directive := range directives {
		directive = strings.TrimSpace(directive)
		if directive == "" {
			continue
		}
		
		// Split by colon to separate directive name from value
		parts := strings.SplitN(directive, ":", 2)
		name := strings.TrimSpace(parts[0])
		var value string
		if len(parts) > 1 {
			value = parts[1] // Don't trim value - empty strings may be intentional
		}
		
		switch name {
		case "env":
			cfg.env = value
		case "name":
			cfg.name = value
		case "prefix":
			cfg.prefix = value
		case "default":
			cfg.defValue = value
			cfg.hasDefault = true
		case "min":
			cfg.min = value
		case "max":
			cfg.max = value
		case "oneof":
			// oneof values are already part of this directive's value
			if value != "" {
				cfg.oneof = strings.Split(value, ",")
				// Trim whitespace from each option
				for i := range cfg.oneof {
					cfg.oneof[i] = strings.TrimSpace(cfg.oneof[i])
				}
			}
		case "required":
			// Boolean directive: no value or explicit "true" means true
			if value == "" || value == "true" {
				cfg.required = true
			} else if value == "false" {
				cfg.required = false
			} else {
				// Invalid value, default to true for safety
				cfg.required = true
			}
		case "secret":
			// Boolean directive: no value or explicit "true" means true
			if value == "" || value == "true" {
				cfg.secret = true
			} else if value == "false" {
				cfg.secret = false
			} else {
				// Invalid value, default to true for safety
				cfg.secret = true
			}
		}
	}
	
	return cfg
}

// splitDirectives splits a tag string into individual directives,
// handling the special case where oneof values contain commas.
func splitDirectives(tag string) []string {
	var directives []string
	var current strings.Builder
	inOneof := false
	
	for i := 0; i < len(tag); i++ {
		ch := tag[i]
		
		// Check if we're entering an oneof directive
		if !inOneof && i+6 <= len(tag) && tag[i:i+6] == "oneof:" {
			inOneof = true
			current.WriteString("oneof:")
			i += 5 // Skip past "oneof:"
			continue
		}
		
		if ch == ',' {
			if inOneof {
				// Check if the next directive starts after this comma
				// Look ahead to see if we have a known directive name
				remaining := tag[i+1:]
				if startsWithDirective(remaining) {
					// This comma ends the oneof directive
					inOneof = false
					directives = append(directives, current.String())
					current.Reset()
					continue
				} else {
					// This comma is part of oneof values
					current.WriteByte(ch)
				}
			} else {
				// Regular comma separator between directives
				directives = append(directives, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(ch)
		}
	}
	
	// Add the last directive
	if current.Len() > 0 {
		directives = append(directives, current.String())
	}
	
	return directives
}

// startsWithDirective checks if a string starts with a known directive name.
func startsWithDirective(s string) bool {
	s = strings.TrimSpace(s)
	directives := []string{"env:", "name:", "prefix:", "default:", "min:", "max:", "oneof:", "required", "secret"}
	for _, d := range directives {
		if strings.HasPrefix(s, d) {
			return true
		}
	}
	return false
}
