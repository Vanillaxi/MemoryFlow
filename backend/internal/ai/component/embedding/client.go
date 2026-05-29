package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

//TODO:OpenAI,后期改为eino

type Client struct {
	baseURL   string
	apiKey    string
	modelName string
	dim       int
	client    *http.Client
}

func NewClient(baseURL string, apiKey string, modelName string, dim int) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		apiKey:    apiKey,
		modelName: modelName,
		dim:       dim,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type embedRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Dimensions     int    `json:"dimensions,omitempty"`
	EncodingFormat string `json:"encoding_format,omitempty"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		return nil, errors.New("embedding base is empty")
	}
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, errors.New("embedding api key is empty")
	}
	if strings.TrimSpace(c.modelName) == "" {
		return nil, errors.New("embedding model name is empty")
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("embedding text is empty")
	}

	reqBody := embedRequest{
		Model:          c.modelName,
		Input:          text,
		Dimensions:     c.dim,
		EncodingFormat: "float",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + "/embeddings"

	log.Printf("[embedding] request url=%s, model=%s, dim=%d, text_len=%d\n",
		url,
		c.modelName,
		c.dim,
		len([]rune(text)),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding request failed:status=%d,body=%s", resp.StatusCode, string(respBytes))
	}

	var embedResp embedResponse
	if err := json.Unmarshal(respBytes, &embedResp); err != nil {
		return nil, err
	}

	if len(embedResp.Data) == 0 {
		return nil, errors.New("embedding response is empty")
	}

	vec := embedResp.Data[0].Embedding
	if c.dim > 0 && len(vec) != c.dim {
		return nil, fmt.Errorf("embedding dim is mismatch:got=%d,want=%d", len(vec), c.dim)
	}

	return vec, nil
}
