package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Azhovan/rigging"
)

func printLoadPlan() {
	fmt.Println("Loading configuration from:")
	fmt.Println("  1. config.yaml (if present)")
	fmt.Println("  2. Environment variables (APP_* prefix)")
	fmt.Println()
}

func printLoadedConfig(cfg *AppConfig) {
	fmt.Println("=== Loaded Configuration ===")
	fmt.Println()
	fmt.Printf("Environment: %s\n", cfg.Environment)
	fmt.Printf("\nDatabase:\n")
	fmt.Printf("  Host: %s\n", cfg.Database.Host)
	fmt.Printf("  Port: %d\n", cfg.Database.Port)
	fmt.Printf("  Name: %s\n", cfg.Database.Name)
	fmt.Printf("  User: %s\n", cfg.Database.User)
	fmt.Printf("  Password: [REDACTED]\n")
	fmt.Printf("  Max Connections: %d\n", cfg.Database.MaxConnections)
	fmt.Printf("  SSL Mode: %s\n", cfg.Database.SSLMode)
	fmt.Printf("  Connect Timeout: %s\n", cfg.Database.ConnectTimeout)

	fmt.Printf("\nServer:\n")
	fmt.Printf("  Host: %s\n", cfg.Server.Host)
	fmt.Printf("  Port: %d\n", cfg.Server.Port)
	fmt.Printf("  Read Timeout: %s\n", cfg.Server.ReadTimeout)
	fmt.Printf("  Write Timeout: %s\n", cfg.Server.WriteTimeout)
	fmt.Printf("  Shutdown Timeout: %s\n", cfg.Server.ShutdownTimeout)

	fmt.Printf("\nLogging:\n")
	fmt.Printf("  Level: %s\n", cfg.Logging.Level)
	fmt.Printf("  Format: %s\n", cfg.Logging.Format)
	fmt.Printf("  Output: %s\n", cfg.Logging.Output)

	fmt.Printf("\nFeatures:\n")
	if metrics, ok := cfg.Features.EnableMetrics.Get(); ok {
		fmt.Printf("  Enable Metrics: %v\n", metrics)
	} else {
		fmt.Printf("  Enable Metrics: [not set]\n")
	}
	if tracing, ok := cfg.Features.EnableTracing.Get(); ok {
		fmt.Printf("  Enable Tracing: %v\n", tracing)
	} else {
		fmt.Printf("  Enable Tracing: [not set]\n")
	}
	if rateLimit, ok := cfg.Features.RateLimit.Get(); ok {
		fmt.Printf("  Rate Limit: %d\n", rateLimit)
	} else {
		fmt.Printf("  Rate Limit: [not set]\n")
	}
}

func printProvenance(cfg *AppConfig) {
	fmt.Println()
	fmt.Println("=== Configuration Provenance ===")
	fmt.Println()
	fmt.Println("Track exactly where each configuration value came from:")
	fmt.Println()
	if prov, ok := rigging.GetProvenance(cfg); ok {
		fmt.Println("Field -> Source")
		fmt.Println("----------------------------------------")
		for _, field := range prov.Fields {
			secretMarker := ""
			if field.Secret {
				secretMarker = " [SECRET]"
			}
			fmt.Printf("%-30s -> %s%s\n", field.FieldPath, field.SourceName, secretMarker)
		}
		fmt.Println()
		fmt.Println("Environment variables are shown with their full names (e.g., env:APP_DATABASE__PASSWORD)")
		fmt.Println("This helps verify which env var was actually used.")
		return
	}

	fmt.Println("Provenance information not available")
}

func printEffectiveDumps(cfg *AppConfig) {
	fmt.Println()
	fmt.Println("=== Effective Configuration Dump ===")
	fmt.Println()
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
}

func printNextSteps() {
	fmt.Println("\n=== Example Complete ===")
	fmt.Println("\nTry setting environment variables to override configuration:")
	fmt.Println("  export APP_ENVIRONMENT=production")
	fmt.Println("  export APP_DATABASE__PASSWORD=secret123")
	fmt.Println("  export APP_SERVER__PORT=9090")
	fmt.Println("  export APP_FEATURES__ENABLE_METRICS=true")
	fmt.Println("\nThen run the example again to see the overrides in action!")
}
