package chat_pipeline

import (
	"context"
	"errors"
	"testing"

	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/domain/model"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestInvokeRequiresInitializedReActAgent(t *testing.T) {
	output, err := (&Pipeline{}).Invoke(context.Background(), ChatInput{Message: "最近做了什么", Debug: true})
	if !errors.Is(err, ErrChatPipelineNotInitialized) {
		t.Fatalf("err = %v, want %v", err, ErrChatPipelineNotInitialized)
	}
	if output == nil || output.Trace == nil || output.Trace.Error == "" {
		t.Fatalf("expected debug trace, got %#v", output)
	}
}

type fakeFlowRetriever struct{}

func (fakeFlowRetriever) Retrieve(context.Context, string, retriever.RetrieveOptions) ([]retriever.RetrievedMemory, error) {
	return []retriever.RetrievedMemory{{Memory: model.MemoryItem{ID: 1}, Score: 0.8}}, nil
}

type fakeToolCallingModel struct {
	toolName      string
	arguments     string
	answer        string
	calls         int
	boundToolName []string
}

func (f *fakeToolCallingModel) Generate(_ context.Context, messages []*schema.Message, _ ...einomodel.Option) (*schema.Message, error) {
	f.calls++
	if f.calls == 1 {
		return schema.AssistantMessage("", []schema.ToolCall{{
			ID: "call_1",
			Function: schema.FunctionCall{
				Name:      f.toolName,
				Arguments: f.arguments,
			},
		}}), nil
	}
	if !containsToolResult(messages) {
		return nil, errors.New("missing tool result")
	}
	return schema.AssistantMessage(f.answer, nil), nil
}

func (f *fakeToolCallingModel) Stream(context.Context, []*schema.Message, ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("stream is not implemented")
}

func (f *fakeToolCallingModel) WithTools(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
	for _, toolInfo := range tools {
		if toolInfo != nil {
			f.boundToolName = append(f.boundToolName, toolInfo.Name)
		}
	}
	return f, nil
}

func containsToolResult(messages []*schema.Message) bool {
	for _, message := range messages {
		if message != nil && message.Role == schema.Tool {
			return true
		}
	}
	return false
}

func TestChatPipelineRunsReActToolLoopWithFakes(t *testing.T) {
	modelClient := &fakeToolCallingModel{
		toolName:  "get_current_time",
		arguments: `{}`,
		answer:    "已经根据当前时间完成判断。",
	}
	pipeline, err := NewPipeline(context.Background(), fakeFlowRetriever{}, &fakeMemoryContextService{}, &fakeChatModel{}, modelClient)
	if err != nil {
		t.Fatal(err)
	}

	output, err := pipeline.Invoke(context.Background(), ChatInput{Message: "今天做了什么", Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	if output.Answer != modelClient.answer || modelClient.calls != 2 || len(modelClient.boundToolName) != 4 {
		t.Fatalf("unexpected output=%#v model=%#v", output, modelClient)
	}
}

func TestChatPipelineReportsLimitedEvidenceInsteadOfInventing(t *testing.T) {
	modelClient := &fakeToolCallingModel{
		toolName:  "query_long_term_memory",
		arguments: `{"mode":"timeline","from":"2026-05-01","to":"2026-05-31"}`,
		answer:    "现有记忆中没有足够依据，暂时无法总结。",
	}
	pipeline, err := NewPipeline(context.Background(), fakeFlowRetriever{}, &fakeMemoryContextService{}, &fakeChatModel{}, modelClient)
	if err != nil {
		t.Fatal(err)
	}

	output, err := pipeline.Invoke(context.Background(), ChatInput{Message: "总结五月份"})
	if err != nil {
		t.Fatal(err)
	}
	if output.Answer != modelClient.answer {
		t.Fatalf("unexpected output: %#v", output)
	}
}
