package rigging

import "sync"

// Provenance contains source information for configuration fields.
type Provenance struct {
	Fields []FieldProvenance
}

// FieldProvenance describes where a field's value came from.
type FieldProvenance struct {
	FieldPath  string // Dot notation (e.g., "Database.Host")
	KeyPath    string // Normalized key (e.g., "database.host")
	SourceName string // Source identifier (e.g., "env:APP_PORT")
	Secret     bool   // Whether field is secret
}

var provenanceStore sync.Map

// GetProvenance returns provenance metadata for a loaded configuration.
// Thread-safe.
func GetProvenance[T any](cfg *T) (*Provenance, bool) {
	if cfg == nil {
		return nil, false
	}

	value, ok := provenanceStore.Load(cfg)
	if !ok {
		return nil, false
	}

	prov, ok := value.(*Provenance)
	return prov, ok
}

func storeProvenance[T any](cfg *T, prov *Provenance) {
	if cfg != nil && prov != nil {
		provenanceStore.Store(cfg, prov)
	}
}

func deleteProvenance[T any](cfg *T) {
	if cfg != nil {
		provenanceStore.Delete(cfg)
	}
}
