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
		Apikey     string        `conf:"required,secret"`
		Timeout    time.Duration `conf:"default:30s"`
		Maxretries int           `conf:"default:3,min:1,max:10"`
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
	fmt.Printf("Maxretries: %d\n", cfg.Maxretries)

	// Output:
	// Timeout: 30s
	// Maxretries: 3
}

// ExampleLoader_WithValidator demonstrates custom validation.
func ExampleLoader_WithValidator() {
	type Config struct {
		Environment string `conf:"default:dev"`
		Debugmode   bool   `conf:"default:false"`
	}

	loader := rigging.NewLoader[Config]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXVAL_"})).
		WithValidator(rigging.ValidatorFunc[Config](func(ctx context.Context, cfg *Config) error {
			// Cross-field validation: debug mode not allowed in production
			if cfg.Environment == "prod" && cfg.Debugmode {
				return &rigging.ValidationError{
					FieldErrors: []rigging.FieldError{{
						FieldPath: "Debugmode",
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
	fmt.Printf("Debugmode: %t\n", cfg.Debugmode)

	// Output:
	// Environment: dev
	// Debugmode: false
}

// ExampleDumpEffective demonstrates dumping configuration with secret redaction.
func ExampleDumpEffective() {
	type Config struct {
		APIKey   string `conf:"secret"`
		Endpoint string
		Timeout  time.Duration
	}

	cfg := &Config{
		APIKey:   "super-secret-key",
		Endpoint: "https://api.example.com",
		Timeout:  30 * time.Second,
	}

	// Store minimal provenance for the example
	// (normally this is done automatically during Load)

	// Dump configuration (secrets will be redacted)
	rigging.DumpEffective(os.Stdout, cfg)

	// Output will show:
	// aPIKey: "***redacted***"
	// endpoint: "https://api.example.com"
	// timeout: 30s
}

// ExampleDumpEffective_withSources demonstrates dumping with source attribution.
func ExampleDumpEffective_withSources() {
	type Config struct {
		Port int    `conf:"default:8080"`
		Host string `conf:"default:localhost"`
	}

	cfg := &Config{
		Port: 8080,
		Host: "localhost",
	}

	// Dump with source information
	rigging.DumpEffective(os.Stdout, cfg, rigging.WithSources())

	// Output will include source attribution
}

// ExampleDumpEffective_asJSON demonstrates JSON output format.
func ExampleDumpEffective_asJSON() {
	type Config struct {
		Environment string `conf:"default:dev"`
		Port        int    `conf:"default:8080"`
	}

	cfg := &Config{
		Environment: "dev",
		Port:        8080,
	}

	// Dump as JSON
	rigging.DumpEffective(os.Stdout, cfg, rigging.AsJSON())

	// Output:
	// {
	//   "environment": "dev",
	//   "port": 8080
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

	// Output will show which source provided each field
}

// ExampleOptional demonstrates using Optional fields.
func ExampleOptional() {
	type Config struct {
		Timeout    rigging.Optional[time.Duration]
		Maxretries rigging.Optional[int]
	}

	cfg := &Config{}

	// Set Timeout but not Maxretries
	cfg.Timeout = rigging.Optional[time.Duration]{
		Value: 30 * time.Second,
		Set:   true,
	}

	// Check if Timeout was set
	if timeout, ok := cfg.Timeout.Get(); ok {
		fmt.Printf("Timeout is set to: %v\n", timeout)
	}

	// Check if Maxretries was set
	if _, ok := cfg.Maxretries.Get(); !ok {
		fmt.Println("Maxretries was not set")
	}

	// Use OrDefault for fallback values
	maxRetries := cfg.Maxretries.OrDefault(3)
	fmt.Printf("Maxretries (with default): %d\n", maxRetries)

	// Output:
	// Timeout is set to: 30s
	// Maxretries was not set
	// Maxretries (with default): 3
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
