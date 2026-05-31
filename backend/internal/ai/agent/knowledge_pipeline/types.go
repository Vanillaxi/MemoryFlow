package knowledge_pipeline

type ReindexInput struct {
	BatchSize int
}

type ReindexOutput struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}
