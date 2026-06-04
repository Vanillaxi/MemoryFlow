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

const ToolGetPullRequests = "get_pull_requests"

type GetPullRequestsTool struct {
	token        string
	defaultLimit int
	client       *http.Client
	baseURL      string
}

type pullRequestResponse struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	HTMLURL   string     `json:"html_url"`
	Draft     bool       `json:"draft"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	User      gitHubUser `json:"user"`
	Base      struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Head struct {
		Ref string `json:"ref"`
	} `json:"head"`
}

type pullRequest struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Author    string    `json:"author"`
	Base      string    `json:"base"`
	Head      string    `json:"head"`
	Draft     bool      `json:"draft"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	URL       string    `json:"url"`
}

type pullRequestsOutput struct {
	Repository   string        `json:"repository"`
	PullRequests []pullRequest `json:"pull_requests"`
}

func NewGetPullRequestsTool(token string, configuredLimit int, _ int, baseURL string, client *http.Client) *GetPullRequestsTool {
	return &GetPullRequestsTool{
		token:        strings.TrimSpace(token),
		defaultLimit: normalizeDefaultLimit(configuredLimit),
		client:       newHTTPClient(client),
		baseURL:      normalizeBaseURL(baseURL),
	}
}

func (t *GetPullRequestsTool) Name() string {
	return ToolGetPullRequests
}

func (t *GetPullRequestsTool) Description() string {
	return "只读查询指定 GitHub 仓库 pull requests。repository 必须为 owner/repo；支持 state、limit、sort、direction、base、head。"
}

func (t *GetPullRequestsTool) Call(ctx context.Context, args map[string]any) (string, error) {
	repository := strings.TrimSpace(stringArg(args, "repository"))
	owner, repo, err := parseRepository(repository)
	if err != nil {
		return "", fmt.Errorf("get_pull_requests: %w", err)
	}

	query := url.Values{}
	query.Set("state", normalizeEnum(stringArg(args, "state"), "open", "open", "closed", "all"))
	query.Set("sort", normalizeEnum(stringArg(args, "sort"), "updated", "created", "updated", "popularity", "long-running"))
	query.Set("direction", normalizeEnum(stringArg(args, "direction"), "desc", "asc", "desc"))
	setLimitQuery(query, clampLimit(intArg(args, "limit"), t.defaultLimit, maxLimit))
	setStringQuery(query, "base", stringArg(args, "base"))
	setStringQuery(query, "head", stringArg(args, "head"))

	endpoint := repositoryEndpoint(t.baseURL, owner, repo, "pulls", query)
	req, err := newGitHubRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return "", fmt.Errorf("get_pull_requests: %w", err)
	}
	applyGitHubHeaders(req, t.token)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("get_pull_requests: GitHub request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := readGitHubBody(resp, t.token)
	if err != nil {
		return "", fmt.Errorf("get_pull_requests: %w", err)
	}

	var prs []pullRequestResponse
	if err := json.Unmarshal(body, &prs); err != nil {
		return "", fmt.Errorf("get_pull_requests: decode GitHub response failed: %w", err)
	}

	output := pullRequestsOutput{Repository: repository, PullRequests: make([]pullRequest, 0, len(prs))}
	for _, pr := range prs {
		output.PullRequests = append(output.PullRequests, pullRequest{
			Number:    pr.Number,
			Title:     pr.Title,
			State:     pr.State,
			Author:    pr.User.Login,
			Base:      pr.Base.Ref,
			Head:      pr.Head.Ref,
			Draft:     pr.Draft,
			CreatedAt: pr.CreatedAt,
			UpdatedAt: pr.UpdatedAt,
			URL:       pr.HTMLURL,
		})
	}

	bytes, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("get_pull_requests: encode result failed: %w", err)
	}
	return string(bytes), nil
}
