package memory_react_agent

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

func newEinoReactAgent(
	ctx context.Context,
	toolCallingModel model.ToolCallingChatModel,
	tools []tool.BaseTool,
) (*react.Agent, error) {
	return react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: toolCallingModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			messages := make([]*schema.Message, 0, len(input)+1)
			messages = append(messages, schema.SystemMessage(SystemPrompt))
			messages = append(messages, input...)
			return messages
		},
		MaxStep: 20,
	})
}
