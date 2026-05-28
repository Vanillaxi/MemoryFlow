package memory_agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"memoryflow/internal/ai/pipelines/memory_chat_pipeline"
	"memoryflow/internal/model"
)

const (
	IntentMemoryQA = "memory_qa"
	IntentRecent   = "recent_memory"
	IntentTimeline = "timeline"
	IntentSearch   = "search_memory"
	IntentEmpty    = "empty"
)

func (a *MemoryAgent) Orchestrate(ctx context.Context, input ChatInput) (*ChatOutput, error) {
	message := strings.TrimSpace(input.Message)
	if message == "" {
		return &ChatOutput{
			Answer: "问题不能为空。",
			Intent: IntentEmpty,
		}, nil
	}

	intent := routeIntent(message, input)

	switch intent {
	case IntentRecent:
		items, err := a.ListRecent(ctx, RecentMemoryInput{
			Limit: 10,
		})
		if err != nil {
			return nil, err
		}

		return &ChatOutput{
			Answer:     formatRecentMemories(items),
			References: buildReferencesFromItems(items),
			Intent:     IntentRecent,
		}, nil

	default:
		result, err := a.AskMemory(ctx, AskMemoryInput{
			Question:  message,
			TopK:      input.TopK,
			Type:      input.Type,
			StartTime: input.StartTime,
			EndTime:   input.EndTime,
		})
		if err != nil {
			return nil, err
		}

		return &ChatOutput{
			Answer:     result.Answer,
			References: result.References,
			Intent:     IntentMemoryQA,
		}, nil
	}
}

func routeIntent(message string, input ChatInput) string {
	msg := strings.TrimSpace(message)

	if msg == "" {
		return IntentEmpty
	}

	if strings.Contains(msg, "最近") || strings.Contains(msg, "最新") {
		return IntentRecent
	}

	if input.StartTime != nil || input.EndTime != nil {
		return IntentMemoryQA
	}

	if strings.Contains(msg, "时间线") || strings.Contains(msg, "按时间") {
		return IntentTimeline
	}

	if strings.Contains(msg, "搜索") || strings.Contains(msg, "查找") {
		return IntentSearch
	}

	return IntentMemoryQA
}

func formatRecentMemories(items []model.MemoryItem) string {
	if len(items) == 0 {
		return "你最近还没有记录任何记忆。"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("我找到了你最近的 %d 条记忆：\n", len(items)))

	for i, item := range items {
		b.WriteString(fmt.Sprintf("\n%d. ", i+1))

		if !item.OccurredAt.IsZero() {
			b.WriteString(item.OccurredAt.Format("2006-01-02 15:04"))
			b.WriteString("，")
		}

		if strings.TrimSpace(item.Summary) != "" {
			b.WriteString(item.Summary)
		} else if strings.TrimSpace(item.ContentText) != "" {
			b.WriteString(item.ContentText)
		} else {
			b.WriteString("一条未生成摘要的记忆")
		}

		if strings.TrimSpace(item.Location) != "" {
			b.WriteString("，地点：")
			b.WriteString(item.Location)
		}
	}

	return b.String()
}

func buildReferencesFromItems(items []model.MemoryItem) []memory_chat_pipeline.MemoryReference {
	refs := make([]memory_chat_pipeline.MemoryReference, 0, len(items))

	for _, item := range items {
		content := truncateRunes(item.ContentText, 120)

		ref := memory_chat_pipeline.MemoryReference{
			ID:       item.ID,
			Summary:  item.Summary,
			Content:  content,
			ImageURL: item.ImageURL,
			Location: item.Location,
			Mood:     item.Mood,
		}

		if !item.OccurredAt.IsZero() {
			ref.OccurredAt = item.OccurredAt.Format(time.RFC3339)
		}

		refs = append(refs, ref)
	}

	return refs
}

func truncateRunes(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
