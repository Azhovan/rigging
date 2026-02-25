package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Azhovan/rigging"
	"github.com/Azhovan/rigging/sourceenv"
)

type AppConfig struct {
	Environment string `conf:"required,oneof:dev,staging,prod"`
	Region      string `conf:"default:us-east-1,oneof:us-east-1,us-west-2,eu-west-1"`
}

func main() {
	ctx := context.Background()

	// Seed demo input so the example runs without external setup.
	os.Setenv("EXTRANS_ENVIRONMENT", " PROD ")
	defer os.Unsetenv("EXTRANS_ENVIRONMENT")

	loader := rigging.NewLoader[AppConfig]().
		WithSource(sourceenv.New(sourceenv.Options{Prefix: "EXTRANS_"})).
		WithTransformerFunc(func(ctx context.Context, cfg *AppConfig) error {
			// Typed transform: canonicalize values after binding/conversion, before validation.
			cfg.Environment = strings.ToLower(strings.TrimSpace(cfg.Environment))
			cfg.Region = strings.ToLower(strings.TrimSpace(cfg.Region))
			return nil
		})

	cfg, err := loader.Load(ctx)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	fmt.Println("Transformer example loaded successfully")
	fmt.Printf("Environment: %s\n", cfg.Environment)
	fmt.Printf("Region: %s\n", cfg.Region)
	fmt.Println()
	fmt.Println("Note: Transformers mutate typed config values, not source keys.")
	fmt.Println("For key aliasing/normalization, use a custom Source wrapper instead.")
}
