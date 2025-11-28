package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	rigging "github.com/Azhovan/rigging"
	"github.com/Azhovan/rigging/sourceenv"
	"github.com/Azhovan/rigging/sourcefile"
)

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host           string        `conf:"required"`
	Port           int           `conf:"default:5432,min:1024,max:65535"`
	Name           string        `conf:"required"`
	User           string        `conf:"required"`
	Password       string        `conf:"secret,required"`
	Maxconnections int           `conf:"default:10,min:1,max:100"`
	Sslmode        string        `conf:"default:disable,oneof:disable,require,verify-ca,verify-full"`
	Connecttimeout time.Duration `conf:"default:5s"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host            string        `conf:"default:localhost"`
	Port            int           `conf:"default:8080,min:1024,max:65535"`
	Readtimeout     time.Duration `conf:"default:15s"`
	Writetimeout    time.Duration `conf:"default:15s"`
	Shutdowntimeout time.Duration `conf:"default:5s"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string `conf:"default:info,oneof:debug,info,warn,error"`
	Format string `conf:"default:text,oneof:text,json"`
	Output string `conf:"default:stdout"`
}

// FeaturesConfig holds feature flags
type FeaturesConfig struct {
	Enablemetrics rigging.Optional[bool]
	Enabletracing rigging.Optional[bool]
	Ratelimit     rigging.Optional[int] `conf:"min:1"`
}

// AppConfig is the root configuration structure
type AppConfig struct {
	Environment string         `conf:"default:development,oneof:development,staging,production"`
	Database    DatabaseConfig `conf:"prefix:database"`
	Server      ServerConfig   `conf:"prefix:server"`
	Logging     LoggingConfig  `conf:"prefix:logging"`
	Features    FeaturesConfig `conf:"prefix:features"`
}

// customValidator demonstrates cross-field validation
func customValidator(ctx context.Context, cfg *AppConfig) error {
	var fieldErrors []rigging.FieldError

	// Production environment must use secure database connection
	if cfg.Environment == "production" {
		if cfg.Database.Host == "localhost" || cfg.Database.Host == "127.0.0.1" {
			fieldErrors = append(fieldErrors, rigging.FieldError{
				FieldPath: "Database.Host",
				Code:      "invalid_prod_host",
				Message:   "production environment cannot use localhost database",
			})
		}

		if cfg.Database.Sslmode == "disable" {
			fieldErrors = append(fieldErrors, rigging.FieldError{
				FieldPath: "Database.Sslmode",
				Code:      "insecure_prod_ssl",
				Message:   "production environment must use SSL for database connections",
			})
		}
	}

	// Server port should not conflict with common services
	if cfg.Server.Port == 5432 || cfg.Server.Port == 3306 {
		fieldErrors = append(fieldErrors, rigging.FieldError{
			FieldPath: "Server.Port",
			Code:      "port_conflict",
			Message:   fmt.Sprintf("server port %d conflicts with common database ports", cfg.Server.Port),
		})
	}

	// If metrics are enabled, rate limit should be set
	if metricsEnabled, ok := cfg.Features.Enablemetrics.Get(); ok && metricsEnabled {
		if rateLimit, ok := cfg.Features.Ratelimit.Get(); !ok || rateLimit == 0 {
			fieldErrors = append(fieldErrors, rigging.FieldError{
				FieldPath: "Features.Ratelimit",
				Code:      "missing_rate_limit",
				Message:   "rate_limit must be set when metrics are enabled",
			})
		}
	}

	if len(fieldErrors) > 0 {
		return &rigging.ValidationError{FieldErrors: fieldErrors}
	}

	return nil
}

func main() {
	ctx := context.Background()

	fmt.Println("=== Configuration Library Example ===\n")

	// Create a loader with multiple sources
	// Sources are processed in order: file first, then environment variables
	// Environment variables will override file values
	loader := rigging.NewLoader[AppConfig]().
		WithSource(sourcefile.New("config.yaml", sourcefile.Options{
			Required: false, // Make file optional for demo purposes
		})).
		WithSource(sourceenv.New(sourceenv.Options{
			Prefix: "APP_", // Only read env vars starting with APP_
		})).
		WithValidator(rigging.ValidatorFunc[AppConfig](customValidator)).
		Strict(false) // Allow unknown configuration keys for demo

	fmt.Println("Loading configuration from:")
	fmt.Println("  1. config.yaml (if present)")
	fmt.Println("  2. Environment variables (APP_* prefix)")
	fmt.Println()

	// Load the configuration
	cfg, err := loader.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v\n", err)
	}

	fmt.Println("✓ Configuration loaded successfully!\n")

	// Display the loaded configuration
	fmt.Println("=== Loaded Configuration ===\n")
	fmt.Printf("Environment: %s\n", cfg.Environment)
	fmt.Printf("\nDatabase:\n")
	fmt.Printf("  Host: %s\n", cfg.Database.Host)
	fmt.Printf("  Port: %d\n", cfg.Database.Port)
	fmt.Printf("  Name: %s\n", cfg.Database.Name)
	fmt.Printf("  User: %s\n", cfg.Database.User)
	fmt.Printf("  Password: [REDACTED]\n")
	fmt.Printf("  Max Connections: %d\n", cfg.Database.Maxconnections)
	fmt.Printf("  SSL Mode: %s\n", cfg.Database.Sslmode)
	fmt.Printf("  Connect Timeout: %s\n", cfg.Database.Connecttimeout)

	fmt.Printf("\nServer:\n")
	fmt.Printf("  Host: %s\n", cfg.Server.Host)
	fmt.Printf("  Port: %d\n", cfg.Server.Port)
	fmt.Printf("  Read Timeout: %s\n", cfg.Server.Readtimeout)
	fmt.Printf("  Write Timeout: %s\n", cfg.Server.Writetimeout)
	fmt.Printf("  Shutdown Timeout: %s\n", cfg.Server.Shutdowntimeout)

	fmt.Printf("\nLogging:\n")
	fmt.Printf("  Level: %s\n", cfg.Logging.Level)
	fmt.Printf("  Format: %s\n", cfg.Logging.Format)
	fmt.Printf("  Output: %s\n", cfg.Logging.Output)

	fmt.Printf("\nFeatures:\n")
	if metrics, ok := cfg.Features.Enablemetrics.Get(); ok {
		fmt.Printf("  Enable Metrics: %v\n", metrics)
	} else {
		fmt.Printf("  Enable Metrics: [not set]\n")
	}
	if tracing, ok := cfg.Features.Enabletracing.Get(); ok {
		fmt.Printf("  Enable Tracing: %v\n", tracing)
	} else {
		fmt.Printf("  Enable Tracing: [not set]\n")
	}
	if rateLimit, ok := cfg.Features.Ratelimit.Get(); ok {
		fmt.Printf("  Rate Limit: %d\n", rateLimit)
	} else {
		fmt.Printf("  Rate Limit: [not set]\n")
	}

	// Demonstrate provenance tracking
	fmt.Println("\n=== Configuration Provenance ===\n")
	if prov, ok := rigging.GetProvenance(cfg); ok {
		fmt.Println("Source information for each field:")
		for _, field := range prov.Fields {
			secretMarker := ""
			if field.Secret {
				secretMarker = " [SECRET]"
			}
			fmt.Printf("  %s = %s (from %s)%s\n",
				field.FieldPath,
				field.KeyPath,
				field.SourceName,
				secretMarker,
			)
		}
	} else {
		fmt.Println("Provenance information not available")
	}

	// Demonstrate DumpEffective with source attribution
	fmt.Println("\n=== Effective Configuration Dump ===\n")
	fmt.Println("Text format with source attribution:")
	fmt.Println("---")
	if err := rigging.DumpEffective(os.Stdout, cfg, rigging.WithSources()); err != nil {
		log.Printf("Failed to dump configuration: %v\n", err)
	}

	fmt.Println("\n---")
	fmt.Println("\nJSON format (secrets redacted):")
	fmt.Println("---")
	if err := rigging.DumpEffective(os.Stdout, cfg, rigging.AsJSON()); err != nil {
		log.Printf("Failed to dump configuration as JSON: %v\n", err)
	}
	fmt.Println("\n---")

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("\nTry setting environment variables to override configuration:")
	fmt.Println("  export APP_ENVIRONMENT=production")
	fmt.Println("  export APP_DATABASE__PASSWORD=secret123")
	fmt.Println("  export APP_SERVER__PORT=9090")
	fmt.Println("  export APP_FEATURES__ENABLE_METRICS=true")
	fmt.Println("\nThen run the example again to see the overrides in action!")
}
