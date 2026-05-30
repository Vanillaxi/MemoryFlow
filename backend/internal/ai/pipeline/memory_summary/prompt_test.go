package memory_summary

import (
	"strings"
	"testing"
	"time"
)

func TestBuildSummaryPrompt(t *testing.T) {
	got := BuildSummaryPrompt(
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 31, 23, 59, 59, 0, time.UTC),
		SummaryAggregation{Count: 2, Tags: []string{"项目"}, Moods: []string{"开心"}, Highlights: []string{"完成测试"}, MemoryList: "- memory"},
	)
	for _, want := range []string{"只能", "不要编造", "中文", "主要做了什么", "3-5", "情绪变化", "主题变化", "依据有限", "完成测试"} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q: %s", want, got)
		}
	}
}
