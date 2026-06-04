package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ToolGetRecentCommits = "get_recent_commits"
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
	return &GetRecentCommitsTool{
		token:        strings.TrimSpace(token),
		defaultLimit: normalizeDefaultLimit(configuredLimit),
		defaultDays:  normalizeDefaultDays(configuredDays),
		client:       newHTTPClient(client),
		baseURL:      normalizeBaseURL(baseURL),
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

	limit := clampLimit(intArg(args, "limit"), t.defaultLimit, maxLimit)

	days := intArg(args, "days")
	if days <= 0 {
		days = t.defaultDays
	}
	since, err := buildSince(days, stringArg(args, "since"))
	if err != nil {
		return "", fmt.Errorf("get_recent_commits: %w", err)
	}

	query := url.Values{}
	setLimitQuery(query, limit)
	query.Set("since", since)
	endpoint := repositoryEndpoint(t.baseURL, owner, repo, "commits", query)
	req, err := newGitHubRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return "", fmt.Errorf("get_recent_commits: %w", err)
	}
	applyGitHubHeaders(req, t.token)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("get_recent_commits: GitHub request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := readGitHubBody(resp, t.token)
	if err != nil {
		return "", fmt.Errorf("get_recent_commits: %w", err)
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
