package memory_chat

import (
	"strings"
	"testing"

	"memoryflow/internal/ai/component/retriever"
	"memoryflow/internal/domain/model"
)

func TestBuildMemoryContextEmpty(t *testing.T) {
	if got := BuildMemoryContext(nil); got != "没有检索到相关记忆。" {
		t.Fatalf("BuildMemoryContext(nil) = %q", got)
	}
}

func TestBuildMemoryContextIncludesFields(t *testing.T) {
	got := BuildMemoryContext([]retriever.RetrievedMemory{{
		Memory: model.MemoryItem{
			ID:              7,
			Type:            "text",
			ContentText:     "项目进展",
			Summary:         "完成测试",
			Tags:            `["MemoryFlow"]`,
			Mood:            "开心",
			Location:        "上海",
			ImportanceScore: 8,
		},
		Score: 0.88,
	}})

	for _, want := range []string{"项目进展", "完成测试", `["MemoryFlow"]`, "开心", "上海", "0.8800"} {
		if !strings.Contains(got, want) {
			t.Fatalf("context missing %q: %s", want, got)
		}
	}
}

func TestBuildMemoryContextTruncatesLongContent(t *testing.T) {
	got := BuildMemoryContext([]retriever.RetrievedMemory{{
		Memory: model.MemoryItem{ID: 1, ContentText: strings.Repeat("长", 1200)},
	}})
	if strings.Count(got, "长") != 1000 || !strings.Contains(got, "...") {
		t.Fatalf("long content was not truncated, rune count=%d", strings.Count(got, "长"))
	}
}

func TestBuildAnswerPrompt(t *testing.T) {
	got := BuildAnswerPrompt("最近做了什么", "上下文内容")
	for _, want := range []string{"最近做了什么", "上下文内容", "只能基于给定的记忆内容回答", "不要编造不存在的事实", "没有在已有记忆中找到足够依据"} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}
