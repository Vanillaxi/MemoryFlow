package memory_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/model"
	"memoryflow/internal/service"
)

type SummaryMemoryDTO struct {
	ID         uint    `json:"id"`
	Type       string  `json:"type"`
	Summary    string  `json:"summary"`
	Content    string  `json:"content,omitempty"`
	ImageURL   string  `json:"image_url,omitempty"`
	OccurredAt string  `json:"occurred_at,omitempty"`
	Location   string  `json:"location,omitempty"`
	Mood       string  `json:"mood,omitempty"`
	Score      float32 `json:"score,omitempty"`
}

type summaryTimelineGroupDTO struct {
	Date  string             `json:"date"`
	Items []SummaryMemoryDTO `json:"items"`
}

func (a *MemoryAgent) summarizeToolResult(ctx context.Context, userMessage string, toolName ToolName, result any) (string, error) {
	cleanResult := simplifyToolResult(result)

	resultJSON, err := json.MarshalIndent(cleanResult, "", "  ")
	if err != nil {
		resultJSON = []byte("[]")
	}

	prompt := BuildToolResultSummaryPrompt(userMessage, toolName, string(resultJSON))

	answer, err := a.chatModel.GenerateWithSystem(
		ctx,
		"你是 MemoryFlow 的个人长期记忆助手。你只能输出给用户看的中文自然语言回答，不要输出 JSON，不要输出 Go 结构体，不要输出字段名或调试信息。",
		prompt,
	)
	if err != nil {
		return "", err
	}

	answer = strings.TrimSpace(answer)
	if answer == "" {
		return "", fmt.Errorf("empty summary answer")
	}

	return answer, nil
}

func simplifyToolResult(result any) any {
	switch val := result.(type) {
	case []retriever.RetrievedMemory:
		items := make([]SummaryMemoryDTO, 0, len(val))
		for _, item := range val {
			items = append(items, memoryItemToSummaryDTO(item.Memory, item.Score))
		}
		return items

	case []model.MemoryItem:
		items := make([]SummaryMemoryDTO, 0, len(val))
		for _, item := range val {
			items = append(items, memoryItemToSummaryDTO(item, 0))
		}
		return items

	case []service.TimelineGroup:
		groups := make([]summaryTimelineGroupDTO, 0, len(val))
		for _, group := range val {
			items := make([]SummaryMemoryDTO, 0, len(group.Items))
			for _, item := range group.Items {
				items = append(items, memoryItemToSummaryDTO(item, 0))
			}
			groups = append(groups, summaryTimelineGroupDTO{
				Date:  group.Date,
				Items: items,
			})
		}
		return groups

	default:
		return result
	}
}

func memoryItemToSummaryDTO(item model.MemoryItem, score float32) SummaryMemoryDTO {
	dto := SummaryMemoryDTO{
		ID:       item.ID,
		Type:     item.Type,
		Summary:  item.Summary,
		Content:  truncateRunes(strings.TrimSpace(item.ContentText), 120),
		ImageURL: item.ImageURL,
		Location: item.Location,
		Mood:     item.Mood,
		Score:    score,
	}

	if !item.OccurredAt.IsZero() {
		dto.OccurredAt = item.OccurredAt.Format("2006-01-02 15:04:05")
	}

	return dto
}
