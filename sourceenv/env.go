package sourceenv

import (
	"context"
	"os"
	"strings"

	"github.com/Azhovan/rigging"
	"github.com/Azhovan/rigging/internal/normalize"
)

// Options configures the environment variable source.
type Options struct {
	// Prefix applied to all environment variables (e.g., "APP_").
	// Only variables starting with this prefix are considered.
	Prefix string

	// CaseSensitive controls whether variable names are case-sensitive.
	// If false (default), variables are uppercased before lookup.
	CaseSensitive bool
}

// envSource implements the Source interface for environment variables.
type envSource struct {
	opts Options
}

// New creates a new environment variable source with the given options.
func New(opts Options) rigging.Source {
	return &envSource{opts: opts}
}

// Load scans environment variables, filters by prefix, normalizes keys,
// and returns a map of configuration values.
func (e *envSource) Load(ctx context.Context) (map[string]any, error) {
	result := make(map[string]any)

	// Scan all environment variables
	for _, env := range os.Environ() {
		// Split into key=value
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Filter by prefix (case-insensitive prefix matching)
		if e.opts.Prefix != "" {
			if !strings.HasPrefix(strings.ToUpper(key), strings.ToUpper(e.opts.Prefix)) {
				continue
			}
			// Strip the prefix
			key = key[len(e.opts.Prefix):]
		}

		// Skip empty keys after prefix stripping
		if key == "" {
			continue
		}

		// Normalize the key to dot-separated lowercase path
		normalizedKey := normalize.ToLowerDotPath(key)

		// Store the value as string
		result[normalizedKey] = value
	}

	return result, nil
}

// Watch returns ErrWatchNotSupported as environment variable watching
// is not supported in this implementation.
func (e *envSource) Watch(ctx context.Context) (<-chan rigging.ChangeEvent, error) {
	return nil, rigging.ErrWatchNotSupported
}
