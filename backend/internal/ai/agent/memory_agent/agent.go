package memory_agent

import (
	"context"
	"errors"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	einoagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"memoryflow/internal/ai/component/retriever"
	"memoryflow/internal/ai/pipeline/memory_chat"
	"memoryflow/internal/domain/service"
)

var ErrMemoryAgentNotInitialized = errors.New("memory agent is not initialized")

type MemoryAgent struct {
	chatPipeline    *memory_chat.Pipeline
	memoryRetriever *retriever.MemoryRetriever
	memoryService   *service.MemoryService

	reactAgent *react.Agent
}

func NewMemoryAgent(
	ctx context.Context,
	chatPipeline *memory_chat.Pipeline,
	memoryRetriever *retriever.MemoryRetriever,
	memoryService *service.MemoryService,
	toolCallingModel model.ToolCallingChatModel,
) (*MemoryAgent, error) {
	a := &MemoryAgent{
		chatPipeline:    chatPipeline,
		memoryRetriever: memoryRetriever,
		memoryService:   memoryService,
	}

	reactAgent, err := newEinoReactAgent(
		ctx,
		toolCallingModel,
		a.BaseTools(),
	)
	if err != nil {
		return nil, err
	}

	a.reactAgent = reactAgent
	return a, nil
}

func (a *MemoryAgent) Invoke(ctx context.Context, input AgentInput) (*AgentOutput, error) {
	var collector *TraceCollector
	if input.Debug {
		collector = NewTraceCollector(reactTraceMode)
		collector.Start("MemoryAgent.Invoke", traceInvokeInput(input))
	}

	if a.reactAgent == nil {
		if collector != nil {
			collector.Error("MemoryAgent.Invoke", ErrMemoryAgentNotInitialized)
			return &AgentOutput{
				Intent: "eino_react",
				Trace:  collector.Trace(),
			}, ErrMemoryAgentNotInitialized
		}
		return nil, ErrMemoryAgentNotInitialized
	}

	msg := schema.UserMessage(input.Message)
	var opts []einoagent.AgentOption

	if collector != nil {
		ctx = ContextWithTraceCollector(ctx, collector)
		opts = append(opts, einoagent.WithComposeOptions(compose.WithCallbacks(NewEinoTraceHandler(collector))))
		collector.Start("EinoReactAgent.Generate", traceInvokeInput(input))
	}

	result, err := a.reactAgent.Generate(ctx, []*schema.Message{msg}, opts...)
	if err != nil {
		if collector != nil {
			collector.Error("EinoReactAgent.Generate", err)
			collector.Error("MemoryAgent.Invoke", err)
			return &AgentOutput{
				Intent: "eino_react",
				Trace:  collector.Trace(),
			}, err
		}
		return nil, err
	}
	if collector != nil {
		collector.End("EinoReactAgent.Generate", traceAnswerOutput(result.Content))
	}

	output := &AgentOutput{
		Answer: result.Content,
		Intent: "eino_react",
	}

	if input.Debug {
		collector.End("MemoryAgent.Invoke", traceAnswerOutput(result.Content))
		output.Trace = collector.Trace()
	}

	return output, nil
}

func traceAnswerOutput(answer string) map[string]any {
	return map[string]any{
		"answer_summary": summarizeTraceText(answer, 160),
	}
}

func summarizeTraceText(text string, limit int) string {
	runes := []rune(text)
	if limit <= 0 || len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

func traceInvokeInput(input AgentInput) map[string]any {
	return map[string]any{
		"message": input.Message,
		"top_k":   input.TopK,
		"type":    input.Type,
	}
}
