package runtime

import (
	"encoding/json"
	"fmt"
)

type ContextBuilder struct{}

func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{}
}

func (b *ContextBuilder) Build(message string, intent string, logs []ToolCallLog) (string, error) {
	resultJSON, err := json.Marshal(logs)
	if err != nil {
		return "", fmt.Errorf("marshal tool results failed: %w", err)
	}
	return fmt.Sprintf(`用户问题：
%s

识别出的 intent：
%s

工具结果 JSON：
%s

请基于工具结果回答用户。`, message, intent, string(resultJSON)), nil
}

const SummarySystemPrompt = `你是 MemoryFlow 的 Tool Calling MVP 总结器。
请使用自然、简洁的中文回答。
只能基于用户问题和工具结果回答，不要编造事实。
工具失败时，明确说明缺少哪项信息以及错误原因，但不要暴露任何 token、密钥或 Authorization header。
不要输出内部 JSON，不要描述内部执行流程。`
