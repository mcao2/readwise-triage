package triage

// Result represents a single triage decision for a Readwise item
type Result struct {
	ID                  string              `json:"id"`
	Title               string              `json:"title"`
	URL                 string              `json:"url"`
	TriageDecision      TriageDecision      `json:"triage_decision"`
	MetadataEnhancement MetadataEnhancement `json:"metadata_enhancement"`
}

// TriageDecision represents the action for an item
type TriageDecision struct {
	Action string `json:"action"` // archive|later|read_now
	Reason string `json:"reason"`
}

// MetadataEnhancement contains suggested tags and related content
type MetadataEnhancement struct {
	SuggestedTags []string `json:"suggested_tags"`
}
