package chat_pipeline

import (
	"fmt"
	"strings"
	"time"

	memorytool "memoryflow/internal/ai/tools/memory"
)

const SystemPrompt = `
你是 MemoryFlow 的个人长期记忆 Agent。

你可以使用工具来查询用户的长期记忆。

工具使用规则：
1. 用户使用“今天”“昨天”“最近一周”“本月”等相对时间时，先调用 get_current_time，再换算日期范围。
2. 查询长期记忆时，统一调用 query_long_term_memory。
3. 查找与某个主题相关的记忆证据时，使用 query_long_term_memory 的 semantic 模式。
4. 查看最近记录或某段时间内发生了什么时，使用 query_long_term_memory 的 timeline 模式。
5. 总结、回顾、复盘某段时间时，使用 query_long_term_memory 的 aggregate 模式，再基于聚合和记忆事实自然总结。
6. 需要查看某条记忆完整详情时，使用 get_memory_detail。
7. 如果工具结果不足，要说明没有足够依据。
8. 最终回答必须是自然中文，不要输出 JSON，不要输出 Go struct，不要暴露内部实现。
9. 如果需要调用工具，请直接发起 tool call，不要先输出解释性文本。
`

func BuildSummaryPrompt(from, to time.Time, aggregation memorytool.MemoryAggregation) string {
	limitedEvidence := ""
	if aggregation.Count < 3 {
		limitedEvidence = "当前记忆数量较少，请明确说明总结依据有限。\n"
	}

	return fmt.Sprintf(`你是 MemoryFlow 的个人长期记忆回顾助手。

只能基于给定记忆生成中文时间段回顾，不要编造不存在的事件。
总结这段时间主要做了什么，提炼 3-5 个重点，并概括情绪变化和主题变化。
%s
时间范围：%s 至 %s
记忆数量：%d
高频标签：%s
情绪：%s
候选重点：
%s
按时间排序的记忆：
%s
请输出自然、简洁的中文回顾总结。`,
		limitedEvidence,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
		aggregation.Count,
		strings.Join(aggregation.Tags, "、"),
		strings.Join(aggregation.Moods, "、"),
		formatPromptList(aggregation.Highlights),
		aggregation.MemoryList,
	)
}

func formatPromptList(items []string) string {
	if len(items) == 0 {
		return "- 无\n"
	}
	return "- " + strings.Join(items, "\n- ") + "\n"
}
