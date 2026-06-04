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

const ToolGetRecentIssues = "get_recent_issues"

type GetRecentIssuesTool struct {
	token        string
	defaultLimit int
	defaultDays  int
	client       *http.Client
	baseURL      string
}

type issueResponse struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	HTMLURL   string     `json:"html_url"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	User      gitHubUser `json:"user"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	PullRequest *struct{} `json:"pull_request"`
}

type recentIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Author    string    `json:"author"`
	Labels    []string  `json:"labels"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	URL       string    `json:"url"`
}

type recentIssuesOutput struct {
	Repository string        `json:"repository"`
	Issues     []recentIssue `json:"issues"`
}

func NewGetRecentIssuesTool(token string, configuredLimit int, configuredDays int, baseURL string, client *http.Client) *GetRecentIssuesTool {
	return &GetRecentIssuesTool{
		token:        strings.TrimSpace(token),
		defaultLimit: normalizeDefaultLimit(configuredLimit),
		defaultDays:  normalizeDefaultDays(configuredDays),
		client:       newHTTPClient(client),
		baseURL:      normalizeBaseURL(baseURL),
	}
}

func (t *GetRecentIssuesTool) Name() string {
	return ToolGetRecentIssues
}

func (t *GetRecentIssuesTool) Description() string {
	return "只读查询指定 GitHub 仓库最近 issues。会过滤 pull_request 字段，仅返回真正的 issue。repository 必须为 owner/repo。"
}

func (t *GetRecentIssuesTool) Call(ctx context.Context, args map[string]any) (string, error) {
	repository := strings.TrimSpace(stringArg(args, "repository"))
	owner, repo, err := parseRepository(repository)
	if err != nil {
		return "", fmt.Errorf("get_recent_issues: %w", err)
	}

	limit := clampLimit(intArg(args, "limit"), t.defaultLimit, maxLimit)
	days := intArg(args, "days")
	if days <= 0 {
		days = t.defaultDays
	}
	since, err := buildSince(days, stringArg(args, "since"))
	if err != nil {
		return "", fmt.Errorf("get_recent_issues: %w", err)
	}

	query := url.Values{}
	query.Set("state", normalizeEnum(stringArg(args, "state"), "open", "open", "closed", "all"))
	query.Set("sort", normalizeEnum(stringArg(args, "sort"), "updated", "created", "updated", "comments"))
	query.Set("direction", normalizeEnum(stringArg(args, "direction"), "desc", "asc", "desc"))
	query.Set("since", since)
	setLimitQuery(query, limit)
	setStringQuery(query, "labels", stringArg(args, "labels"))

	endpoint := repositoryEndpoint(t.baseURL, owner, repo, "issues", query)
	req, err := newGitHubRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return "", fmt.Errorf("get_recent_issues: %w", err)
	}
	applyGitHubHeaders(req, t.token)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("get_recent_issues: GitHub request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := readGitHubBody(resp, t.token)
	if err != nil {
		return "", fmt.Errorf("get_recent_issues: %w", err)
	}

	var issues []issueResponse
	if err := json.Unmarshal(body, &issues); err != nil {
		return "", fmt.Errorf("get_recent_issues: decode GitHub response failed: %w", err)
	}

	output := recentIssuesOutput{Repository: repository, Issues: make([]recentIssue, 0, len(issues))}
	for _, issue := range issues {
		if issue.PullRequest != nil {
			continue
		}
		output.Issues = append(output.Issues, recentIssue{
			Number:    issue.Number,
			Title:     issue.Title,
			State:     issue.State,
			Author:    issue.User.Login,
			Labels:    issueLabelNames(issue.Labels),
			CreatedAt: issue.CreatedAt,
			UpdatedAt: issue.UpdatedAt,
			URL:       issue.HTMLURL,
		})
	}

	bytes, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("get_recent_issues: encode result failed: %w", err)
	}
	return string(bytes), nil
}

func issueLabelNames(labels []struct {
	Name string `json:"name"`
}) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		if strings.TrimSpace(label.Name) != "" {
			names = append(names, label.Name)
		}
	}
	return names
}

func normalizeEnum(value string, defaultValue string, allowed ...string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	return defaultValue
}
