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

// Options configures file source behavior.
type Options struct {
	// Format: "yaml", "json", or "toml". Auto-detected from extension if empty.
	Format string

	// Required: if true, missing files cause an error. Default: false (returns empty map).
	Required bool
}

type fileSource struct {
	path string
	opts Options
}

// New creates a file-based configuration source.
func New(path string, opts Options) rigging.Source {
	return &fileSource{
		path: path,
		opts: opts,
	}
}

// Load reads and parses the file, returning flattened configuration.
func (f *fileSource) Load(ctx context.Context) (map[string]any, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			if f.opts.Required {
				return nil, fmt.Errorf("required config file not found: %s: %w", f.path, err)
			}
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("read config file %s: %w", f.path, err)
	}

	format := f.opts.Format
	if format == "" {
		format = inferFormat(f.path)
	}

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

// Watch returns ErrWatchNotSupported (file watching not yet implemented).
func (f *fileSource) Watch(ctx context.Context) (<-chan rigging.ChangeEvent, error) {
	return nil, rigging.ErrWatchNotSupported
}

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

// flattenMap recursively flattens nested maps to dot-separated keys.
func flattenMap(prefix string, value any, result map[string]any) {
	switch v := value.(type) {
	case map[string]any:
		for key, val := range v {
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "." + key
			}
			flattenMap(newPrefix, val, result)
		}
	case map[any]any:
		for key, val := range v {
			keyStr, ok := key.(string)
			if !ok {
				continue
			}
			newPrefix := keyStr
			if prefix != "" {
				newPrefix = prefix + "." + keyStr
			}
			flattenMap(newPrefix, val, result)
		}
	default:
		if prefix != "" {
			result[prefix] = value
		}
	}
}
