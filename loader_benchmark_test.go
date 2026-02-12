package rigging

import (
	"context"
	"fmt"
	"testing"
)

type benchDynamicNode struct {
	Host string
	Port int
}

type benchDynamicConfig struct {
	Clickhouse map[string]benchDynamicNode
}

type benchMultiDynamicConfig struct {
	Clickhouse map[string]benchDynamicNode
	Kafka      map[string]benchDynamicNode
	Redis      map[string]benchDynamicNode
}

func BenchmarkLoad_StrictMode_DynamicMapManyKeys(b *testing.B) {
	const entries = 1000

	data := make(map[string]any, entries*2)
	for i := 0; i < entries; i++ {
		name := fmt.Sprintf("node_%04d", i)
		data["clickhouse."+name+".host"] = fmt.Sprintf("ch-%d.internal", i)
		data["clickhouse."+name+".port"] = 9000 + (i % 10)
	}

	loader := NewLoader[benchDynamicConfig]().WithSource(&mockSource{data: data}).Strict(true)
	ctx := context.Background()

	// Warm caches (metadata + dynamic map matcher) outside timing loop.
	if _, err := loader.Load(ctx); err != nil {
		b.Fatalf("warm load failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cfg, err := loader.Load(ctx)
		if err != nil {
			b.Fatalf("load failed: %v", err)
		}
		if len(cfg.Clickhouse) != entries {
			b.Fatalf("unexpected number of entries: got %d, want %d", len(cfg.Clickhouse), entries)
		}
	}
}

func BenchmarkLoad_StrictMode_MultipleDynamicMapsManyKeys(b *testing.B) {
	const entries = 1000

	data := make(map[string]any, entries*6)
	for i := 0; i < entries; i++ {
		name := fmt.Sprintf("node_%04d", i)

		data["clickhouse."+name+".host"] = fmt.Sprintf("ch-%d.internal", i)
		data["clickhouse."+name+".port"] = 9000 + (i % 10)

		data["kafka."+name+".host"] = fmt.Sprintf("kafka-%d.internal", i)
		data["kafka."+name+".port"] = 9092 + (i % 5)

		data["redis."+name+".host"] = fmt.Sprintf("redis-%d.internal", i)
		data["redis."+name+".port"] = 6379 + (i % 3)
	}

	loader := NewLoader[benchMultiDynamicConfig]().WithSource(&mockSource{data: data}).Strict(true)
	ctx := context.Background()

	// Warm caches (metadata + dynamic map matcher) outside timing loop.
	if _, err := loader.Load(ctx); err != nil {
		b.Fatalf("warm load failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cfg, err := loader.Load(ctx)
		if err != nil {
			b.Fatalf("load failed: %v", err)
		}
		if len(cfg.Clickhouse) != entries || len(cfg.Kafka) != entries || len(cfg.Redis) != entries {
			b.Fatalf(
				"unexpected map sizes: clickhouse=%d kafka=%d redis=%d, want %d each",
				len(cfg.Clickhouse), len(cfg.Kafka), len(cfg.Redis), entries,
			)
		}
	}
}
