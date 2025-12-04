package sourcefile

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Azhovan/rigging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSource_Load_YAML(t *testing.T) {
	// Create a temporary YAML file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `
database:
  host: localhost
  port: 5432
  credentials:
    user: admin
    password: secret
server:
  address: 0.0.0.0
  timeout: 30
features:
  - feature1
  - feature2
`
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load the file
	src := New(yamlFile, Options{})
	ctx := context.Background()
	data, err := src.Load(ctx)
	require.NoError(t, err)

	// Verify flattened keys
	assert.Equal(t, "localhost", data["database.host"])
	assert.Equal(t, 5432, data["database.port"])
	assert.Equal(t, "admin", data["database.credentials.user"])
	assert.Equal(t, "secret", data["database.credentials.password"])
	assert.Equal(t, "0.0.0.0", data["server.address"])
	assert.Equal(t, 30, data["server.timeout"])

	// Arrays should be preserved
	features, ok := data["features"].([]any)
	require.True(t, ok, "features should be an array")
	assert.Len(t, features, 2)
}

func TestFileSource_Load_JSON(t *testing.T) {
	// Create a temporary JSON file
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "config.json")
	jsonContent := `{
  "database": {
    "host": "db.example.com",
    "port": 3306
  },
  "api": {
    "key": "secret-key",
    "endpoint": "https://api.example.com"
  }
}`
	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	require.NoError(t, err)

	// Load the file
	src := New(jsonFile, Options{})
	ctx := context.Background()
	data, err := src.Load(ctx)
	require.NoError(t, err)

	// Verify flattened keys
	assert.Equal(t, "db.example.com", data["database.host"])
	assert.Equal(t, float64(3306), data["database.port"]) // JSON numbers are float64
	assert.Equal(t, "secret-key", data["api.key"])
	assert.Equal(t, "https://api.example.com", data["api.endpoint"])
}

func TestFileSource_Load_TOML(t *testing.T) {
	// Create a temporary TOML file
	tmpDir := t.TempDir()
	tomlFile := filepath.Join(tmpDir, "config.toml")
	tomlContent := `
[database]
host = "localhost"
port = 5432

[database.pool]
max_connections = 100
min_connections = 10

[server]
address = "127.0.0.1"
`
	err := os.WriteFile(tomlFile, []byte(tomlContent), 0644)
	require.NoError(t, err)

	// Load the file
	src := New(tomlFile, Options{})
	ctx := context.Background()
	data, err := src.Load(ctx)
	require.NoError(t, err)

	// Verify flattened keys
	assert.Equal(t, "localhost", data["database.host"])
	assert.Equal(t, int64(5432), data["database.port"])
	assert.Equal(t, int64(100), data["database.pool.max_connections"])
	assert.Equal(t, int64(10), data["database.pool.min_connections"])
	assert.Equal(t, "127.0.0.1", data["server.address"])
}

func TestFileSource_FormatInference(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		expected map[string]any
	}{
		{
			name:     "yaml extension",
			filename: "config.yaml",
			content:  "key: value",
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "yml extension",
			filename: "config.yml",
			content:  "key: value",
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "json extension",
			filename: "config.json",
			content:  `{"key": "value"}`,
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "toml extension",
			filename: "config.toml",
			content:  `key = "value"`,
			expected: map[string]any{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			require.NoError(t, err)

			src := New(filePath, Options{}) // No explicit format
			ctx := context.Background()
			data, err := src.Load(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, data)
		})
	}
}

func TestFileSource_ExplicitFormat(t *testing.T) {
	// Create a file with wrong extension but specify format explicitly
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "config.txt")
	yamlContent := "key: value"
	err := os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	src := New(filePath, Options{Format: "yaml"})
	ctx := context.Background()
	data, err := src.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, "value", data["key"])
}

func TestFileSource_MissingFile_NotRequired(t *testing.T) {
	// Try to load a non-existent file with Required=false
	src := New("/nonexistent/config.yaml", Options{Required: false})
	ctx := context.Background()
	data, err := src.Load(ctx)
	require.NoError(t, err)
	assert.Empty(t, data, "should return empty map for missing non-required file")
}

func TestFileSource_MissingFile_Required(t *testing.T) {
	// Try to load a non-existent file with Required=true
	src := New("/nonexistent/config.yaml", Options{Required: true})
	ctx := context.Background()
	data, err := src.Load(ctx)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "required config file not found")
}

func TestFileSource_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "invalid.yaml")
	invalidContent := "key: value\n\t\tinvalid: [unclosed"
	err := os.WriteFile(yamlFile, []byte(invalidContent), 0644)
	require.NoError(t, err)

	src := New(yamlFile, Options{})
	ctx := context.Background()
	data, err := src.Load(ctx)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "parse YAML file")
}

func TestFileSource_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "invalid.json")
	invalidContent := `{"key": "value"`
	err := os.WriteFile(jsonFile, []byte(invalidContent), 0644)
	require.NoError(t, err)

	src := New(jsonFile, Options{})
	ctx := context.Background()
	data, err := src.Load(ctx)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "parse JSON file")
}

func TestFileSource_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	tomlFile := filepath.Join(tmpDir, "invalid.toml")
	invalidContent := `[section\nkey = "value"`
	err := os.WriteFile(tomlFile, []byte(invalidContent), 0644)
	require.NoError(t, err)

	src := New(tomlFile, Options{})
	ctx := context.Background()
	data, err := src.Load(ctx)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "parse TOML file")
}

func TestFileSource_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	txtFile := filepath.Join(tmpDir, "config.txt")
	err := os.WriteFile(txtFile, []byte("some content"), 0644)
	require.NoError(t, err)

	src := New(txtFile, Options{}) // No format specified, .txt not recognized
	ctx := context.Background()
	data, err := src.Load(ctx)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "unsupported file format")
}

func TestFileSource_Watch(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(yamlFile, []byte("key: value"), 0644)
	require.NoError(t, err)

	src := New(yamlFile, Options{})
	ctx := context.Background()
	ch, err := src.Watch(ctx)
	assert.ErrorIs(t, err, rigging.ErrWatchNotSupported)
	assert.Nil(t, ch)
}

func TestFileSource_DeepNesting(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `
level1:
  level2:
    level3:
      level4:
        key: deep-value
`
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	src := New(yamlFile, Options{})
	ctx := context.Background()
	data, err := src.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, "deep-value", data["level1.level2.level3.level4.key"])
}

func TestFileSource_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "empty.yaml")
	err := os.WriteFile(yamlFile, []byte(""), 0644)
	require.NoError(t, err)

	src := New(yamlFile, Options{})
	ctx := context.Background()
	data, err := src.Load(ctx)
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestFileSource_ArraysPreserved(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "config.json")
	jsonContent := `{
  "servers": ["server1", "server2", "server3"],
  "ports": [8080, 8081, 8082]
}`
	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	require.NoError(t, err)

	src := New(jsonFile, Options{})
	ctx := context.Background()
	data, err := src.Load(ctx)
	require.NoError(t, err)

	servers, ok := data["servers"].([]any)
	require.True(t, ok)
	assert.Len(t, servers, 3)
	assert.Equal(t, "server1", servers[0])

	ports, ok := data["ports"].([]any)
	require.True(t, ok)
	assert.Len(t, ports, 3)
}

func TestFlattenMapWithKeys_SimpleMap(t *testing.T) {
	input := map[string]any{
		"key1": "value1",
		"key2": "value2",
	}
	result := make(map[string]any)
	originalKeys := make(map[string]string)

	flattenMapWithKeys("", input, result, originalKeys)

	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, "value2", result["key2"])
	assert.Equal(t, "key1", originalKeys["key1"])
	assert.Equal(t, "key2", originalKeys["key2"])
}

func TestFlattenMapWithKeys_NestedMap(t *testing.T) {
	input := map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
		},
	}
	result := make(map[string]any)
	originalKeys := make(map[string]string)

	flattenMapWithKeys("", input, result, originalKeys)

	assert.Equal(t, "localhost", result["database.host"])
	assert.Equal(t, 5432, result["database.port"])
	assert.Equal(t, "database.host", originalKeys["database.host"])
	assert.Equal(t, "database.port", originalKeys["database.port"])
}

func TestFlattenMapWithKeys_DeepNesting(t *testing.T) {
	input := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": map[string]any{
					"key": "deep-value",
				},
			},
		},
	}
	result := make(map[string]any)
	originalKeys := make(map[string]string)

	flattenMapWithKeys("", input, result, originalKeys)

	assert.Equal(t, "deep-value", result["level1.level2.level3.key"])
	assert.Equal(t, "level1.level2.level3.key", originalKeys["level1.level2.level3.key"])
}

func TestFlattenMapWithKeys_WithPrefix(t *testing.T) {
	input := map[string]any{
		"host": "localhost",
		"port": 5432,
	}
	result := make(map[string]any)
	originalKeys := make(map[string]string)

	flattenMapWithKeys("database", input, result, originalKeys)

	assert.Equal(t, "localhost", result["database.host"])
	assert.Equal(t, 5432, result["database.port"])
	assert.Equal(t, "database.host", originalKeys["database.host"])
	assert.Equal(t, "database.port", originalKeys["database.port"])
}

func TestFlattenMapWithKeys_MapAnyAny(t *testing.T) {
	input := map[any]any{
		"key1": "value1",
		"key2": 123,
	}
	result := make(map[string]any)
	originalKeys := make(map[string]string)

	flattenMapWithKeys("", input, result, originalKeys)

	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, 123, result["key2"])
	assert.Equal(t, "key1", originalKeys["key1"])
	assert.Equal(t, "key2", originalKeys["key2"])
}

func TestFlattenMapWithKeys_MapAnyAnyNested(t *testing.T) {
	input := map[any]any{
		"database": map[any]any{
			"host": "localhost",
			"port": 5432,
		},
	}
	result := make(map[string]any)
	originalKeys := make(map[string]string)

	flattenMapWithKeys("", input, result, originalKeys)

	assert.Equal(t, "localhost", result["database.host"])
	assert.Equal(t, 5432, result["database.port"])
}

func TestFlattenMapWithKeys_MapAnyAnyNonStringKey(t *testing.T) {
	input := map[any]any{
		"valid":   "value1",
		123:       "ignored", // non-string key should be skipped
		"another": "value2",
	}
	result := make(map[string]any)
	originalKeys := make(map[string]string)

	flattenMapWithKeys("", input, result, originalKeys)

	assert.Equal(t, "value1", result["valid"])
	assert.Equal(t, "value2", result["another"])
	assert.NotContains(t, result, "123")
}

func TestFlattenMapWithKeys_MixedTypes(t *testing.T) {
	input := map[string]any{
		"string": "text",
		"number": 42,
		"bool":   true,
		"float":  3.14,
		"array":  []any{1, 2, 3},
		"nested": map[string]any{
			"key": "value",
		},
	}
	prefix := "pref"
	result := make(map[string]any)
	originalKeys := make(map[string]string)

	flattenMapWithKeys(prefix, input, result, originalKeys)

	assert.Equal(t, "text", result["pref.string"])
	assert.Equal(t, 42, result["pref.number"])
	assert.Equal(t, true, result["pref.bool"])
	assert.Equal(t, 3.14, result["pref.float"])
	assert.Equal(t, []any{1, 2, 3}, result["pref.array"])
	assert.Equal(t, "value", result["pref.nested.key"])
}

func TestFlattenMapWithKeys_EmptyMap(t *testing.T) {
	input := map[string]any{}
	result := make(map[string]any)
	originalKeys := make(map[string]string)

	flattenMapWithKeys("", input, result, originalKeys)

	assert.Empty(t, result)
	assert.Empty(t, originalKeys)
}

func TestFlattenMapWithKeys_EmptyPrefix(t *testing.T) {
	input := "value"
	result := make(map[string]any)
	originalKeys := make(map[string]string)

	// When prefix is empty, the value should not be added
	flattenMapWithKeys("", input, result, originalKeys)

	assert.Empty(t, result)
	assert.Empty(t, originalKeys)
}
