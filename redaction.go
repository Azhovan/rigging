package rigging

// shouldRedactField determines if a field should be redacted.
// Tag-based secret directives are authoritative, with provenance secret as fallback.
func shouldRedactField(tagCfg tagConfig, prov *FieldProvenance, inherited bool) bool {
	if inherited || tagCfg.secret {
		return true
	}
	return prov != nil && prov.Secret
}
