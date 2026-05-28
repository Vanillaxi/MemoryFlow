package rag_answer_eino

import (
	"fmt"
	"strings"

	"memoryflow/internal/ai/retriever"
)

func BuildMemoryContext(memories []retriever.RetrievedMemory) string {
	if len(memories) == 0 {
		return "没有检索到相关记忆。"
	}

	var builder strings.Builder

	for i, item := range memories {
		memory := item.Memory

		builder.WriteString(fmt.Sprintf("【记忆 %d】\n", i+1))
		builder.WriteString(fmt.Sprintf("ID: %d\n", memory.ID))
		builder.WriteString(fmt.Sprintf("类型: %s\n", memory.Type))

		if strings.TrimSpace(memory.ContentText) != "" {
			builder.WriteString(fmt.Sprintf("内容: %s\n", memory.ContentText))
		}

		if strings.TrimSpace(memory.Summary) != "" {
			builder.WriteString(fmt.Sprintf("摘要: %s\n", memory.Summary))
		}

		if strings.TrimSpace(memory.Tags) != "" {
			builder.WriteString(fmt.Sprintf("标签: %s\n", memory.Tags))
		}

		if strings.TrimSpace(memory.Mood) != "" {
			builder.WriteString(fmt.Sprintf("情绪: %s\n", memory.Mood))
		}

		if strings.TrimSpace(memory.Location) != "" {
			builder.WriteString(fmt.Sprintf("地点: %s\n", memory.Location))
		}

		if strings.TrimSpace(memory.ImageURL) != "" {
			builder.WriteString(fmt.Sprintf("图片: %s\n", memory.ImageURL))
		}

		if !memory.OccurredAt.IsZero() {
			builder.WriteString(fmt.Sprintf("发生时间: %s\n", memory.OccurredAt.Format("2006-01-02 15:04:05")))
		}

		builder.WriteString(fmt.Sprintf("重要度: %.2f\n", memory.ImportanceScore))
		builder.WriteString(fmt.Sprintf("召回分数: %.4f\n", item.Score))
		builder.WriteString("\n")
	}

	return builder.String()
}
