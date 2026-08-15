package triage

// ValidActions is the set of recognized triage actions.
// Used for prompt generation, result validation, and tag filtering.
var ValidActions = map[string]bool{
	"read_now":     true,
	"later":        true,
	"archive":      true,
	"delete":       true,
	"needs_review": true,
}
