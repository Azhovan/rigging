# Quick Start

## Installation

```bash
# Core library (zero dependencies)
go get github.com/Azhovan/rigging

# File support (YAML/JSON/TOML)
go get github.com/Azhovan/rigging/sourcefile

# Environment variables
go get github.com/Azhovan/rigging/sourceenv
```

## Basic Usage

Create `config.yaml`:
```yaml
database:
  host: localhost
```

Set the required password:
```bash
export APP_DATABASE__PASSWORD=secret
```

```go
package main

import (
    "context"
    "log"

    "github.com/Azhovan/rigging"
    "github.com/Azhovan/rigging/sourcefile"
    "github.com/Azhovan/rigging/sourceenv"
)

type Config struct {
    Server struct {
        Port int    `conf:"default:8080"`
        Host string `conf:"default:0.0.0.0"`
    } `conf:"prefix:server"`

    Database struct {
        Host     string `conf:"required"`
        Port     int    `conf:"default:5432"`
        Password string `conf:"required,secret"`
    } `conf:"prefix:database"`
}

func main() {
    loader := rigging.NewLoader[Config]().
        WithSource(sourcefile.New("config.yaml", sourcefile.Options{})).
        WithSource(sourceenv.New(sourceenv.Options{Prefix: "APP_"}))

    cfg, err := loader.Load(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Use your configuration
    log.Printf("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)
}
```

## Multi-Source Configuration

Sources are processed in order. Later sources override earlier ones.

```go
loader := rigging.NewLoader[Config]().
    WithSource(sourcefile.New("defaults.yaml", sourcefile.Options{})).  // Base configuration
    WithSource(sourcefile.New("config.yaml", sourcefile.Options{})).    // Environment-specific
    WithSource(sourceenv.New(sourceenv.Options{Prefix: "APP_"}))        // Runtime overrides
```

## Validation

Tag-based validation:

```go
type Config struct {
    Port        int    `conf:"required,min:1024,max:65535"`
    Environment string `conf:"required,oneof:prod,staging,dev"`
    Timeout     time.Duration `conf:"default:30s"`
}
```

Custom validation:

```go
loader.WithValidator(rigging.ValidatorFunc[Config](func(ctx context.Context, cfg *Config) error {
    if cfg.Environment == "prod" && cfg.Server.Host == "localhost" {
        return errors.New("production cannot use localhost")
    }
    return nil
}))
```

## Observability

Track configuration sources:

```go
cfg, err := loader.Load(ctx)
if err != nil {
    log.Fatal(err)
}

prov, _ := rigging.GetProvenance(cfg)
for _, field := range prov.Fields {
    log.Printf("%s from %s", field.FieldPath, field.SourceName)
}
```

Dump configuration safely:

```go
import "os"

// Secrets are automatically redacted
rigging.DumpEffective(os.Stdout, cfg, rigging.WithSources())

// Output:
// server.host: "0.0.0.0" (source: file:config.yaml)
// server.port: 8080 (source: default)
// database.host: "localhost" (source: file:config.yaml)
// database.port: 5432 (source: default)
// database.password: "***redacted***" (source: env:APP_DATABASE__PASSWORD)
```

## Optional Fields

Distinguish "not set" from "zero value":

```go
type Config struct {
    Timeout rigging.Optional[time.Duration]
}

cfg, _ := loader.Load(ctx)

if timeout, ok := cfg.Timeout.Get(); ok {
    // Value was explicitly set
    client.SetTimeout(timeout)
} else {
    // Value not set, use computed default
    client.SetTimeout(computeDefault())
}
```
