package memory_summary

import (
	"fmt"
	"strings"
	"time"
)

func BuildSummaryPrompt(from, to time.Time, aggregation SummaryAggregation) string {
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
