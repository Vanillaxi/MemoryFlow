package knowledge_pipeline

import (
	"strings"

	"memoryflow/internal/ai/indexer"
	"memoryflow/internal/domain/model"
)

//这里把 MemoryItem 转成适合 embedding 的文本。

func BuildEmbeddingText(item model.MemoryItem) string {
	var b strings.Builder

	b.WriteString("类型：")
	b.WriteString(item.Type)
	b.WriteString("\n")

	if strings.TrimSpace(item.ContentText) != "" {
		b.WriteString("内容：")
		b.WriteString(item.ContentText)
		b.WriteString("\n")
	}

	if strings.TrimSpace(item.ImageURL) != "" {
		b.WriteString("图片地址：")
		b.WriteString(item.ImageURL)
		b.WriteString("\n")
	}

	if strings.TrimSpace(item.Summary) != "" {
		b.WriteString("摘要：")
		b.WriteString(item.Summary)
		b.WriteString("\n")
	}

	if strings.TrimSpace(item.Tags) != "" {
		b.WriteString("标签：")
		b.WriteString(item.Tags)
		b.WriteString("\n")
	}

	if strings.TrimSpace(item.Location) != "" {
		b.WriteString("地点：")
		b.WriteString(item.Location)
		b.WriteString("\n")
	}

	if strings.TrimSpace(item.Mood) != "" {
		b.WriteString("情绪：")
		b.WriteString(item.Mood)
		b.WriteString("\n")
	}

	if !item.OccurredAt.IsZero() {
		b.WriteString("时间：")
		b.WriteString(item.OccurredAt.Format("2006-01-02 15:04:05"))
	}

	return b.String()
}

func ToIndexDocument(item model.MemoryItem) indexer.IndexDocument {
	return indexer.IndexDocument{
		MemoryID:   int64(item.ID),
		Content:    BuildEmbeddingText(item),
		MemoryType: item.Type,
		OccurredAt: item.OccurredAt.Unix(),
		Memory:     item,
	}
}
