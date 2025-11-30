// Package sourcefile loads configuration from YAML, JSON, or TOML files.
//
// Format is auto-detected from extension (.yaml, .json, .toml).
//
// Example:
//
//	source := sourcefile.New("config.yaml", sourcefile.Options{Required: true})
//	loader := rigging.NewLoader[Config]().WithSource(source)
package sourcefile
