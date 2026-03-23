package rigging

import (
	"context"
	"testing"
	"time"
)

func BenchmarkLoad_WarmSchemaValidation(b *testing.B) {
	type TLSConfig struct {
		Enabled bool
		CertPEM string
		KeyPEM  string `conf:"secret"`
	}

	type ServerConfig struct {
		Host string `conf:"required"`
		Port int    `conf:"default:8080,min:1024,max:65535"`
		TLS  TLSConfig
	}

	type DatabaseConfig struct {
		Host            string        `conf:"required"`
		Port            int           `conf:"default:5432,min:1"`
		MaxOpenConns    int           `conf:"default:10,min:1"`
		ConnectTimeout  time.Duration `conf:"default:5s,min:1"`
		ConnectionAlias Optional[string]
	}

	type Config struct {
		Server   ServerConfig   `conf:"prefix:server"`
		Database DatabaseConfig `conf:"prefix:database"`
		Env      string         `conf:"default:dev,oneof:dev,staging,prod"`
	}

	source := &mockSource{
		data: map[string]any{
			"server.host":               "127.0.0.1",
			"server.tls.enabled":        true,
			"server.tls.cert_pem":       "cert",
			"server.tls.key_pem":        "key",
			"database.host":             "db.internal",
			"database.max_open_conns":   32,
			"database.connect_timeout":  "10s",
			"database.connection_alias": "primary",
			"env":                       "prod",
		},
	}

	loader := NewLoader[Config]().WithSource(source)

	if _, err := loader.Load(context.Background()); err != nil {
		b.Fatalf("warm-up load failed: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cfg, err := loader.Load(context.Background())
		if err != nil {
			b.Fatalf("Load failed: %v", err)
		}
		if cfg == nil {
			b.Fatal("Load returned nil config")
		}
	}
}
