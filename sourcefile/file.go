package sourcefile

import (
	"context"
	"encoding/json"
	"errors"
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

	// Root selects a dot-separated map path inside the parsed file (for example, "hub.msa").
	// When set, only that subtree is flattened and exposed as the source root.
	Root string
}

var (
	// ErrRootNotFound is returned when Options.Root does not exist in the parsed file.
	ErrRootNotFound = errors.New("sourcefile: root not found")
	// ErrRootNotMap is returned when Options.Root resolves to a non-map value.
	ErrRootNotMap = errors.New("sourcefile: root is not a map")
	// ErrInvalidRoot is returned when Options.Root has invalid dot-path syntax.
	ErrInvalidRoot = errors.New("sourcefile: invalid root path")
)

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
	result, _, err := f.LoadWithKeys(ctx)
	return result, err
}

// LoadWithKeys reads and parses the file, returning flattened configuration with original keys.
func (f *fileSource) LoadWithKeys(ctx context.Context) (map[string]any, map[string]string, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			if f.opts.Required {
				return nil, nil, fmt.Errorf("required config file not found: %s: %w", f.path, err)
			}
			return make(map[string]any), make(map[string]string), nil
		}
		return nil, nil, fmt.Errorf("read config file %s: %w", f.path, err)
	}

	format := f.opts.Format
	if format == "" {
		format = inferFormat(f.path)
	}

	var raw map[string]any
	switch format {
	case "yaml", "yml":
		if parseErr := yaml.Unmarshal(data, &raw); parseErr != nil {
			return nil, nil, fmt.Errorf("parse YAML file %s: %w", f.path, parseErr)
		}
	case "json":
		if parseErr := json.Unmarshal(data, &raw); parseErr != nil {
			return nil, nil, fmt.Errorf("parse JSON file %s: %w", f.path, parseErr)
		}
	case "toml":
		if parseErr := toml.Unmarshal(data, &raw); parseErr != nil {
			return nil, nil, fmt.Errorf("parse TOML file %s: %w", f.path, parseErr)
		}
	default:
		return nil, nil, fmt.Errorf("unsupported file format: %s (supported: yaml, json, toml)", format)
	}

	rootValue := any(raw)
	if f.opts.Root != "" {
		rootValue, err = resolveRoot(raw, f.opts.Root, f.path)
		if err != nil {
			return nil, nil, err
		}
	}

	// Flatten nested structures to dot-separated keys
	flattened := make(map[string]any)
	originalKeys := make(map[string]string)
	flattenMapWithKeys("", rootValue, flattened, originalKeys)

	return flattened, originalKeys, nil
}

func resolveRoot(raw map[string]any, root string, path string) (any, error) {
	if err := validateRoot(root); err != nil {
		return nil, err
	}

	current := any(raw)
	segments := strings.Split(root, ".")
	for i, segment := range segments {
		next, ok := lookupRootSegment(current, segment)
		if !ok {
			return nil, fmt.Errorf("%w: root %q not found in %s", ErrRootNotFound, root, path)
		}
		current = next

		if i < len(segments)-1 && !isRootMap(current) {
			return nil, fmt.Errorf("%w: root %q traverses non-map path %q in %s", ErrRootNotMap, root, strings.Join(segments[:i+1], "."), path)
		}
	}

	if !isRootMap(current) {
		return nil, fmt.Errorf("%w: root %q is not a map in %s", ErrRootNotMap, root, path)
	}

	return current, nil
}

func validateRoot(root string) error {
	if strings.HasPrefix(root, ".") || strings.HasSuffix(root, ".") || strings.Contains(root, "..") || strings.Contains(root, "*") {
		return fmt.Errorf("%w: %q", ErrInvalidRoot, root)
	}
	return nil
}

func lookupRootSegment(value any, segment string) (any, bool) {
	switch node := value.(type) {
	case map[string]any:
		v, ok := node[segment]
		return v, ok
	case map[any]any:
		v, ok := node[segment]
		return v, ok
	default:
		return nil, false
	}
}

func isRootMap(value any) bool {
	switch value.(type) {
	case map[string]any, map[any]any:
		return true
	default:
		return false
	}
}

// flattenMapWithKeys recursively flattens nested maps to dot-separated keys and tracks original keys.
func flattenMapWithKeys(prefix string, value any, result map[string]any, originalKeys map[string]string) {
	switch v := value.(type) {
	case map[string]any:
		for key, val := range v {
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "." + key
			}
			flattenMapWithKeys(newPrefix, val, result, originalKeys)
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
			flattenMapWithKeys(newPrefix, val, result, originalKeys)
		}
	default:
		if prefix != "" {
			result[prefix] = value
			originalKeys[prefix] = prefix // For files, the key is already in the right format
		}
	}
}

// Watch returns ErrWatchNotSupported (file watching not yet implemented).
func (f *fileSource) Watch(ctx context.Context) (<-chan rigging.ChangeEvent, error) {
	return nil, rigging.ErrWatchNotSupported
}

// Name returns a human-readable identifier for this source.
func (f *fileSource) Name() string {
	return "file:" + filepath.Base(f.path)
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
