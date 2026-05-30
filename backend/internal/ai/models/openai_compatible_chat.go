package models

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type ChatModel struct {
	baseURL   string
	apiKey    string
	modelName string
	client    *http.Client
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func NewChatModel(baseURL string, apiKey string, modelName string) *ChatModel {
	return &ChatModel{
		baseURL:   baseURL,
		apiKey:    apiKey,
		modelName: modelName,
		client:    &http.Client{},
	}
}

// TODO:这版是OpenAI-compatible HTTP调用，后续可再改为 Eino component。
func (m *ChatModel) Generate(ctx context.Context, prompt string) (string, error) {
	return m.GenerateWithSystem(
		ctx,
		"你是 MemoryFlow 的个人记忆分析助手，只输出 JSON。",
		prompt,
	)
}

func (m *ChatModel) GenerateWithSystem(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	if m.baseURL == "" {
		return "", errors.New("aimodel base url is required")
	}
	if m.apiKey == "" {
		return "", errors.New("aimodel api_key is required")
	}
	if m.modelName == "" {
		return "", errors.New("aimodel mode_name is required")
	}

	reqBody := chatRequest{
		Model: m.modelName,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := m.baseURL + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("chat request failed: status=%d,body=%s", resp.StatusCode, string(respBytes))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", errors.New("chat response choices is empty")
	}

	return chatResp.Choices[0].Message.Content, nil
}
