// Package sourceenv loads configuration from environment variables.
//
// Key normalization: FOO__BAR → foo.bar, FOO_BAR → foo_bar
//
// Example:
//
//	source := sourceenv.New(sourceenv.Options{Prefix: "APP_"})
//	loader := rigging.NewLoader[Config]().WithSource(source)
package sourceenv
