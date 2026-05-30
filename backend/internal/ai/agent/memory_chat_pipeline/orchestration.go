package memory_chat_pipeline

import (
	"context"

	"github.com/cloudwego/eino/compose"
)

func (p *Pipeline) buildGraph(ctx context.Context) error {
	graph := compose.NewGraph[ChatInput, *ChatOutput]()

	_ = graph.AddLambdaNode("init", compose.InvokableLambda(
		func(ctx context.Context, input ChatInput) (*ChatState, error) {
			return &ChatState{
				Input: input,
			}, nil
		},
	))

	_ = graph.AddLambdaNode("retrieve", compose.InvokableLambda(p.retrieve))
	_ = graph.AddLambdaNode("rerank", compose.InvokableLambda(p.rerank))
	_ = graph.AddLambdaNode("build_context", compose.InvokableLambda(p.buildContext))
	_ = graph.AddLambdaNode("build_prompt", compose.InvokableLambda(p.buildPrompt))
	_ = graph.AddLambdaNode("generate", compose.InvokableLambda(p.generate))

	_ = graph.AddLambdaNode("output", compose.InvokableLambda(
		func(ctx context.Context, state *ChatState) (*ChatOutput, error) {
			return toOutput(state), nil
		},
	))

	_ = graph.AddEdge(compose.START, "init")
	_ = graph.AddEdge("init", "retrieve")
	_ = graph.AddEdge("retrieve", "rerank")
	_ = graph.AddEdge("rerank", "build_context")
	_ = graph.AddEdge("build_context", "build_prompt")
	_ = graph.AddEdge("build_prompt", "generate")
	_ = graph.AddEdge("generate", "output")
	_ = graph.AddEdge("output", compose.END)

	runnable, err := graph.Compile(ctx)
	if err != nil {
		return err
	}

	p.runnable = runnable
	return nil
}
