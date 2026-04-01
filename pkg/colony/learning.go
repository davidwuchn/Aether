package colony

// Observation represents a single learning observation in learning-observations.json.
type Observation struct {
	ContentHash      string   `json:"content_hash"`
	Content          string   `json:"content"`
	WisdomType       string   `json:"wisdom_type"`
	ObservationCount int      `json:"observation_count"`
	FirstSeen        string   `json:"first_seen"`
	LastSeen         string   `json:"last_seen"`
	Colonies         []string `json:"colonies"`
}

// LearningFile represents the top-level learning-observations.json file.
type LearningFile struct {
	Observations []Observation `json:"observations"`
}
