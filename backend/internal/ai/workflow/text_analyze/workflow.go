package text_analyze

import "context"

type ChatModel interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type Workflow struct {
	chatModel ChatModel
}

func NewWorkflow(chatModel ChatModel) *Workflow {
	return &Workflow{
		chatModel: chatModel,
	}
}

func (w *Workflow) Run(ctx context.Context, input TextAnalyzeInput) (*AIAnalyzeResult, error) {
	prompt := BuildPrompt(input)

	raw, err := w.chatModel.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}
	return ParseAnalyzeResult(raw)
}
