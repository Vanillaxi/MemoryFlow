package reranker

import (
	"encoding/json"
	"memoryflow/internal/ai/retriever"
	"sort"
	"strings"
)

type MemoryReranker struct{}

func NewMemoryReranker() *MemoryReranker {
	return &MemoryReranker{}
}

func (r *MemoryReranker) Rerank(query string, memories []retriever.RetrievedMemory, topK int) []retriever.RetrievedMemory {
	query = strings.TrimSpace(query)

	if len(memories) == 0 {
		return memories
	}

	for i := range memories {
		vectorScore := float64(memories[i].Score)

		importanceScore := memories[i].Memory.ImportanceScore / 10.0
		if importanceScore > 1 {
			importanceScore = 1
		}
		if importanceScore < 0 {
			importanceScore = 0
		}

		keywordScore := calcKeywordScore(query, memories[i])

		finalScore := vectorScore*0.7 + importanceScore*0.2 + keywordScore*0.1
		memories[i].Score = float32(finalScore)
	}

	sort.Slice(memories, func(i, j int) bool {
		return memories[i].Score > memories[j].Score
	})

	if topK <= 0 || topK > len(memories) {
		topK = len(memories)
	}
	return memories[:topK]
}

func calcKeywordScore(query string, item retriever.RetrievedMemory) float64 {
	if query == "" {
		return 0
	}

	tagsText := parseTagsText(item.Memory.Tags)

	text := strings.Join([]string{
		item.Memory.ContentText,
		item.Memory.Summary,
		item.Memory.Mood,
		item.Memory.Location,
		tagsText,
	}, "")

	text = strings.ToLower(text)
	query = strings.ToLower(query)

	score := 0.0

	// 1. 整句命中，直接加较高分
	if strings.Contains(text, query) {
		score += 0.5
	}

	// 2. 分词命中
	for _, part := range strings.Fields(query) {
		if part == "" {
			continue
		}

		if strings.Contains(text, part) {
			score += 0.2
		}
	}

	if score > 1 {
		return 1
	}

	return score
}

func parseTagsText(tags string) string {
	tags = strings.TrimSpace(tags)
	if tags == "" {
		return ""
	}

	var arr []string
	if err := json.Unmarshal([]byte(tags), &arr); err != nil {
		return strings.Join(arr, "")
	}

	// 如果不是合法 JSON，就直接当普通字符串用，防止程序炸
	return tags
}
