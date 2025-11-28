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
		name      string
		filename  string
		content   string
		expected  map[string]any
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
