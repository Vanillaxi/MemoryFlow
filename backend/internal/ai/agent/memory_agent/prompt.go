package memory_agent

import (
	"encoding/json"
	"fmt"
)

func BuildRouterPrompt(userMessage string, tools []ToolDefinition) string {
	toolsJSON, _ := json.MarshalIndent(tools, "", "  ")

	return fmt.Sprintf(`
你是 MemoryFlow 的个人长期记忆 Agent。

你需要根据用户问题，从工具列表中选择一个最合适的工具调用。

可用工具：
%s

工具选择规则：
1. 如果用户是在问一个需要基于长期记忆回答的问题，使用 ask_memory。
2. 如果用户只是想查找、搜索、看看原始相关记忆，使用 search_memory。
3. 如果用户问“最近我记录了什么”“最新记忆有哪些”，使用 list_recent。
4. 如果用户问某个时间段做了什么，使用 get_timeline。
5. 不要编造工具。
6. 必须严格返回 JSON，不要返回 Markdown，不要解释。

返回格式：
{
  "tool_name": "ask_memory",
  "arguments": {
    "question": "用户问题",
    "top_k": 20
  }
}

用户问题：
%s
`, string(toolsJSON), userMessage)
}

func BuildToolResultSummaryPrompt(userMessage string, toolName ToolName, toolResultJSON string) string {
	return fmt.Sprintf(`
用户的问题是：
%s

你刚刚调用的记忆工具是：
%s

工具返回的记忆数据如下：
%s

请基于这些记忆数据，给用户一个自然、简洁、有条理的中文回答。

输出要求：
1. 只输出最终回答正文。
2. 不要输出 JSON。
3. 不要输出 Go struct。
4. 不要复制原始结果。
5. 不要出现字段名，例如 Memory、Score、ContentText、OccurredAt、DeletedAt、CreatedAt。
6. 如果有多条记忆，请用自然语言总结，不要逐字段罗列。
7. 如果没有找到相关记忆，请直接说明没有找到足够相关的记忆。
`, userMessage, toolName, toolResultJSON)
}
