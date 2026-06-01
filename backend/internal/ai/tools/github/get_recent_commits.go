package github

import (
	"context"
	"encoding/json"
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
	ToolGetRecentCommits = "get_recent_commits"
	defaultLimit         = 10
	maxLimit             = 20
	defaultDays          = 7
	maxErrorBodyLength   = 500
)

type GetRecentCommitsTool struct {
	token        string
	defaultLimit int
	defaultDays  int
	client       *http.Client
	baseURL      string
}

type commitResponse struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Commit  struct {
		Message string `json:"message"`
		Author  struct {
			Name string    `json:"name"`
			Date time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
	Author *struct {
		Login string `json:"login"`
	} `json:"author"`
}

type recentCommit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Committed time.Time `json:"committed_at"`
	URL       string    `json:"url"`
}

type recentCommitsOutput struct {
	Repository string         `json:"repository"`
	Commits    []recentCommit `json:"commits"`
}

func NewGetRecentCommitsTool(token string, configuredLimit int, configuredDays int, baseURL string, client *http.Client) *GetRecentCommitsTool {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if configuredLimit <= 0 {
		configuredLimit = defaultLimit
	}
	if configuredLimit > maxLimit {
		configuredLimit = maxLimit
	}
	if configuredDays <= 0 {
		configuredDays = defaultDays
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.github.com"
	}
	return &GetRecentCommitsTool{
		token:        strings.TrimSpace(token),
		defaultLimit: configuredLimit,
		defaultDays:  configuredDays,
		client:       client,
		baseURL:      strings.TrimRight(baseURL, "/"),
	}
}

func (t *GetRecentCommitsTool) Name() string {
	return ToolGetRecentCommits
}

func (t *GetRecentCommitsTool) Description() string {
	return "实时查询指定 GitHub 仓库最近 commits。repository 必须为 owner/repo；limit 可选且最大 20；since 为 RFC3339，可选且优先于 days。"
}

func (t *GetRecentCommitsTool) Call(ctx context.Context, args map[string]any) (string, error) {
	repository := strings.TrimSpace(stringArg(args, "repository"))
	owner, repo, err := parseRepository(repository)
	if err != nil {
		return "", fmt.Errorf("get_recent_commits: %w", err)
	}

	limit := intArg(args, "limit")
	if limit <= 0 {
		limit = t.defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	days := intArg(args, "days")
	if days <= 0 {
		days = t.defaultDays
	}
	since := strings.TrimSpace(stringArg(args, "since"))
	if since == "" {
		since = time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	} else if _, err := time.Parse(time.RFC3339, since); err != nil {
		return "", fmt.Errorf("get_recent_commits: invalid since format, expected RFC3339: %w", err)
	}

	query := url.Values{}
	query.Set("per_page", strconv.Itoa(limit))
	query.Set("since", since)
	endpoint := fmt.Sprintf("%s/repos/%s/%s/commits?%s",
		t.baseURL,
		url.PathEscape(owner),
		url.PathEscape(repo),
		query.Encode(),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("get_recent_commits: build request failed: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("get_recent_commits: GitHub request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("get_recent_commits: read GitHub response failed: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("get_recent_commits: GitHub API returned status=%d body=%s", resp.StatusCode, truncate(redactToken(string(body), t.token), maxErrorBodyLength))
	}

	var commits []commitResponse
	if err := json.Unmarshal(body, &commits); err != nil {
		return "", fmt.Errorf("get_recent_commits: decode GitHub response failed: %w", err)
	}

	output := recentCommitsOutput{
		Repository: repository,
		Commits:    make([]recentCommit, 0, len(commits)),
	}
	for _, commit := range commits {
		author := commit.Commit.Author.Name
		if commit.Author != nil && commit.Author.Login != "" {
			author = commit.Author.Login
		}
		output.Commits = append(output.Commits, recentCommit{
			SHA:       commit.SHA,
			Message:   commit.Commit.Message,
			Author:    author,
			Committed: commit.Commit.Author.Date,
			URL:       commit.HTMLURL,
		})
	}

	bytes, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("get_recent_commits: encode result failed: %w", err)
	}
	return string(bytes), nil
}

func parseRepository(repository string) (string, string, error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", errors.New("repository is required in owner/repo format")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
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
