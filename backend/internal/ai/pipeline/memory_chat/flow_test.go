package memory_chat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"memoryflow/internal/ai/component/retriever"
	"memoryflow/internal/domain/model"
)

type fakeChatRetriever struct {
	memories []retriever.RetrievedMemory
	err      error
}

func (f *fakeChatRetriever) Retrieve(_ context.Context, _ string, _ retriever.RetrieveOptions) ([]retriever.RetrievedMemory, error) {
	return f.memories, f.err
}

type fakeChatReranker struct{}

func (f *fakeChatReranker) Rerank(_ string, memories []retriever.RetrievedMemory, _ int) []retriever.RetrievedMemory {
	return memories
}

type fakeChatModel struct {
	answer string
	err    error
	prompt string
}

func (f *fakeChatModel) Generate(_ context.Context, prompt string) (string, error) {
	f.prompt = prompt
	return f.answer, f.err
}

func TestPipelineRun(t *testing.T) {
	modelClient := &fakeChatModel{answer: " expected answer "}
	pipeline, err := NewPipeline(context.Background(), &fakeChatRetriever{
		memories: []retriever.RetrievedMemory{{Memory: model.MemoryItem{ID: 1, Summary: "memory"}, Score: 0.9}},
	}, &fakeChatReranker{}, modelClient)
	if err != nil {
		t.Fatal(err)
	}

	output, err := pipeline.Run(context.Background(), ChatInput{Question: "question"})
	if err != nil {
		t.Fatal(err)
	}
	if output.Answer != "expected answer" || len(output.References) != 1 {
		t.Fatalf("unexpected output: %#v", output)
	}
	if !strings.Contains(modelClient.prompt, "question") || !strings.Contains(modelClient.prompt, "memory") {
		t.Fatalf("unexpected prompt: %s", modelClient.prompt)
	}
}

func TestPipelineRetrieveError(t *testing.T) {
	pipeline, err := NewPipeline(context.Background(), &fakeChatRetriever{err: errors.New("retrieve failed")}, &fakeChatReranker{}, &fakeChatModel{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pipeline.Run(context.Background(), ChatInput{Question: "question"}); err == nil {
		t.Fatal("expected retrieve error")
	}
}

func TestPipelineChatModelError(t *testing.T) {
	pipeline, err := NewPipeline(context.Background(), &fakeChatRetriever{
		memories: []retriever.RetrievedMemory{{Memory: model.MemoryItem{ID: 1}}},
	}, &fakeChatReranker{}, &fakeChatModel{err: errors.New("model failed")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pipeline.Run(context.Background(), ChatInput{Question: "question"}); err == nil {
		t.Fatal("expected model error")
	}
}

func TestPipelineNoMemoryDoesNotCallModel(t *testing.T) {
	modelClient := &fakeChatModel{answer: "should not be used"}
	pipeline, err := NewPipeline(context.Background(), &fakeChatRetriever{}, &fakeChatReranker{}, modelClient)
	if err != nil {
		t.Fatal(err)
	}
	output, err := pipeline.Run(context.Background(), ChatInput{Question: "question"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.Answer, "没有在已有记忆中找到足够依据") || modelClient.prompt != "" {
		t.Fatalf("unexpected output=%#v prompt=%q", output, modelClient.prompt)
	}
}
