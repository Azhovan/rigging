package sourcefile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azhovan/rigging"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Options configures the file source.
type Options struct {
	// Format specifies the file format: "yaml", "json", or "toml".
	// If empty, format is inferred from file extension.
	Format string

	// Required controls whether a missing file is an error.
	// If false (default), missing files return an empty map.
	Required bool
}

// fileSource implements the Source interface for file-based configuration.
type fileSource struct {
	path string
	opts Options
}

// New creates a new file-based configuration source.
func New(path string, opts Options) rigging.Source {
	return &fileSource{
		path: path,
		opts: opts,
	}
}

// Load reads and parses the configuration file.
func (f *fileSource) Load(ctx context.Context) (map[string]any, error) {
	// Read the file
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			if f.opts.Required {
				return nil, fmt.Errorf("required config file not found: %s: %w", f.path, err)
			}
			// Not required, return empty map
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("read config file %s: %w", f.path, err)
	}

	// Determine format
	format := f.opts.Format
	if format == "" {
		format = inferFormat(f.path)
	}

	// Parse based on format
	var raw map[string]any
	switch format {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse YAML file %s: %w", f.path, err)
		}
	case "json":
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse JSON file %s: %w", f.path, err)
		}
	case "toml":
		if err := toml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse TOML file %s: %w", f.path, err)
		}
	default:
		return nil, fmt.Errorf("unsupported file format: %s (supported: yaml, json, toml)", format)
	}

	// Flatten nested structures to dot-separated keys
	flattened := make(map[string]any)
	flattenMap("", raw, flattened)

	return flattened, nil
}

// Watch returns ErrWatchNotSupported as file watching is not yet implemented.
func (f *fileSource) Watch(ctx context.Context) (<-chan rigging.ChangeEvent, error) {
	return nil, rigging.ErrWatchNotSupported
}

// inferFormat determines the file format from the file extension.
func inferFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".toml":
		return "toml"
	default:
		return ""
	}
}

// flattenMap recursively flattens a nested map structure into dot-separated keys.
// For example: {database: {host: "localhost"}} becomes {"database.host": "localhost"}
func flattenMap(prefix string, value any, result map[string]any) {
	switch v := value.(type) {
	case map[string]any:
		// Recursively flatten nested maps
		for key, val := range v {
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "." + key
			}
			flattenMap(newPrefix, val, result)
		}
	case map[any]any:
		// Handle map[any]any from YAML parser
		for key, val := range v {
			keyStr, ok := key.(string)
			if !ok {
				// Skip non-string keys
				continue
			}
			newPrefix := keyStr
			if prefix != "" {
				newPrefix = prefix + "." + keyStr
			}
			flattenMap(newPrefix, val, result)
		}
	default:
		// Leaf value - store it
		if prefix != "" {
			result[prefix] = value
		}
	}
}
