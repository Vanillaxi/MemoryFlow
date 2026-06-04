package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLimit       = 10
	maxLimit           = 20
	defaultDays        = 7
	maxErrorBodyLength = 500
	defaultBaseURL     = "https://api.github.com"
)

func newHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func normalizeBaseURL(baseURL string) string {
	if strings.TrimSpace(baseURL) == "" {
		return defaultBaseURL
	}
	return strings.TrimRight(baseURL, "/")
}

func normalizeDefaultLimit(configuredLimit int) int {
	return clampLimit(configuredLimit, defaultLimit, maxLimit)
}

func normalizeDefaultDays(configuredDays int) int {
	if configuredDays <= 0 {
		return defaultDays
	}
	return configuredDays
}

func parseRepository(repository string) (string, string, error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", errors.New("repository is required in owner/repo format")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
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

func buildSince(days int, since string) (string, error) {
	since = strings.TrimSpace(since)
	if since != "" {
		if _, err := time.Parse(time.RFC3339, since); err != nil {
			return "", fmt.Errorf("invalid since format, expected RFC3339: %w", err)
		}
		return since, nil
	}
	if days <= 0 {
		days = defaultDays
	}
	return time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339), nil
}

func repositoryEndpoint(baseURL, owner, repo, resource string, query url.Values) string {
	return fmt.Sprintf("%s/repos/%s/%s/%s?%s",
		normalizeBaseURL(baseURL),
		url.PathEscape(owner),
		url.PathEscape(repo),
		strings.TrimPrefix(resource, "/"),
		query.Encode(),
	)
}

func newGitHubRequest(ctx context.Context, method string, endpoint string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request failed: %w", err)
	}
	return req, nil
}

func applyGitHubHeaders(req *http.Request, token string) {
	req.Header.Set("Accept", "application/vnd.github+json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func readGitHubBody(resp *http.Response, token string) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read GitHub response failed: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, readGitHubError(resp, body, token)
	}
	return body, nil
}

func readGitHubError(resp *http.Response, body []byte, token string) error {
	return fmt.Errorf("GitHub API returned status=%d body=%s", resp.StatusCode, truncate(redactToken(string(body), token), maxErrorBodyLength))
}

func stringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func intArg(args map[string]any, key string) int {
	switch value := args[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		return 0
	}
}

func setStringQuery(query url.Values, key, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		query.Set(key, value)
	}
}

func setLimitQuery(query url.Values, limit int) {
	query.Set("per_page", strconv.Itoa(limit))
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}

func redactToken(value string, token string) string {
	if token == "" {
		return value
	}
	return strings.ReplaceAll(value, token, "[REDACTED]")
}
