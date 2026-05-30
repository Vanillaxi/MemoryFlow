package memory_chat_pipeline

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
			builder.WriteString(fmt.Sprintf("内容: %s\n", truncateRunes(memory.ContentText, 1000)))
		}

		if strings.TrimSpace(memory.Summary) != "" {
			builder.WriteString(fmt.Sprintf("摘要: %s\n", truncateRunes(memory.Summary, 500)))
		}

		if strings.TrimSpace(memory.Tags) != "" {
			builder.WriteString(fmt.Sprintf("标签: %s\n", truncateRunes(memory.Tags, 300)))
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

func BuildAnswerPrompt(question string, memoryContext string) string {
	return fmt.Sprintf(`
你是 MemoryFlow 的个人长期记忆 Agent。

你的任务是：根据用户的长期记忆内容，回答用户的问题。

要求：
1. 只能基于给定的记忆内容回答。
2. 如果记忆里没有相关信息，要明确说“我没有在已有记忆中找到足够依据”。
3. 不要编造不存在的事实。
4. 回答要自然、简洁，像一个真正了解用户生活轨迹的记忆助手。
5. 如果能推断时间线，可以帮用户整理出来。
6. 如果引用了某条记忆，可以用“根据某条记忆”这种自然说法。
7. 如果检索到的是图片记忆，也要结合图片说明、地点、时间回答。

用户问题：
%s

检索到的相关记忆：
%s

请基于以上记忆回答用户问题：
`, question, memoryContext)
}
