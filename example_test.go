package rigging_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Azhovan/rigging"
	"github.com/Azhovan/rigging/sourceenv"
)

// Example demonstrates basic configuration loading from multiple sources.
func Example() {
	// Define configuration structure
	type Config struct {
		Environment string `conf:"default:dev,oneof:prod,staging,dev"`
		Port        int    `conf:"default:8080,min:1024,max:65535"`
		Database    struct {
			Host     string `conf:"required"`
			Port     int    `conf:"default:5432"`
			User     string `conf:"required"`
			Password string `conf:"required,secret"`
		} `conf:"prefix:database"`
	}

	// Set up environment variables for this example
	os.Setenv("EXAMPLE_DATABASE__HOST", "localhost")
	os.Setenv("EXAMPLE_DATABASE__USER", "testuser")
	os.Setenv("EXAMPLE_DATABASE__PASSWORD", "testpass")
	defer func() {
		os.Unsetenv("EXAMPLE_DATABASE__HOST")
		os.Unsetenv("EXAMPLE_DATABASE__USER")
		os.Unsetenv("EXAMPLE_DATABASE__PASSWORD")
	}()

	// Create loader with environment source (using prefix to avoid conflicts)
	loader := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXAMPLE_"})).
		Strict(true)

	// Load configuration
	cfg, err := loader.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Environment: %s\n", cfg.Environment)
	fmt.Printf("Port: %d\n", cfg.Port)
	fmt.Printf("Database Host: %s\n", cfg.Database.Host)
	fmt.Printf("Database User: %s\n", cfg.Database.User)

	// Output:
	// Environment: dev
	// Port: 8080
	// Database Host: localhost
	// Database User: testuser
}

// ExampleLoader_Load demonstrates loading configuration with validation.
func ExampleLoader_Load() {
	type Config struct {
		APIKey     string        `conf:"required,secret"`
		Timeout    time.Duration `conf:"default:30s"`
		MaxRetries int           `conf:"default:3,min:1,max:10"`
	}

	os.Setenv("EXLOAD_APIKEY", "test-key-12345")
	defer os.Unsetenv("EXLOAD_APIKEY")

	loader := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXLOAD_"}))

	cfg, err := loader.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Timeout: %v\n", cfg.Timeout)
	fmt.Printf("MaxRetries: %d\n", cfg.MaxRetries)
	fmt.Printf("APIKey: %s\n", cfg.APIKey)

	// Output:
	// Timeout: 30s
	// MaxRetries: 3
	// APIKey: test-key-12345
}

// ExampleLoader_WithValidator demonstrates custom validation.
func ExampleLoader_WithValidator() {
	type Config struct {
		Environment string `conf:"default:dev"`
		DebugMode   bool   `conf:"default:false"`
	}

	loader := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXVAL_"})).
		WithValidator(rigging.ValidatorFunc[Config](func(ctx context.Context, cfg *Config) error {
			// Cross-field validation: debug mode not allowed in production
			if cfg.Environment == "prod" && cfg.DebugMode {
				return &rigging.ValidationError{
					FieldErrors: []rigging.FieldError{{
						FieldPath: "DebugMode",
						Code:      "invalid_prod_debug",
						Message:   "debug mode cannot be enabled in production",
					}},
				}
			}
			return nil
		}))

	cfg, err := loader.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Environment: %s\n", cfg.Environment)
	fmt.Printf("DebugMode: %t\n", cfg.DebugMode)

	// Output:
	// Environment: dev
	// DebugMode: false
}

// ExampleDumpEffective demonstrates dumping configuration with secret redaction.
func ExampleDumpEffective() {
	type Config struct {
		APIKey   string        `conf:"secret,required"`
		Endpoint string        `conf:"required"`
		Timeout  time.Duration `conf:"default:30s"`
	}

	os.Setenv("EXDMPEFF_APIKEY", "super-secret-key")
	os.Setenv("EXDMPEFF_ENDPOINT", "https://api.example.com")
	defer func() {
		os.Unsetenv("EXDMPEFF_APIKEY")
		os.Unsetenv("EXDMPEFF_ENDPOINT")
	}()

	loader := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXDMPEFF_"}))

	cfg, err := loader.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Dump configuration (secrets will be redacted)
	rigging.DumpEffective(os.Stdout, cfg)

	// Output:
	// apikey: ***redacted***
	// endpoint: "https://api.example.com"
	// timeout: 30s
}

// ExampleDumpEffective_withSources demonstrates dumping with source attribution.
func ExampleDumpEffective_withSources() {
	type Config struct {
		Port int    `conf:"default:8080"`
		Host string `conf:"default:localhost"`
	}

	os.Setenv("EXDUMP_PORT", "9090")
	defer os.Unsetenv("EXDUMP_PORT")

	loader := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXDUMP_"}))

	cfg, err := loader.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Dump with source information
	rigging.DumpEffective(os.Stdout, cfg, rigging.WithSources())

	// Output:
	// port: 9090 (source: env:EXDUMP_PORT)
	// host: "localhost" (source: default)
}

// ExampleDumpEffective_asJSON demonstrates JSON output format.
func ExampleDumpEffective_asJSON() {
	type Config struct {
		Environment string `conf:"default:dev"`
		Port        int    `conf:"default:8080"`
	}

	os.Setenv("EXJSON_ENVIRONMENT", "production")
	defer os.Unsetenv("EXJSON_ENVIRONMENT")

	loader := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXJSON_"}))

	cfg, err := loader.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Dump as JSON with source attribution
	rigging.DumpEffective(os.Stdout, cfg, rigging.AsJSON(), rigging.WithSources())

	// Output:
	// {
	//   "environment": {
	//     "source": "env:EXJSON_ENVIRONMENT",
	//     "value": "production"
	//   },
	//   "port": {
	//     "source": "default",
	//     "value": 8080
	//   }
	// }
}

// ExampleGetProvenance demonstrates querying configuration provenance.
func ExampleGetProvenance() {
	type Config struct {
		Host string `conf:"required"`
		Port int    `conf:"default:8080"`
	}

	os.Setenv("EXPROV_HOST", "example.com")
	defer os.Unsetenv("EXPROV_HOST")

	loader := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXPROV_"}))

	cfg, err := loader.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Query provenance
	prov, ok := rigging.GetProvenance(cfg)
	if ok {
		for _, field := range prov.Fields {
			fmt.Printf("%s from %s\n", field.FieldPath, field.SourceName)
		}
	}

	// Output:
	// Host from env:EXPROV_HOST
	// Port from default
}

// ExampleOptional demonstrates using Optional fields.
func ExampleOptional() {
	type Config struct {
		Timeout    rigging.Optional[time.Duration]
		MaxRetries rigging.Optional[int]
	}

	cfg := &Config{}

	// Set Timeout but not MaxRetries
	cfg.Timeout = rigging.Optional[time.Duration]{
		Value: 30 * time.Second,
		Set:   true,
	}

	// Check if Timeout was set
	if timeout, ok := cfg.Timeout.Get(); ok {
		fmt.Printf("Timeout is set to: %v\n", timeout)
	}

	// Check if MaxRetries was set
	if _, ok := cfg.MaxRetries.Get(); !ok {
		fmt.Println("MaxRetries was not set")
	}

	// Use OrDefault for fallback values
	maxRetries := cfg.MaxRetries.OrDefault(3)
	fmt.Printf("MaxRetries (with default): %d\n", maxRetries)

	// Output:
	// Timeout is set to: 30s
	// MaxRetries was not set
	// MaxRetries (with default): 3
}

// ExampleLoader_Strict demonstrates strict mode behavior.
func ExampleLoader_Strict() {
	type Config struct {
		Host string `conf:"required"`
		Port int    `conf:"default:8080"`
	}

	// Set an unknown configuration key
	os.Setenv("EXSTRICT_HOST", "localhost")
	os.Setenv("EXSTRICT_UNKNOWNKEY", "some-value")
	defer func() {
		os.Unsetenv("EXSTRICT_HOST")
		os.Unsetenv("EXSTRICT_UNKNOWNKEY")
	}()

	// Strict mode enabled (default)
	loader := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXSTRICT_"})).
		Strict(true)

	_, err := loader.Load(context.Background())
	if err != nil {
		fmt.Printf("Strict mode error detected\n")
	}

	// Strict mode disabled
	loader = rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXSTRICT_"})).
		Strict(false)

	cfg, err := loader.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Host: %s (strict mode disabled)\n", cfg.Host)

	// Output:
	// Strict mode error detected
	// Host: localhost (strict mode disabled)
}

// ExampleValidationError demonstrates handling validation errors.
func ExampleValidationError() {
	type Config struct {
		Port int    `conf:"required,min:1024,max:65535"`
		Env  string `conf:"required,oneof:prod,staging,dev"`
	}

	// Set invalid values
	os.Setenv("EXVERR_PORT", "80")        // Below minimum
	os.Setenv("EXVERR_ENV", "production") // Not in oneof list
	defer func() {
		os.Unsetenv("EXVERR_PORT")
		os.Unsetenv("EXVERR_ENV")
	}()

	loader := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXVERR_"}))

	_, err := loader.Load(context.Background())
	if err != nil {
		if valErr, ok := err.(*rigging.ValidationError); ok {
			fmt.Printf("Validation failed with %d errors\n", len(valErr.FieldErrors))
		}
	}

	// Output:
	// Validation failed with 2 errors
}

// staticSource is a custom source that provides static configuration.
// This demonstrates how to implement the Source interface.
type staticSource struct {
	data map[string]any
}

// Load implements the Source interface for staticSource.
func (s *staticSource) Load(ctx context.Context) (map[string]any, error) {
	return s.data, nil
}

// Watch implements the Source interface for staticSource.
func (s *staticSource) Watch(ctx context.Context) (<-chan rigging.ChangeEvent, error) {
	return nil, rigging.ErrWatchNotSupported
}

// Name implements the Source interface for staticSource.
func (s *staticSource) Name() string {
	return "static"
}

// ExampleSource demonstrates implementing a custom source.
func ExampleSource() {
	// Create a custom source with static data
	source := &staticSource{
		data: map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	}

	type Config struct {
		Host string `conf:"required"`
		Port int    `conf:"required"`
	}

	loader := rigging.NewLoader[Config]().
		WithSource(source)

	cfg, err := loader.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Host: %s, Port: %d\n", cfg.Host, cfg.Port)

	// Output:
	// Host: localhost, Port: 8080
}

// Example_envCaseSensitive demonstrates case-sensitive prefix matching.
func Example_envCaseSensitive() {
	type Config struct {
		Host string `conf:"required"`
		Port int    `conf:"required"`
	}

	// Set environment variables with different cases
	os.Setenv("APP_HOST", "prod.example.com")
	os.Setenv("APP_PORT", "8080")
	os.Setenv("app_host", "dev.example.com") // lowercase prefix
	os.Setenv("app_port", "9090")            // lowercase prefix
	defer func() {
		os.Unsetenv("APP_HOST")
		os.Unsetenv("APP_PORT")
		os.Unsetenv("app_host")
		os.Unsetenv("app_port")
	}()

	// Case-insensitive (default) - matches all variations
	// Both APP_* and app_* are loaded, later ones override
	loaderInsensitive := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{
			Prefix:        "APP_",
			CaseSensitive: false, // default
		}))

	cfg, err := loaderInsensitive.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Case-insensitive: Host=%s, Port=%d\n", cfg.Host, cfg.Port)

	// Case-sensitive - only exact match (APP_* only)
	loaderSensitive := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{
			Prefix:        "APP_",
			CaseSensitive: true,
		}))

	cfg2, err := loaderSensitive.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Case-sensitive: Host=%s, Port=%d\n", cfg2.Host, cfg2.Port)

	// Output:
	// Case-insensitive: Host=dev.example.com, Port=9090
	// Case-sensitive: Host=prod.example.com, Port=8080
}

// Example_underscoreNormalization demonstrates how underscores in environment
// variables are normalized to match camelCase field names.
func Example_underscoreNormalization() {
	type Config struct {
		MaxConnections int
		APIKey         string
	}

	// All these environment variable formats match the same fields:
	// - Single underscores are stripped for flexible matching
	// - Double underscores (__) create nested structures
	// - Everything is case-insensitive

	os.Setenv("EXNORM_MAX_CONNECTIONS", "100") // Underscores → matches MaxConnections
	os.Setenv("EXNORM_API_KEY", "secret-key")  // Underscores → matches APIKey
	defer func() {
		os.Unsetenv("EXNORM_MAX_CONNECTIONS")
		os.Unsetenv("EXNORM_API_KEY")
	}()

	loader := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXNORM_"}))

	cfg, err := loader.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("MaxConnections: %d\n", cfg.MaxConnections)
	fmt.Printf("APIKey: %s\n", cfg.APIKey)

	// Output:
	// MaxConnections: 100
	// APIKey: secret-key
}

// ExampleLoader_Watch demonstrates configuration watching.
// Built-in sources (sourceenv, sourcefile) don't support watching yet.
// Custom sources can implement Watch() to enable hot-reload.
func ExampleLoader_Watch() {
	type Config struct {
		Host string `conf:"required"`
		Port int    `conf:"default:8080"`
	}

	source := &staticSource{
		data: map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	}

	loader := rigging.NewLoader[Config]().
		WithSource(source)

	// Watch starts monitoring and returns channels
	snapshots, errors, err := loader.Watch(context.Background())
	if err != nil {
		fmt.Printf("Watch failed: %v\n", err)
		return
	}

	// Receive initial snapshot
	snapshot := <-snapshots
	fmt.Printf("Initial config loaded (version %d)\n", snapshot.Version)

	// In a real application, you would monitor snapshots and errors
	// in a goroutine for configuration updates
	_ = errors

	// Output:
	// Initial config loaded (version 1)
}
