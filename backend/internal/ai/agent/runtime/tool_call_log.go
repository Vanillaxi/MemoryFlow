package runtime

type ToolCall struct {
	Name string
	Args map[string]any
}

type ToolCallLog struct {
	Name   string `json:"name"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}
