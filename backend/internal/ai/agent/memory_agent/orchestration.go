package memory_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"memoryflow/internal/ai/pipelines/memory_chat_pipeline"
	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/model"
)

const (
	IntentMemoryQA = "memory_qa"
	IntentRecent   = "recent_memory"
	IntentTimeline = "timeline"
	IntentSearch   = "search_memory"
	IntentEmpty    = "empty"

	IntentFallbackMemoryQA = "fallback_memory_qa"
)

func (a *MemoryAgent) Orchestrate(ctx context.Context, input ChatInput) (*ChatOutput, error) {
	message := strings.TrimSpace(input.Message)
	if message == "" {
		output := &ChatOutput{
			Answer: "问题不能为空。",
			Intent: IntentEmpty,
		}

		if input.Debug {
			output.Trace = &AgentTrace{
				UsedFallback: false,
				Error:        "empty message",
			}
		}

		return output, nil
	}

	decision, err := a.routeByLLM(ctx, message)
	if err != nil {
		return a.fallbackAskMemory(ctx, input, err)
	}

	enrichDecisionWithInput(decision, input)

	var trace *AgentTrace
	if input.Debug {
		trace = &AgentTrace{
			RouterTool:      string(decision.ToolName),
			RouterArguments: decision.Arguments,
			UsedFallback:    false,
		}
	}

	toolResult, err := a.CallTool(ctx, ToolCall{
		Name:      decision.ToolName,
		Arguments: decision.Arguments,
	})
	if err != nil {
		return a.fallbackAskMemory(ctx, input, err)
	}

	if trace != nil {
		trace.ToolResultCount = countToolResult(toolResult.Result)
	}

	output, err := a.formatToolResult(ctx, message, decision.ToolName, toolResult)
	if err != nil {
		return nil, err
	}

	if trace != nil {
		trace.Summarized = shouldBeSummarized(decision.ToolName)
		output.Trace = trace
	}

	return output, nil
}

func (a *MemoryAgent) routeByLLM(ctx context.Context, message string) (*RouterDecision, error) {
	prompt := BuildRouterPrompt(message, a.ListTools())

	raw, err := a.chatModel.GenerateWithSystem(
		ctx,
		"你是 MemoryFlow 的工具路由器。你必须只输出合法 JSON，不要输出 Markdown，不要解释。",
		prompt,
	)
	if err != nil {
		return nil, err
	}

	jsonText, err := extractJSONObject(raw)
	if err != nil {
		return nil, err
	}

	var decision RouterDecision
	if err := json.Unmarshal([]byte(jsonText), &decision); err != nil {
		return nil, err
	}

	if decision.ToolName == "" {
		return nil, fmt.Errorf("tool_name is empty")
	}

	if decision.Arguments == nil {
		decision.Arguments = map[string]any{}
	}

	return &decision, nil
}

func enrichDecisionWithInput(decision *RouterDecision, input ChatInput) {
	if decision.Arguments == nil {
		decision.Arguments = map[string]any{}
	}

	if input.TopK > 0 {
		decision.Arguments["top_k"] = input.TopK
	}

	if strings.TrimSpace(input.Type) != "" {
		decision.Arguments["type"] = input.Type
	}

	if input.StartTime != nil {
		decision.Arguments["start"] = input.StartTime.Format("2006-01-02")
	}

	if input.EndTime != nil {
		decision.Arguments["end"] = input.EndTime.Format("2006-01-02")
	}
}

func (a *MemoryAgent) fallbackAskMemory(ctx context.Context, input ChatInput, reason error) (*ChatOutput, error) {
	result, err := a.AskMemory(ctx, AskMemoryInput{
		Question:  input.Message,
		TopK:      input.TopK,
		Type:      input.Type,
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
	})
	if err != nil {
		return nil, fmt.Errorf("router failed: %v, fallback failed: %w", reason, err)
	}

	output := &ChatOutput{
		Answer:     result.Answer,
		References: result.References,
		Intent:     IntentFallbackMemoryQA,
	}

	if input.Debug {
		output.Trace = &AgentTrace{
			UsedFallback: true,
			Error:        reason.Error(),
		}
	}

	return output, nil
}

func (a *MemoryAgent) formatToolResult(ctx context.Context, userMessage string, toolName ToolName, toolResult *ToolResult) (*ChatOutput, error) {
	switch toolName {
	case ToolAskMemory:
		result, ok := toolResult.Result.(*memory_chat_pipeline.ChatOutput)
		if ok {
			return &ChatOutput{
				Answer:     result.Answer,
				References: result.References,
				Intent:     string(ToolAskMemory),
			}, nil
		}

		answer, err := a.summarizeToolResult(ctx, userMessage, toolName, toolResult.Result)
		if err != nil {
			answer = "我已经找到了相关记忆，但暂时没能整理成自然语言回答。"
		}

		return &ChatOutput{
			Answer: answer,
			Intent: string(ToolAskMemory),
		}, nil

	case ToolListRecent:
		items, ok := toolResult.Result.([]model.MemoryItem)
		if ok {
			return &ChatOutput{
				Answer:     formatRecentMemories(items),
				References: buildReferencesFromItems(items),
				Intent:     string(ToolListRecent),
			}, nil
		}

		answer, err := a.summarizeToolResult(ctx, userMessage, toolName, toolResult.Result)
		if err != nil {
			answer = "我查到了最近记忆，但暂时没能整理成自然语言回答。"
		}

		return &ChatOutput{
			Answer: answer,
			Intent: string(ToolListRecent),
		}, nil

	case ToolSearchMemory:
		items, ok := toolResult.Result.([]retriever.RetrievedMemory)
		if ok {
			answer, err := a.summarizeToolResult(ctx, userMessage, toolName, items)
			if err != nil {
				answer = "我查到了相关记忆，但暂时没能整理成自然语言总结。"
			}

			return &ChatOutput{
				Answer:     answer,
				References: buildReferencesFromRetrieved(items),
				Intent:     string(ToolSearchMemory),
			}, nil
		}

		answer, err := a.summarizeToolResult(ctx, userMessage, toolName, toolResult.Result)
		if err != nil {
			answer = "我搜索到了相关内容，但暂时没能整理成自然语言回答。"
		}

		return &ChatOutput{
			Answer: answer,
			Intent: string(ToolSearchMemory),
		}, nil

	case ToolGetTimeline:
		answer, err := a.summarizeToolResult(ctx, userMessage, toolName, toolResult.Result)
		if err != nil {
			answer = "我查到了这段时间的记忆，但暂时没能把它们整理成自然语言总结。"
		}

		return &ChatOutput{
			Answer: answer,
			Intent: string(ToolGetTimeline),
		}, nil

	default:
		answer, err := a.summarizeToolResult(ctx, userMessage, toolName, toolResult.Result)
		if err != nil {
			answer = "我已经调用了相关记忆工具，但暂时没能把结果整理成自然语言回答。"
		}

		return &ChatOutput{
			Answer: answer,
			Intent: string(toolName),
		}, nil
	}
}

func extractJSONObject(raw string) (string, error) {
	raw = strings.TrimSpace(raw)

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")

	if start == -1 || end == -1 || start > end {
		return "", fmt.Errorf("no json object found")
	}

	return raw[start : end+1], nil
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

func formatSearchMemories(items []retriever.RetrievedMemory) string {
	if len(items) == 0 {
		return "没有搜索到相关记忆。"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("我搜索到了 %d 条相关记忆：\n", len(items)))

	for i, item := range items {
		memory := item.Memory

		b.WriteString(fmt.Sprintf("\n%d. ", i+1))

		if !memory.OccurredAt.IsZero() {
			b.WriteString(memory.OccurredAt.Format("2006-01-02 15:04"))
			b.WriteString("，")
		}

		if strings.TrimSpace(memory.Summary) != "" {
			b.WriteString(memory.Summary)
		} else if strings.TrimSpace(memory.ContentText) != "" {
			b.WriteString(memory.ContentText)
		} else {
			b.WriteString("一条未生成摘要的记忆")
		}

		if strings.TrimSpace(memory.Location) != "" {
			b.WriteString("，地点：")
			b.WriteString(memory.Location)
		}

		b.WriteString(fmt.Sprintf("，相关度：%.4f", item.Score))
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

func buildReferencesFromRetrieved(items []retriever.RetrievedMemory) []memory_chat_pipeline.MemoryReference {
	refs := make([]memory_chat_pipeline.MemoryReference, 0, len(items))

	for _, item := range items {
		memory := item.Memory
		content := truncateRunes(memory.ContentText, 120)

		ref := memory_chat_pipeline.MemoryReference{
			ID:       memory.ID,
			Summary:  memory.Summary,
			Content:  content,
			ImageURL: memory.ImageURL,
			Location: memory.Location,
			Mood:     memory.Mood,
			Score:    item.Score,
		}

		if !memory.OccurredAt.IsZero() {
			ref.OccurredAt = memory.OccurredAt.Format(time.RFC3339)
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

func countToolResult(result any) int {
	switch v := result.(type) {
	case []retriever.RetrievedMemory:
		return len(v)
	case []model.MemoryItem:
		return len(v)
	default:
		return 0
	}
}

func shouldBeSummarized(toolName ToolName) bool {
	switch toolName {
	case ToolSearchMemory, ToolGetTimeline:
		return true
	default:
		return false
	}
}
