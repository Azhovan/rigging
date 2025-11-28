package rigging

import "sync"

// Provenance contains source information for all configuration fields.
type Provenance struct {
	Fields []FieldProvenance
}

// FieldProvenance describes where a single field's value came from.
type FieldProvenance struct {
	FieldPath  string // e.g., "Database.Password"
	KeyPath    string // e.g., "database.password"
	SourceName string // e.g., "env", "file:/etc/app/config.yaml"
	Secret     bool   // true if field is tagged as secret
}

// provenanceStore holds the mapping from config instances to their provenance data.
// Uses sync.Map for thread-safe access without explicit locking.
var provenanceStore sync.Map

// GetProvenance returns provenance metadata for a loaded configuration.
// Returns (nil, false) if provenance is not available for this instance.
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

// storeProvenance stores provenance data for a config instance.
// This is an internal helper used during binding.
func storeProvenance[T any](cfg *T, prov *Provenance) {
	if cfg != nil && prov != nil {
		provenanceStore.Store(cfg, prov)
	}
}

// deleteProvenance removes provenance data for a config instance.
// This is an internal helper for cleanup if needed.
func deleteProvenance[T any](cfg *T) {
	if cfg != nil {
		provenanceStore.Delete(cfg)
	}
}
