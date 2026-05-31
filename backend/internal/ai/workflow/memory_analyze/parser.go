package memory_analyze

import (
	"encoding/json"
	"errors"
	"strings"
)

func ParseAnalyzeResult(raw string) (*AnalyzeResult, error) {
	jsonText, err := extractJSON(raw)
	if err != nil {
		return nil, err
	}

	var result AnalyzeResult
	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		return nil, err
	}

	if result.Summary == "" {
		return nil, errors.New("summary is empty")
	}

	if len(result.Tags) == 0 {
		result.Tags = []string{"生活记录"}
	}

	result.Mood = normalizeMood(result.Mood)

	if result.ImportanceScore > 1 && result.ImportanceScore <= 10 {
		result.ImportanceScore = result.ImportanceScore / 10
	}
	if result.ImportanceScore < 0 {
		result.ImportanceScore = 0
	}
	if result.ImportanceScore > 1 {
		result.ImportanceScore = 1
	}

	return &result, nil
}

func extractJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")

	if start == -1 || end == -1 || start > end {
		return "", errors.New("no json object found")
	}

	return raw[start : end+1], nil
}

func normalizeMood(mood string) string {
	switch mood {
	case "positive", "neutral", "negative":
		return mood
	case "开心", "高兴", "成就感", "兴奋", "满足":
		return "positive"
	case "难过", "焦虑", "沮丧", "生气", "失落":
		return "negative"
	default:
		return "neutral"
	}
}
