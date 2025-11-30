// Package rigging provides type-safe configuration management with validation and provenance tracking.
//
// Quick Start:
//
//	type Config struct {
//	    Port int    `conf:"default:8080,min:1024"`
//	    Host string `conf:"required"`
//	}
//
//	loader := rigging.NewLoader[Config]().
//	    WithSource(sourcefile.New("config.yaml", sourcefile.Options{})).
//	    WithSource(sourceenv.New(sourceenv.Options{Prefix: "APP_"}))
//
//	cfg, err := loader.Load(context.Background())
//
// Tag directives: env:VAR, default:val, required, min:N, max:N, oneof:a,b,c, secret, prefix:path, name:path
//
// See example_test.go and README.md for detailed usage.
package rigging
