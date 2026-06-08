package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	ToolWebSearch       = "web_search"
	defaultSearchLimit  = 5
	maxSearchLimit      = 10
	defaultSearchSource = "web"
)

var ErrWebSearchProviderNotConfigured = errors.New("web search provider is not configured")

type SearchProvider interface {
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
}

type WebSearchTool struct {
	provider SearchProvider
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
}

type webSearchOutput struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

func NewWebSearchTool(provider SearchProvider) *WebSearchTool {
	return &WebSearchTool{provider: provider}
}

func (t *WebSearchTool) Name() string {
	return ToolWebSearch
}

func (t *WebSearchTool) Description() string {
	return "只读 Web 搜索工具。query 必填；limit 可选，默认 5，最大 10。当前需要配置 SearchProvider 后才能访问真实搜索服务。"
}

func (t *WebSearchTool) Call(ctx context.Context, args map[string]any) (string, error) {
	query := strings.TrimSpace(stringArg(args, "query"))
	if query == "" {
		return "", fmt.Errorf("%s: query is required", ToolWebSearch)
	}
	limit := clampLimit(intArg(args, "limit"), defaultSearchLimit, maxSearchLimit)
	if t == nil || t.provider == nil {
		return "", ErrWebSearchProviderNotConfigured
	}

	results, err := t.provider.Search(ctx, query, limit)
	if err != nil {
		return "", fmt.Errorf("%s: provider search failed: %w", ToolWebSearch, err)
	}
	if len(results) > limit {
		results = results[:limit]
	}
	for index := range results {
		results[index].Title = strings.TrimSpace(results[index].Title)
		results[index].URL = strings.TrimSpace(results[index].URL)
		results[index].Snippet = strings.TrimSpace(results[index].Snippet)
		results[index].Source = strings.TrimSpace(results[index].Source)
		if results[index].Source == "" {
			results[index].Source = defaultSearchSource
		}
	}

	bytes, err := json.Marshal(webSearchOutput{Query: query, Results: results})
	if err != nil {
		return "", fmt.Errorf("%s: encode result failed: %w", ToolWebSearch, err)
	}
	return string(bytes), nil
}

func clampLimit(limit, defaultValue, maxValue int) int {
	if limit <= 0 {
		limit = defaultValue
	}
	if limit > maxValue {
		return maxValue
	}
	return limit
}

func stringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func intArg(args map[string]any, key string) int {
	switch value := args[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}
