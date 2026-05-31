package chat_pipeline

import (
	"context"
	"errors"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	einoagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

var ErrChatPipelineNotInitialized = errors.New("chat pipeline is not initialized")

type Pipeline struct {
	memoryRetriever MemoryRetriever
	memoryService   MemoryService
	chatModel       ChatModel

	reactAgent *react.Agent
}

func NewPipeline(
	ctx context.Context,
	memoryRetriever MemoryRetriever,
	memoryService MemoryService,
	chatModel ChatModel,
	toolCallingModel model.ToolCallingChatModel,
) (*Pipeline, error) {
	p := &Pipeline{
		memoryRetriever: memoryRetriever,
		memoryService:   memoryService,
		chatModel:       chatModel,
	}

	reactAgent, err := newEinoReactAgent(
		ctx,
		toolCallingModel,
		p.BaseTools(),
	)
	if err != nil {
		return nil, err
	}

	p.reactAgent = reactAgent
	return p, nil
}

func (p *Pipeline) Invoke(ctx context.Context, input ChatInput) (*ChatOutput, error) {
	var collector *TraceCollector
	if input.Debug {
		collector = NewTraceCollector(reactTraceMode)
		collector.Start("ChatPipeline.Invoke", traceInvokeInput(input))
	}

	if p.reactAgent == nil {
		if collector != nil {
			collector.Error("ChatPipeline.Invoke", ErrChatPipelineNotInitialized)
			return &ChatOutput{
				Intent: "eino_react",
				Trace:  collector.Trace(),
			}, ErrChatPipelineNotInitialized
		}
		return nil, ErrChatPipelineNotInitialized
	}

	msg := schema.UserMessage(input.Message)
	var opts []einoagent.AgentOption

	if collector != nil {
		ctx = ContextWithTraceCollector(ctx, collector)
		opts = append(opts, einoagent.WithComposeOptions(compose.WithCallbacks(NewEinoTraceHandler(collector))))
		collector.Start("EinoReactAgent.Generate", traceInvokeInput(input))
	}

	result, err := p.reactAgent.Generate(ctx, []*schema.Message{msg}, opts...)
	if err != nil {
		if collector != nil {
			collector.Error("EinoReactAgent.Generate", err)
			collector.Error("ChatPipeline.Invoke", err)
			return &ChatOutput{
				Intent: "eino_react",
				Trace:  collector.Trace(),
			}, err
		}
		return nil, err
	}
	if collector != nil {
		collector.End("EinoReactAgent.Generate", traceAnswerOutput(result.Content))
	}

	output := &ChatOutput{
		Answer: result.Content,
		Intent: "eino_react",
	}

	if input.Debug {
		collector.End("ChatPipeline.Invoke", traceAnswerOutput(result.Content))
		output.Trace = collector.Trace()
	}

	return output, nil
}

func traceAnswerOutput(answer string) map[string]any {
	return map[string]any{
		"answer_summary": summarizeTraceText(answer, 160),
	}
}

func traceInvokeInput(input ChatInput) map[string]any {
	return map[string]any{
		"message": input.Message,
		"top_k":   input.TopK,
		"type":    input.Type,
	}
}
