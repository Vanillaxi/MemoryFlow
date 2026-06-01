package github

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGetRecentCommitsToolCallsGitHubAndDoesNotExposeToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "secret-token")
	t.Setenv("GITHUB_REPOSITORY", "vanillaxi/MemoryFlow")

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/repos/vanillaxi/MemoryFlow/commits" || r.URL.Query().Get("per_page") != "10" {
			t.Fatalf("unexpected request URL: %s", r.URL.String())
		}
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Fatal("missing authorization header")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`[{"sha":"abc","html_url":"https://github.com/vanillaxi/MemoryFlow/commit/abc","commit":{"message":"add tool calling mvp","author":{"name":"Vanilla","date":"2026-06-01T00:00:00Z"}},"author":{"login":"vanillaxi"}}]`)),
		}, nil
	})}

	currentTool := NewGetRecentCommitsTool(client)
	output, err := currentTool.Call(context.Background(), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `"sha":"abc"`) || strings.Contains(output, "secret-token") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestGetRecentCommitsToolRequiresToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	currentTool := NewGetRecentCommitsTool(nil)
	if _, err := currentTool.Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"}); err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestGetRecentCommitsToolRedactsTokenFromErrorResponse(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "secret-token")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`upstream echoed secret-token`)),
		}, nil
	})}

	_, err := NewGetRecentCommitsTool(client).Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"})
	if err == nil || strings.Contains(err.Error(), "secret-token") || !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("unexpected error: %v", err)
	}
}
