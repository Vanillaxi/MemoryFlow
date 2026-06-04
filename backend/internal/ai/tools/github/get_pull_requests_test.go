package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const pullRequestsResponseJSON = `[{"number":3,"title":"add tools","state":"open","html_url":"https://github.com/vanillaxi/MemoryFlow/pull/3","draft":false,"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-03T00:00:00Z","user":{"login":"vanillaxi"},"base":{"ref":"main"},"head":{"ref":"feature/tool-calling"}}]`

func newPRTestTool(server *httptest.Server, token string) *GetPullRequestsTool {
	return NewGetPullRequestsTool(token, 10, 7, server.URL, server.Client())
}

func TestGetPullRequestsToolWithoutTokenOmitsAuthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		_, _ = w.Write([]byte(pullRequestsResponseJSON))
	}))
	defer server.Close()

	if _, err := newPRTestTool(server, "").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"}); err != nil {
		t.Fatal(err)
	}
}

func TestGetPullRequestsToolWithTokenSetsAuthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(pullRequestsResponseJSON))
	}))
	defer server.Close()

	output, err := newPRTestTool(server, "secret-token").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "secret-token") {
		t.Fatalf("token leaked in output: %s", output)
	}
}

func TestGetPullRequestsToolRequiresRepository(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	defer server.Close()

	_, err := newPRTestTool(server, "").Call(context.Background(), map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "owner/repo format") || requests.Load() != 0 {
		t.Fatalf("err=%v requests=%d", err, requests.Load())
	}
}

func TestGetPullRequestsToolRejectsInvalidRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unexpected request")
	}))
	defer server.Close()

	_, err := newPRTestTool(server, "").Call(context.Background(), map[string]any{"repository": "invalid"})
	if err == nil || !strings.Contains(err.Error(), "owner/repo format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPullRequestsToolQueryParametersAndLimitClamp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/vanillaxi/MemoryFlow/pulls" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		want := map[string]string{
			"state":     "all",
			"per_page":  "20",
			"sort":      "long-running",
			"direction": "asc",
			"base":      "main",
			"head":      "vanillaxi:feature/tool-calling",
		}
		for key, value := range want {
			if got := r.URL.Query().Get(key); got != value {
				t.Fatalf("%s = %q, want %q", key, got, value)
			}
		}
		_, _ = w.Write([]byte(pullRequestsResponseJSON))
	}))
	defer server.Close()

	_, err := newPRTestTool(server, "").Call(context.Background(), map[string]any{
		"repository": "vanillaxi/MemoryFlow",
		"state":      "all",
		"limit":      100,
		"sort":       "long-running",
		"direction":  "asc",
		"base":       "main",
		"head":       "vanillaxi:feature/tool-calling",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetPullRequestsToolReturnsGitHubNon2xxError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream echoed secret-token", http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := newPRTestTool(server, "secret-token").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"})
	if err == nil || !strings.Contains(err.Error(), "status=401") || strings.Contains(err.Error(), "secret-token") || !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPullRequestsToolParsesOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(pullRequestsResponseJSON))
	}))
	defer server.Close()

	output, err := newPRTestTool(server, "").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"})
	if err != nil {
		t.Fatal(err)
	}
	var decoded pullRequestsOutput
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.PullRequests) != 1 || decoded.PullRequests[0].Number != 3 || decoded.PullRequests[0].Base != "main" || decoded.PullRequests[0].Head != "feature/tool-calling" {
		t.Fatalf("unexpected output: %#v", decoded)
	}
}
