package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Azhovan/rigging"
)

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host           string        `conf:"required"`
	Port           int           `conf:"default:5432,min:1024,max:65535"`
	Name           string        `conf:"required"`
	User           string        `conf:"required"`
	Password       string        `conf:"secret,required"`
	MaxConnections int           `conf:"default:10,min:1,max:100"`
	SSLMode        string        `conf:"default:disable,oneof:disable,require,verify-ca,verify-full"`
	ConnectTimeout time.Duration `conf:"default:5s"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host            string        `conf:"default:localhost"`
	Port            int           `conf:"default:8080,min:1024,max:65535"`
	ReadTimeout     time.Duration `conf:"default:15s"`
	WriteTimeout    time.Duration `conf:"default:15s"`
	ShutdownTimeout time.Duration `conf:"default:5s"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string `conf:"default:info,oneof:debug,info,warn,error"`
	Format string `conf:"default:text,oneof:text,json"`
	Output string `conf:"default:stdout"`
}

// FeaturesConfig holds feature flags
type FeaturesConfig struct {
	EnableMetrics rigging.Optional[bool]
	EnableTracing rigging.Optional[bool]
	RateLimit     rigging.Optional[int] `conf:"min:1"`
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

		if cfg.Database.SSLMode == "disable" {
			fieldErrors = append(fieldErrors, rigging.FieldError{
				FieldPath: "Database.SSLMode",
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
	if metricsEnabled, ok := cfg.Features.EnableMetrics.Get(); ok && metricsEnabled {
		if rateLimit, ok := cfg.Features.RateLimit.Get(); !ok || rateLimit == 0 {
			fieldErrors = append(fieldErrors, rigging.FieldError{
				FieldPath: "Features.RateLimit",
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
