package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Azhovan/rigging"
	"github.com/Azhovan/rigging/sourceenv"
	"github.com/Azhovan/rigging/sourcefile"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Configuration Library Example ===")
	fmt.Println()

	// Create a loader with multiple sources
	// Sources are processed in order: file first, then environment variables
	// Environment variables will override file values

	// Try to find config.yaml in current directory or examples/basic directory
	configPath := "config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "examples/basic/config.yaml"
	}

	loader := rigging.NewLoader[AppConfig]().
		WithSource(sourcefile.New(configPath, sourcefile.Options{
			Required: false, // Make file optional for demo purposes
		})).
		WithSource(sourceenv.New(sourceenv.Options{
			Prefix: "APP_", // Only read env vars starting with APP_
		})).
		WithValidator(rigging.ValidatorFunc[AppConfig](customValidator)).
		Strict(false) // Allow unknown configuration keys for demo

	printLoadPlan()

	// Load the configuration
	cfg, err := loader.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v\n", err)
	}

	fmt.Println("✓ Configuration loaded successfully!")
	fmt.Println()

	printLoadedConfig(cfg)
	printProvenance(cfg)
	printEffectiveDumps(cfg)
	printNextSteps()
}
