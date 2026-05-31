package models

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var _ model.ChatModel = (*ArkToolCallingChatModel)(nil)
var _ model.ToolCallingChatModel = (*ArkToolCallingChatModel)(nil)

type ArkToolCallingChatModel struct {
	baseURL   string
	apiKey    string
	modelName string
	client    *http.Client

	mu    sync.RWMutex
	tools []*schema.ToolInfo
}

func NewArkToolCallingChatModel(cfg Config) *ArkToolCallingChatModel {
	client := &http.Client{}
	if cfg.HTTPTimeout > 0 {
		client.Timeout = cfg.HTTPTimeout
	}

	return &ArkToolCallingChatModel{
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:    cfg.APIKey,
		modelName: cfg.ModelName,
		client:    client,
	}
}

func (m *ArkToolCallingChatModel) Generate(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	req, err := m.buildChatCompletionRequest(msgs, opts...)
	if err != nil {
		return nil, err
	}

	resp, err := m.doChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	return toEinoMessage(resp)
}

func (m *ArkToolCallingChatModel) Stream(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("ArkToolCallingChatModel Stream is not implemented")
}

func (m *ArkToolCallingChatModel) BindTools(tools []*schema.ToolInfo) error {
	if len(tools) == 0 {
		return errors.New("no tools to bind")
	}

	copied := make([]*schema.ToolInfo, len(tools))
	copy(copied, tools)

	m.mu.Lock()
	m.tools = copied
	m.mu.Unlock()

	return nil
}

func (m *ArkToolCallingChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	if len(tools) == 0 {
		return nil, errors.New("no tools to bind")
	}

	copiedTools := make([]*schema.ToolInfo, len(tools))
	copy(copiedTools, tools)

	return &ArkToolCallingChatModel{
		baseURL:   m.baseURL,
		apiKey:    m.apiKey,
		modelName: m.modelName,
		client:    m.client,
		tools:     copiedTools,
	}, nil
}

func (m *ArkToolCallingChatModel) buildChatCompletionRequest(msgs []*schema.Message, opts ...model.Option) (*chatCompletionRequest, error) {
	if m.baseURL == "" {
		return nil, errors.New("ark base url is required")
	}
	if m.apiKey == "" {
		return nil, errors.New("ark api key is required")
	}
	if m.modelName == "" {
		return nil, errors.New("ark model name is required")
	}

	base := &model.Options{
		Model: &m.modelName,
	}
	options := model.GetCommonOptions(base, opts...)

	modelName := m.modelName
	if options.Model != nil && strings.TrimSpace(*options.Model) != "" {
		modelName = strings.TrimSpace(*options.Model)
	}

	req := &chatCompletionRequest{
		Model:    modelName,
		Messages: make([]openAIMessage, 0, len(msgs)),
	}

	if options.Temperature != nil {
		req.Temperature = options.Temperature
	}
	if options.TopP != nil {
		req.TopP = options.TopP
	}
	if options.MaxTokens != nil {
		req.MaxTokens = options.MaxTokens
	}
	if len(options.Stop) > 0 {
		req.Stop = options.Stop
	}

	for _, msg := range msgs {
		converted, err := toOpenAIMessage(msg)
		if err != nil {
			return nil, err
		}
		req.Messages = append(req.Messages, converted)
	}

	tools := options.Tools
	if tools == nil {
		m.mu.RLock()
		if len(m.tools) > 0 {
			tools = make([]*schema.ToolInfo, len(m.tools))
			copy(tools, m.tools)
		}
		m.mu.RUnlock()
	}

	if len(tools) > 0 {
		req.Tools = make([]openAITool, 0, len(tools))
		for _, toolInfo := range tools {
			toolDef, err := toOpenAITool(toolInfo)
			if err != nil {
				return nil, err
			}
			req.Tools = append(req.Tools, toolDef)
		}
	}

	if options.ToolChoice != nil {
		req.ToolChoice = toOpenAIToolChoice(*options.ToolChoice, options.AllowedToolNames)
	}

	return req, nil
}

func (m *ArkToolCallingChatModel) doChatCompletion(ctx context.Context, reqBody *chatCompletionRequest) (*chatCompletionResponse, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ark chat request failed: status=%d, body=%s", resp.StatusCode, string(respBytes))
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, err
	}

	return &chatResp, nil
}

type chatCompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools,omitempty"`
	ToolChoice  any             `json:"tool_choice,omitempty"`
	Temperature *float32        `json:"temperature,omitempty"`
	TopP        *float32        `json:"top_p,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIContentPart struct {
	Type     string              `json:"type"`
	Text     string              `json:"text,omitempty"`
	ImageURL *openAIImageURLPart `json:"image_url,omitempty"`
	AudioURL *openAIURLPart      `json:"audio_url,omitempty"`
	VideoURL *openAIURLPart      `json:"video_url,omitempty"`
	FileURL  *openAIURLPart      `json:"file_url,omitempty"`
}

type openAIImageURLPart struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type openAIURLPart struct {
	URL string `json:"url"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	Index    *int               `json:"index,omitempty"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function openAIToolCallFunc `json:"function"`
	Extra    map[string]any     `json:"-"`
}

type openAIToolCallFunc struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type chatCompletionResponse struct {
	ID      string                 `json:"id,omitempty"`
	Model   string                 `json:"model,omitempty"`
	Choices []chatCompletionChoice `json:"choices"`
	Usage   *openAIUsage           `json:"usage,omitempty"`
}

type chatCompletionChoice struct {
	Index        int                 `json:"index"`
	Message      openAIResultMessage `json:"message"`
	FinishReason string              `json:"finish_reason,omitempty"`
}

type openAIResultMessage struct {
	Role       string           `json:"role"`
	Content    *string          `json:"content"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIUsage struct {
	PromptTokens       int `json:"prompt_tokens"`
	CompletionTokens   int `json:"completion_tokens"`
	TotalTokens        int `json:"total_tokens"`
	PromptTokenDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
	CompletionTokenDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
}

func toOpenAIMessage(msg *schema.Message) (openAIMessage, error) {
	if msg == nil {
		return openAIMessage{}, errors.New("nil schema message")
	}

	content, err := toOpenAIContent(msg)
	if err != nil {
		return openAIMessage{}, err
	}

	return openAIMessage{
		Role:       string(msg.Role),
		Content:    content,
		Name:       msg.Name,
		ToolCallID: msg.ToolCallID,
		ToolCalls:  toOpenAIToolCalls(msg.ToolCalls),
	}, nil
}

func toOpenAIContent(msg *schema.Message) (any, error) {
	if len(msg.UserInputMultiContent) > 0 {
		return inputPartsToOpenAIContent(msg.UserInputMultiContent)
	}
	if len(msg.AssistantGenMultiContent) > 0 {
		return outputPartsToOpenAIContent(msg.AssistantGenMultiContent)
	}
	if len(msg.MultiContent) > 0 {
		return legacyPartsToOpenAIContent(msg.MultiContent)
	}
	return msg.Content, nil
}

func inputPartsToOpenAIContent(parts []schema.MessageInputPart) ([]openAIContentPart, error) {
	out := make([]openAIContentPart, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case schema.ChatMessagePartTypeText:
			out = append(out, openAIContentPart{Type: string(part.Type), Text: part.Text})
		case schema.ChatMessagePartTypeImageURL:
			if part.Image == nil {
				return nil, errors.New("image part is nil")
			}
			url := firstNonEmptyPtr(part.Image.URL, part.Image.Base64Data)
			out = append(out, openAIContentPart{
				Type:     string(part.Type),
				ImageURL: &openAIImageURLPart{URL: url, Detail: string(part.Image.Detail)},
			})
		case schema.ChatMessagePartTypeAudioURL:
			if part.Audio == nil {
				return nil, errors.New("audio part is nil")
			}
			out = append(out, openAIContentPart{Type: string(part.Type), AudioURL: &openAIURLPart{URL: firstNonEmptyPtr(part.Audio.URL, part.Audio.Base64Data)}})
		case schema.ChatMessagePartTypeVideoURL:
			if part.Video == nil {
				return nil, errors.New("video part is nil")
			}
			out = append(out, openAIContentPart{Type: string(part.Type), VideoURL: &openAIURLPart{URL: firstNonEmptyPtr(part.Video.URL, part.Video.Base64Data)}})
		case schema.ChatMessagePartTypeFileURL:
			if part.File == nil {
				return nil, errors.New("file part is nil")
			}
			out = append(out, openAIContentPart{Type: string(part.Type), FileURL: &openAIURLPart{URL: firstNonEmptyPtr(part.File.URL, part.File.Base64Data)}})
		default:
			return nil, fmt.Errorf("unsupported user input message part type: %s", part.Type)
		}
	}
	return out, nil
}

func outputPartsToOpenAIContent(parts []schema.MessageOutputPart) ([]openAIContentPart, error) {
	out := make([]openAIContentPart, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case schema.ChatMessagePartTypeText:
			out = append(out, openAIContentPart{Type: string(part.Type), Text: part.Text})
		case schema.ChatMessagePartTypeImageURL:
			if part.Image == nil {
				return nil, errors.New("assistant image part is nil")
			}
			out = append(out, openAIContentPart{Type: string(part.Type), ImageURL: &openAIImageURLPart{URL: firstNonEmptyPtr(part.Image.URL, part.Image.Base64Data)}})
		case schema.ChatMessagePartTypeAudioURL:
			if part.Audio == nil {
				return nil, errors.New("assistant audio part is nil")
			}
			out = append(out, openAIContentPart{Type: string(part.Type), AudioURL: &openAIURLPart{URL: firstNonEmptyPtr(part.Audio.URL, part.Audio.Base64Data)}})
		case schema.ChatMessagePartTypeVideoURL:
			if part.Video == nil {
				return nil, errors.New("assistant video part is nil")
			}
			out = append(out, openAIContentPart{Type: string(part.Type), VideoURL: &openAIURLPart{URL: firstNonEmptyPtr(part.Video.URL, part.Video.Base64Data)}})
		default:
			return nil, fmt.Errorf("unsupported assistant output message part type: %s", part.Type)
		}
	}
	return out, nil
}

func legacyPartsToOpenAIContent(parts []schema.ChatMessagePart) ([]openAIContentPart, error) {
	out := make([]openAIContentPart, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case schema.ChatMessagePartTypeText:
			out = append(out, openAIContentPart{Type: string(part.Type), Text: part.Text})
		case schema.ChatMessagePartTypeImageURL:
			if part.ImageURL == nil {
				return nil, errors.New("legacy image_url part is nil")
			}
			out = append(out, openAIContentPart{Type: string(part.Type), ImageURL: &openAIImageURLPart{URL: part.ImageURL.URL, Detail: string(part.ImageURL.Detail)}})
		case schema.ChatMessagePartTypeAudioURL:
			if part.AudioURL == nil {
				return nil, errors.New("legacy audio_url part is nil")
			}
			out = append(out, openAIContentPart{Type: string(part.Type), AudioURL: &openAIURLPart{URL: part.AudioURL.URL}})
		case schema.ChatMessagePartTypeVideoURL:
			if part.VideoURL == nil {
				return nil, errors.New("legacy video_url part is nil")
			}
			out = append(out, openAIContentPart{Type: string(part.Type), VideoURL: &openAIURLPart{URL: part.VideoURL.URL}})
		case schema.ChatMessagePartTypeFileURL:
			if part.FileURL == nil {
				return nil, errors.New("legacy file_url part is nil")
			}
			out = append(out, openAIContentPart{Type: string(part.Type), FileURL: &openAIURLPart{URL: part.FileURL.URL}})
		default:
			return nil, fmt.Errorf("unsupported legacy message part type: %s", part.Type)
		}
	}
	return out, nil
}

func firstNonEmptyPtr(values ...*string) string {
	for _, value := range values {
		if value != nil && *value != "" {
			return *value
		}
	}
	return ""
}

func toOpenAIToolCalls(toolCalls []schema.ToolCall) []openAIToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	out := make([]openAIToolCall, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		toolType := toolCall.Type
		if toolType == "" {
			toolType = "function"
		}
		out = append(out, openAIToolCall{
			Index: toolCall.Index,
			ID:    toolCall.ID,
			Type:  toolType,
			Function: openAIToolCallFunc{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
			Extra: toolCall.Extra,
		})
	}
	return out
}

func toOpenAITool(toolInfo *schema.ToolInfo) (openAITool, error) {
	if toolInfo == nil {
		return openAITool{}, errors.New("nil tool info")
	}

	var parameters any = map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	if toolInfo.ParamsOneOf != nil {
		jsonSchema, err := toolInfo.ParamsOneOf.ToJSONSchema()
		if err != nil {
			return openAITool{}, err
		}
		parameters = jsonSchema
	}

	return openAITool{
		Type: "function",
		Function: openAIFunction{
			Name:        toolInfo.Name,
			Description: toolInfo.Desc,
			Parameters:  parameters,
		},
	}, nil
}

func toOpenAIToolChoice(choice schema.ToolChoice, allowedNames []string) any {
	switch choice {
	case schema.ToolChoiceForbidden:
		return "none"
	case schema.ToolChoiceForced:
		if len(allowedNames) == 1 {
			return map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": allowedNames[0],
				},
			}
		}
		return "required"
	case schema.ToolChoiceAllowed:
		return "auto"
	default:
		return "auto"
	}
}

func toEinoMessage(resp *chatCompletionResponse) (*schema.Message, error) {
	if resp == nil || len(resp.Choices) == 0 {
		return nil, errors.New("ark chat response choices is empty")
	}

	choice := resp.Choices[0]
	for _, candidate := range resp.Choices {
		if candidate.Index == 0 {
			choice = candidate
			break
		}
	}

	role := schema.RoleType(choice.Message.Role)
	if role == "" {
		role = schema.Assistant
	}

	content := ""
	if choice.Message.Content != nil {
		content = *choice.Message.Content
	}

	return &schema.Message{
		Role:       role,
		Content:    content,
		ToolCallID: choice.Message.ToolCallID,
		ToolCalls:  toEinoToolCalls(choice.Message.ToolCalls),
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: choice.FinishReason,
			Usage:        toEinoUsage(resp.Usage),
		},
		Extra: map[string]any{
			"id":    resp.ID,
			"model": resp.Model,
		},
	}, nil
}

func toEinoToolCalls(toolCalls []openAIToolCall) []schema.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	out := make([]schema.ToolCall, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		toolType := toolCall.Type
		if toolType == "" {
			toolType = "function"
		}
		out = append(out, schema.ToolCall{
			Index: toolCall.Index,
			ID:    toolCall.ID,
			Type:  toolType,
			Function: schema.FunctionCall{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
			Extra: toolCall.Extra,
		})
	}
	return out
}

func toEinoUsage(usage *openAIUsage) *schema.TokenUsage {
	if usage == nil {
		return nil
	}

	return &schema.TokenUsage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		PromptTokenDetails: schema.PromptTokenDetails{
			CachedTokens: usage.PromptTokenDetails.CachedTokens,
		},
		CompletionTokensDetails: schema.CompletionTokensDetails{
			ReasoningTokens: usage.CompletionTokenDetails.ReasoningTokens,
		},
	}
}
