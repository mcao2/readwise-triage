package triage

// Result represents a single triage decision for a Readwise item
type Result struct {
	ID                  string              `json:"id"`
	Title               string              `json:"title"`
	URL                 string              `json:"url"`
	TriageDecision      TriageDecision      `json:"triage_decision"`
	ContentAnalysis     ContentAnalysis     `json:"content_analysis"`
	CredibilityCheck    CredibilityCheck    `json:"credibility_check"`
	ReadingGuide        ReadingGuide        `json:"reading_guide"`
	MetadataEnhancement MetadataEnhancement `json:"metadata_enhancement"`
}

// TriageDecision represents the action and priority for an item
type TriageDecision struct {
	Action   string `json:"action"`   // delete|archive|later|read_now
	Priority string `json:"priority"` // high|medium|low
	Reason   string `json:"reason"`
}

// ContentAnalysis represents the analysis of the content
type ContentAnalysis struct {
	Type           string   `json:"type"` // tutorial|tool_doc|opinion|analysis|news|research|other
	KeyTopics      []string `json:"key_topics"`
	EffortRequired string   `json:"effort_required"` // 5 mins skim|15 mins focused|1 hour deep
	BestReadWhen   string   `json:"best_read_when"`
}

// CredibilityCheck represents credibility assessment
type CredibilityCheck struct {
	AuthorBackground string   `json:"author_background"`
	EvidenceType     string   `json:"evidence_type"` // first_hand|data_backed|opinion_based|aggregate
	Recency          string   `json:"recency"`
	RiskFlags        []string `json:"risk_flags"`
}

// ReadingGuide provides guidance on how to read the item
type ReadingGuide struct {
	WhyValuable   string   `json:"why_valuable"`
	ReadFor       []string `json:"read_for"`
	SkipSections  string   `json:"skip_sections"`
	ActionItems   []string `json:"action_items"`
	Prerequisites []string `json:"prerequisites"`
}

// MetadataEnhancement contains suggested tags and related content
type MetadataEnhancement struct {
	SuggestedTags []string `json:"suggested_tags"`
	RelatedReads  []string `json:"related_reads"`
	SaveAs        string   `json:"save_as"`
}
