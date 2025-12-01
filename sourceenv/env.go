package sourceenv

import (
	"context"
	"os"
	"strings"

	"github.com/Azhovan/rigging"
	"github.com/Azhovan/rigging/internal/normalize"
)

// Options configures environment variable source behavior.
type Options struct {
	// Prefix filters vars starting with prefix (stripped before normalization).
	// Empty = load all vars.
	// Prefix matching behavior is controlled by CaseSensitive.
	Prefix string

	// CaseSensitive controls prefix matching (default: false).
	// When false, prefix matching is case-insensitive (APP_ matches app_, App_, etc.).
	// When true, prefix must match exactly.
	// Keys are always normalized to lowercase after prefix stripping.
	CaseSensitive bool
}

type envSource struct {
	opts Options
}

// New creates an environment variable source.
func New(opts Options) rigging.Source {
	return &envSource{opts: opts}
}

// Load scans environment variables, filters by prefix, and normalizes keys.
func (e *envSource) Load(ctx context.Context) (map[string]any, error) {
	result, _, err := e.LoadWithKeys(ctx)
	return result, err
}

// LoadWithKeys scans environment variables and returns both data and original key mappings.
func (e *envSource) LoadWithKeys(ctx context.Context) (map[string]any, map[string]string, error) {
	result := make(map[string]any)
	originalKeys := make(map[string]string)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		originalKey := parts[0]
		value := parts[1]
		key := originalKey

		if e.opts.Prefix != "" {
			var hasPrefix bool
			if e.opts.CaseSensitive {
				hasPrefix = strings.HasPrefix(key, e.opts.Prefix)
			} else {
				hasPrefix = strings.HasPrefix(strings.ToUpper(key), strings.ToUpper(e.opts.Prefix))
			}

			if !hasPrefix {
				continue
			}
			key = key[len(e.opts.Prefix):]
		}

		if key == "" {
			continue
		}

		// Normalize: FOO__BAR → foo.bar
		normalizedKey := normalize.ToLowerDotPath(key)
		result[normalizedKey] = value
		originalKeys[normalizedKey] = originalKey
	}

	return result, originalKeys, nil
}

// Watch returns ErrWatchNotSupported (env vars don't change at runtime).
func (e *envSource) Watch(ctx context.Context) (<-chan rigging.ChangeEvent, error) {
	return nil, rigging.ErrWatchNotSupported
}

// Name returns a human-readable identifier for this source.
func (e *envSource) Name() string {
	if e.opts.Prefix != "" {
		return "env:" + e.opts.Prefix
	}
	return "env"
}
