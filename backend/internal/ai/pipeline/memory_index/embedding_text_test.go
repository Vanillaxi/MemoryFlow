package memory_index

import (
	"strings"
	"testing"

	"memoryflow/internal/domain/model"
)

func TestBuildEmbeddingTextIncludesFields(t *testing.T) {
	got := BuildEmbeddingText(model.MemoryItem{
		Type:        "text",
		ContentText: "content",
		Summary:     "summary",
		Tags:        `["tag"]`,
		Mood:        "happy",
		Location:    "Shanghai",
	})
	for _, want := range []string{"内容：content", "摘要：summary", `标签：["tag"]`, "情绪：happy", "地点：Shanghai"} {
		if !strings.Contains(got, want) {
			t.Fatalf("embedding text missing %q: %s", want, got)
		}
	}
}

func TestBuildEmbeddingTextSkipsEmptyFields(t *testing.T) {
	got := BuildEmbeddingText(model.MemoryItem{Type: "text"})
	for _, unwanted := range []string{"内容：", "摘要：", "标签：", "情绪：", "地点：", "图片地址："} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("embedding text contains empty field %q: %s", unwanted, got)
		}
	}
}
