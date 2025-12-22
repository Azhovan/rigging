package rigging

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Benchmark configurations of various sizes

// Small config (10 fields)
type BenchConfigSmall struct {
	Host     string `conf:"name:host"`
	Port     int    `conf:"name:port"`
	Debug    bool   `conf:"name:debug"`
	Timeout  int    `conf:"name:timeout"`
	MaxConns int    `conf:"name:max_conns"`
	LogLevel string `conf:"name:log_level"`
	APIKey   string `conf:"name:api_key,secret"`
	Region   string `conf:"name:region"`
	Env      string `conf:"name:env"`
	Version  string `conf:"name:version"`
}

// Medium config (100 fields) - using nested structs
type BenchConfigMedium struct {
	App      BenchAppConfig      `conf:"prefix:app"`
	Database BenchDatabaseConfig `conf:"prefix:database"`
	Cache    BenchCacheConfig    `conf:"prefix:cache"`
	Auth     BenchAuthConfig     `conf:"prefix:auth"`
	Logging  BenchLoggingConfig  `conf:"prefix:logging"`
	Metrics  BenchMetricsConfig  `conf:"prefix:metrics"`
	Server   BenchServerConfig   `conf:"prefix:server"`
	Features BenchFeaturesConfig `conf:"prefix:features"`
	Limits   BenchLimitsConfig   `conf:"prefix:limits"`
	External BenchExternalConfig `conf:"prefix:external"`
}

type BenchAppConfig struct {
	Name        string `conf:"name:name"`
	Version     string `conf:"name:version"`
	Environment string `conf:"name:environment"`
	Debug       bool   `conf:"name:debug"`
	LogLevel    string `conf:"name:log_level"`
	Timezone    string `conf:"name:timezone"`
	Locale      string `conf:"name:locale"`
	BaseURL     string `conf:"name:base_url"`
	AdminEmail  string `conf:"name:admin_email"`
	SupportURL  string `conf:"name:support_url"`
}

type BenchDatabaseConfig struct {
	Host         string `conf:"name:host"`
	Port         int    `conf:"name:port"`
	Name         string `conf:"name:name"`
	User         string `conf:"name:user"`
	Password     string `conf:"name:password,secret"`
	SSLMode      string `conf:"name:ssl_mode"`
	MaxOpenConns int    `conf:"name:max_open_conns"`
	MaxIdleConns int    `conf:"name:max_idle_conns"`
	ConnTimeout  int    `conf:"name:conn_timeout"`
	QueryTimeout int    `conf:"name:query_timeout"`
}

type BenchCacheConfig struct {
	Host       string `conf:"name:host"`
	Port       int    `conf:"name:port"`
	Password   string `conf:"name:password,secret"`
	DB         int    `conf:"name:db"`
	MaxRetries int    `conf:"name:max_retries"`
	PoolSize   int    `conf:"name:pool_size"`
	TTL        int    `conf:"name:ttl"`
	Prefix     string `conf:"name:prefix"`
	Enabled    bool   `conf:"name:enabled"`
	Cluster    bool   `conf:"name:cluster"`
}

type BenchAuthConfig struct {
	JWTSecret     string `conf:"name:jwt_secret,secret"`
	JWTExpiry     int    `conf:"name:jwt_expiry"`
	RefreshExpiry int    `conf:"name:refresh_expiry"`
	BCryptCost    int    `conf:"name:bcrypt_cost"`
	OAuthClientID string `conf:"name:oauth_client_id"`
	OAuthSecret   string `conf:"name:oauth_secret,secret"`
	SessionTTL    int    `conf:"name:session_ttl"`
	MaxSessions   int    `conf:"name:max_sessions"`
	MFAEnabled    bool   `conf:"name:mfa_enabled"`
	MFAIssuer     string `conf:"name:mfa_issuer"`
}

type BenchLoggingConfig struct {
	Level      string `conf:"name:level"`
	Format     string `conf:"name:format"`
	Output     string `conf:"name:output"`
	FilePath   string `conf:"name:file_path"`
	MaxSize    int    `conf:"name:max_size"`
	MaxBackups int    `conf:"name:max_backups"`
	MaxAge     int    `conf:"name:max_age"`
	Compress   bool   `conf:"name:compress"`
	JSON       bool   `conf:"name:json"`
	Caller     bool   `conf:"name:caller"`
}

type BenchMetricsConfig struct {
	Enabled    bool   `conf:"name:enabled"`
	Port       int    `conf:"name:port"`
	Path       string `conf:"name:path"`
	Namespace  string `conf:"name:namespace"`
	Subsystem  string `conf:"name:subsystem"`
	Buckets    string `conf:"name:buckets"`
	Objectives string `conf:"name:objectives"`
	MaxAge     int    `conf:"name:max_age"`
	AgeBuckets int    `conf:"name:age_buckets"`
	BufCap     int    `conf:"name:buf_cap"`
}

type BenchServerConfig struct {
	Host           string `conf:"name:host"`
	Port           int    `conf:"name:port"`
	ReadTimeout    int    `conf:"name:read_timeout"`
	WriteTimeout   int    `conf:"name:write_timeout"`
	IdleTimeout    int    `conf:"name:idle_timeout"`
	MaxHeaderBytes int    `conf:"name:max_header_bytes"`
	TLSEnabled     bool   `conf:"name:tls_enabled"`
	TLSCert        string `conf:"name:tls_cert"`
	TLSKey         string `conf:"name:tls_key,secret"`
	GracefulStop   int    `conf:"name:graceful_stop"`
}

type BenchFeaturesConfig struct {
	FeatureA bool `conf:"name:feature_a"`
	FeatureB bool `conf:"name:feature_b"`
	FeatureC bool `conf:"name:feature_c"`
	FeatureD bool `conf:"name:feature_d"`
	FeatureE bool `conf:"name:feature_e"`
	FeatureF bool `conf:"name:feature_f"`
	FeatureG bool `conf:"name:feature_g"`
	FeatureH bool `conf:"name:feature_h"`
	FeatureI bool `conf:"name:feature_i"`
	FeatureJ bool `conf:"name:feature_j"`
}

type BenchLimitsConfig struct {
	MaxRequests    int `conf:"name:max_requests"`
	MaxConnections int `conf:"name:max_connections"`
	MaxBodySize    int `conf:"name:max_body_size"`
	MaxUploadSize  int `conf:"name:max_upload_size"`
	RateLimit      int `conf:"name:rate_limit"`
	BurstLimit     int `conf:"name:burst_limit"`
	Timeout        int `conf:"name:timeout"`
	RetryLimit     int `conf:"name:retry_limit"`
	QueueSize      int `conf:"name:queue_size"`
	WorkerCount    int `conf:"name:worker_count"`
}

type BenchExternalConfig struct {
	APIURL     string `conf:"name:api_url"`
	APIKey     string `conf:"name:api_key,secret"`
	APISecret  string `conf:"name:api_secret,secret"`
	WebhookURL string `conf:"name:webhook_url"`
	Timeout    int    `conf:"name:timeout"`
	Retries    int    `conf:"name:retries"`
	RateLimit  int    `conf:"name:rate_limit"`
	BatchSize  int    `conf:"name:batch_size"`
	Enabled    bool   `conf:"name:enabled"`
	Debug      bool   `conf:"name:debug"`
}

// Large config (1000 fields) - using deeply nested structs with many sections
type BenchConfigLarge struct {
	Section1  BenchLargeSection `conf:"prefix:section1"`
	Section2  BenchLargeSection `conf:"prefix:section2"`
	Section3  BenchLargeSection `conf:"prefix:section3"`
	Section4  BenchLargeSection `conf:"prefix:section4"`
	Section5  BenchLargeSection `conf:"prefix:section5"`
	Section6  BenchLargeSection `conf:"prefix:section6"`
	Section7  BenchLargeSection `conf:"prefix:section7"`
	Section8  BenchLargeSection `conf:"prefix:section8"`
	Section9  BenchLargeSection `conf:"prefix:section9"`
	Section10 BenchLargeSection `conf:"prefix:section10"`
}

type BenchLargeSection struct {
	SubA BenchLargeSubSection `conf:"prefix:sub_a"`
	SubB BenchLargeSubSection `conf:"prefix:sub_b"`
	SubC BenchLargeSubSection `conf:"prefix:sub_c"`
	SubD BenchLargeSubSection `conf:"prefix:sub_d"`
	SubE BenchLargeSubSection `conf:"prefix:sub_e"`
}

type BenchLargeSubSection struct {
	Field1  string `conf:"name:field1"`
	Field2  string `conf:"name:field2"`
	Field3  string `conf:"name:field3"`
	Field4  string `conf:"name:field4"`
	Field5  string `conf:"name:field5"`
	Field6  int    `conf:"name:field6"`
	Field7  int    `conf:"name:field7"`
	Field8  int    `conf:"name:field8"`
	Field9  int    `conf:"name:field9"`
	Field10 int    `conf:"name:field10"`
	Field11 bool   `conf:"name:field11"`
	Field12 bool   `conf:"name:field12"`
	Field13 bool   `conf:"name:field13"`
	Field14 bool   `conf:"name:field14"`
	Field15 bool   `conf:"name:field15"`
	Field16 string `conf:"name:field16,secret"`
	Field17 string `conf:"name:field17"`
	Field18 string `conf:"name:field18"`
	Field19 string `conf:"name:field19"`
	Field20 string `conf:"name:field20"`
}

// Helper functions to create populated configs

func newBenchConfigSmall() *BenchConfigSmall {
	return &BenchConfigSmall{
		Host:     "localhost",
		Port:     8080,
		Debug:    true,
		Timeout:  30,
		MaxConns: 100,
		LogLevel: "info",
		APIKey:   "secret-api-key-12345",
		Region:   "us-east-1",
		Env:      "production",
		Version:  "1.0.0",
	}
}

func newBenchConfigMedium() *BenchConfigMedium {
	return &BenchConfigMedium{
		App: BenchAppConfig{
			Name: "myapp", Version: "2.0.0", Environment: "prod", Debug: false,
			LogLevel: "warn", Timezone: "UTC", Locale: "en-US",
			BaseURL: "https://api.example.com", AdminEmail: "admin@example.com",
			SupportURL: "https://support.example.com",
		},
		Database: BenchDatabaseConfig{
			Host: "db.example.com", Port: 5432, Name: "mydb", User: "dbuser",
			Password: "dbpass123", SSLMode: "require", MaxOpenConns: 25,
			MaxIdleConns: 5, ConnTimeout: 10, QueryTimeout: 30,
		},
		Cache: BenchCacheConfig{
			Host: "redis.example.com", Port: 6379, Password: "redispass",
			DB: 0, MaxRetries: 3, PoolSize: 10, TTL: 3600,
			Prefix: "myapp:", Enabled: true, Cluster: false,
		},
		Auth: BenchAuthConfig{
			JWTSecret: "jwt-secret-key", JWTExpiry: 3600, RefreshExpiry: 86400,
			BCryptCost: 12, OAuthClientID: "oauth-client", OAuthSecret: "oauth-secret",
			SessionTTL: 7200, MaxSessions: 5, MFAEnabled: true, MFAIssuer: "MyApp",
		},
		Logging: BenchLoggingConfig{
			Level: "info", Format: "json", Output: "stdout", FilePath: "/var/log/app.log",
			MaxSize: 100, MaxBackups: 3, MaxAge: 28, Compress: true, JSON: true, Caller: true,
		},
		Metrics: BenchMetricsConfig{
			Enabled: true, Port: 9090, Path: "/metrics", Namespace: "myapp",
			Subsystem: "http", Buckets: "0.1,0.5,1,5", Objectives: "0.5,0.9,0.99",
			MaxAge: 600, AgeBuckets: 5, BufCap: 500,
		},
		Server: BenchServerConfig{
			Host: "0.0.0.0", Port: 8080, ReadTimeout: 30, WriteTimeout: 30,
			IdleTimeout: 120, MaxHeaderBytes: 1048576, TLSEnabled: true,
			TLSCert: "/etc/ssl/cert.pem", TLSKey: "tls-private-key", GracefulStop: 30,
		},
		Features: BenchFeaturesConfig{
			FeatureA: true, FeatureB: false, FeatureC: true, FeatureD: true,
			FeatureE: false, FeatureF: true, FeatureG: false, FeatureH: true,
			FeatureI: true, FeatureJ: false,
		},
		Limits: BenchLimitsConfig{
			MaxRequests: 1000, MaxConnections: 500, MaxBodySize: 10485760,
			MaxUploadSize: 52428800, RateLimit: 100, BurstLimit: 200,
			Timeout: 60, RetryLimit: 3, QueueSize: 1000, WorkerCount: 10,
		},
		External: BenchExternalConfig{
			APIURL: "https://external.api.com", APIKey: "ext-api-key",
			APISecret: "ext-api-secret", WebhookURL: "https://webhook.example.com",
			Timeout: 30, Retries: 3, RateLimit: 50, BatchSize: 100,
			Enabled: true, Debug: false,
		},
	}
}

func newBenchConfigLarge() *BenchConfigLarge {
	section := func() BenchLargeSection {
		sub := func() BenchLargeSubSection {
			return BenchLargeSubSection{
				Field1: "value1", Field2: "value2", Field3: "value3",
				Field4: "value4", Field5: "value5", Field6: 100, Field7: 200,
				Field8: 300, Field9: 400, Field10: 500, Field11: true,
				Field12: false, Field13: true, Field14: false, Field15: true,
				Field16: "secret-value", Field17: "value17", Field18: "value18",
				Field19: "value19", Field20: "value20",
			}
		}
		return BenchLargeSection{
			SubA: sub(), SubB: sub(), SubC: sub(), SubD: sub(), SubE: sub(),
		}
	}
	return &BenchConfigLarge{
		Section1: section(), Section2: section(), Section3: section(),
		Section4: section(), Section5: section(), Section6: section(),
		Section7: section(), Section8: section(), Section9: section(),
		Section10: section(),
	}
}

func setupBenchProvenance[T any](cfg *T, fields []FieldProvenance) {
	prov := &Provenance{Fields: fields}
	storeProvenance(cfg, prov)
}

// Benchmarks for CreateSnapshot with various config sizes

func BenchmarkCreateSnapshot_SmallConfig(b *testing.B) {
	cfg := newBenchConfigSmall()
	setupBenchProvenance(cfg, []FieldProvenance{
		{FieldPath: "APIKey", KeyPath: "api_key", SourceName: "env", Secret: true},
	})
	defer deleteProvenance(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CreateSnapshot(cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCreateSnapshot_MediumConfig(b *testing.B) {
	cfg := newBenchConfigMedium()
	setupBenchProvenance(cfg, []FieldProvenance{
		{FieldPath: "Database.Password", KeyPath: "database.password", SourceName: "env", Secret: true},
		{FieldPath: "Cache.Password", KeyPath: "cache.password", SourceName: "env", Secret: true},
		{FieldPath: "Auth.JWTSecret", KeyPath: "auth.jwt_secret", SourceName: "env", Secret: true},
		{FieldPath: "Auth.OAuthSecret", KeyPath: "auth.oauth_secret", SourceName: "env", Secret: true},
		{FieldPath: "Server.TLSKey", KeyPath: "server.tls_key", SourceName: "env", Secret: true},
		{FieldPath: "External.APIKey", KeyPath: "external.api_key", SourceName: "env", Secret: true},
		{FieldPath: "External.APISecret", KeyPath: "external.api_secret", SourceName: "env", Secret: true},
	})
	defer deleteProvenance(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CreateSnapshot(cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCreateSnapshot_LargeConfig(b *testing.B) {
	cfg := newBenchConfigLarge()
	// Add provenance for some secret fields
	var provFields []FieldProvenance
	for i := 1; i <= 10; i++ {
		for _, sub := range []string{"sub_a", "sub_b", "sub_c", "sub_d", "sub_e"} {
			provFields = append(provFields, FieldProvenance{
				FieldPath:  "Section" + string(rune('0'+i)) + "." + sub + ".Field16",
				KeyPath:    "section" + string(rune('0'+i)) + "." + sub + ".field16",
				SourceName: "env",
				Secret:     true,
			})
		}
	}
	setupBenchProvenance(cfg, provFields)
	defer deleteProvenance(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CreateSnapshot(cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCreateSnapshot_WithExclusions(b *testing.B) {
	cfg := newBenchConfigMedium()
	exclusions := []string{
		"database.password", "cache.password", "auth.jwt_secret",
		"auth.oauth_secret", "server.tls_key", "external.api_key",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CreateSnapshot(cfg, WithExcludeFields(exclusions...))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for WriteSnapshot

func BenchmarkWriteSnapshot_SmallConfig(b *testing.B) {
	cfg := newBenchConfigSmall()
	snapshot, err := CreateSnapshot(cfg)
	if err != nil {
		b.Fatal(err)
	}

	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tmpDir, "snapshot_small_"+string(rune('a'+i%26))+".json")
		if err := WriteSnapshot(snapshot, path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteSnapshot_MediumConfig(b *testing.B) {
	cfg := newBenchConfigMedium()
	snapshot, err := CreateSnapshot(cfg)
	if err != nil {
		b.Fatal(err)
	}

	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tmpDir, "snapshot_medium_"+string(rune('a'+i%26))+".json")
		if err := WriteSnapshot(snapshot, path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteSnapshot_LargeConfig(b *testing.B) {
	cfg := newBenchConfigLarge()
	snapshot, err := CreateSnapshot(cfg)
	if err != nil {
		b.Fatal(err)
	}

	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tmpDir, "snapshot_large_"+string(rune('a'+i%26))+".json")
		if err := WriteSnapshot(snapshot, path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteSnapshot_WithTemplateExpansion(b *testing.B) {
	cfg := newBenchConfigMedium()
	snapshot, err := CreateSnapshot(cfg)
	if err != nil {
		b.Fatal(err)
	}

	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use template path - timestamp will be expanded
		path := filepath.Join(tmpDir, "config-{{timestamp}}-"+string(rune('a'+i%26))+".json")
		if err := WriteSnapshot(snapshot, path); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for ExpandPath

func BenchmarkExpandPath_NoVariables(b *testing.B) {
	template := "/var/log/app/config/snapshot.json"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExpandPath(template)
	}
}

func BenchmarkExpandPath_SingleTimestamp(b *testing.B) {
	template := "/var/log/app/config-{{timestamp}}.json"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExpandPath(template)
	}
}

func BenchmarkExpandPath_MultipleTimestamps(b *testing.B) {
	template := "/var/log/{{timestamp}}/app/config-{{timestamp}}-backup-{{timestamp}}.json"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExpandPath(template)
	}
}

func BenchmarkExpandPathWithTime_NoVariables(b *testing.B) {
	template := "/var/log/app/config/snapshot.json"
	testTime := time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExpandPathWithTime(template, testTime)
	}
}

func BenchmarkExpandPathWithTime_SingleTimestamp(b *testing.B) {
	template := "/var/log/app/config-{{timestamp}}.json"
	testTime := time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExpandPathWithTime(template, testTime)
	}
}

func BenchmarkExpandPathWithTime_MultipleTimestamps(b *testing.B) {
	template := "/var/log/{{timestamp}}/app/config-{{timestamp}}-backup-{{timestamp}}.json"
	testTime := time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExpandPathWithTime(template, testTime)
	}
}

// Benchmarks for ReadSnapshot

func BenchmarkReadSnapshot_SmallConfig(b *testing.B) {
	cfg := newBenchConfigSmall()
	snapshot, err := CreateSnapshot(cfg)
	if err != nil {
		b.Fatal(err)
	}

	tmpDir := b.TempDir()
	path := filepath.Join(tmpDir, "snapshot_small.json")
	if err := WriteSnapshot(snapshot, path); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ReadSnapshot(path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadSnapshot_MediumConfig(b *testing.B) {
	cfg := newBenchConfigMedium()
	snapshot, err := CreateSnapshot(cfg)
	if err != nil {
		b.Fatal(err)
	}

	tmpDir := b.TempDir()
	path := filepath.Join(tmpDir, "snapshot_medium.json")
	if err := WriteSnapshot(snapshot, path); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ReadSnapshot(path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadSnapshot_LargeConfig(b *testing.B) {
	cfg := newBenchConfigLarge()
	snapshot, err := CreateSnapshot(cfg)
	if err != nil {
		b.Fatal(err)
	}

	tmpDir := b.TempDir()
	path := filepath.Join(tmpDir, "snapshot_large.json")
	if err := WriteSnapshot(snapshot, path); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ReadSnapshot(path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark for complete round-trip

func BenchmarkRoundTrip_SmallConfig(b *testing.B) {
	cfg := newBenchConfigSmall()
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snapshot, err := CreateSnapshot(cfg)
		if err != nil {
			b.Fatal(err)
		}

		path := filepath.Join(tmpDir, "roundtrip_small.json")
		if writeErr := WriteSnapshot(snapshot, path); writeErr != nil {
			b.Fatal(writeErr)
		}

		_, err = ReadSnapshot(path)
		if err != nil {
			b.Fatal(err)
		}

		os.Remove(path)
	}
}

func BenchmarkRoundTrip_MediumConfig(b *testing.B) {
	cfg := newBenchConfigMedium()
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snapshot, err := CreateSnapshot(cfg)
		if err != nil {
			b.Fatal(err)
		}

		path := filepath.Join(tmpDir, "roundtrip_medium.json")
		if writeErr := WriteSnapshot(snapshot, path); writeErr != nil {
			b.Fatal(writeErr)
		}

		_, err = ReadSnapshot(path)
		if err != nil {
			b.Fatal(err)
		}

		os.Remove(path)
	}
}
