package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const issuesResponseJSON = `[
  {"number":12,"title":"fix startup","state":"open","html_url":"https://github.com/vanillaxi/MemoryFlow/issues/12","created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-03T00:00:00Z","user":{"login":"vanillaxi"},"labels":[{"name":"bug"},{"name":"backend"}]},
  {"number":13,"title":"pr hidden from issues","state":"open","html_url":"https://github.com/vanillaxi/MemoryFlow/pull/13","created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-03T00:00:00Z","user":{"login":"vanillaxi"},"labels":[],"pull_request":{}}
]`

func newIssueTestTool(server *httptest.Server, token string) *GetRecentIssuesTool {
	return NewGetRecentIssuesTool(token, 10, 7, server.URL, server.Client())
}

func TestGetRecentIssuesToolWithoutTokenOmitsAuthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		_, _ = w.Write([]byte(issuesResponseJSON))
	}))
	defer server.Close()

	if _, err := newIssueTestTool(server, "").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"}); err != nil {
		t.Fatal(err)
	}
}

func TestGetRecentIssuesToolWithTokenSetsAuthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(issuesResponseJSON))
	}))
	defer server.Close()

	output, err := newIssueTestTool(server, "secret-token").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "secret-token") {
		t.Fatalf("token leaked in output: %s", output)
	}
}

func TestGetRecentIssuesToolRequiresRepository(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	defer server.Close()

	_, err := newIssueTestTool(server, "").Call(context.Background(), map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "owner/repo format") || requests.Load() != 0 {
		t.Fatalf("err=%v requests=%d", err, requests.Load())
	}
}

func TestGetRecentIssuesToolRejectsInvalidRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unexpected request")
	}))
	defer server.Close()

	_, err := newIssueTestTool(server, "").Call(context.Background(), map[string]any{"repository": "invalid"})
	if err == nil || !strings.Contains(err.Error(), "owner/repo format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetRecentIssuesToolQueryParametersAndLimitClamp(t *testing.T) {
	const since = "2026-05-01T00:00:00Z"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/vanillaxi/MemoryFlow/issues" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		want := map[string]string{
			"state":     "all",
			"per_page":  "20",
			"since":     since,
			"labels":    "bug,backend",
			"sort":      "comments",
			"direction": "asc",
		}
		for key, value := range want {
			if got := r.URL.Query().Get(key); got != value {
				t.Fatalf("%s = %q, want %q", key, got, value)
			}
		}
		_, _ = w.Write([]byte(issuesResponseJSON))
	}))
	defer server.Close()

	_, err := newIssueTestTool(server, "").Call(context.Background(), map[string]any{
		"repository": "vanillaxi/MemoryFlow",
		"state":      "all",
		"limit":      100,
		"since":      since,
		"labels":     "bug,backend",
		"sort":       "comments",
		"direction":  "asc",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRecentIssuesToolGeneratesSinceFromDays(t *testing.T) {
	before := time.Now().UTC().AddDate(0, 0, -5)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertSinceBetween(t, r.URL.Query().Get("since"), before, time.Now().UTC().AddDate(0, 0, -5))
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	_, err := newIssueTestTool(server, "").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow", "days": 5})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRecentIssuesToolReturnsGitHubNon2xxError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream echoed secret-token", http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := newIssueTestTool(server, "secret-token").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"})
	if err == nil || !strings.Contains(err.Error(), "status=401") || strings.Contains(err.Error(), "secret-token") || !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetRecentIssuesToolFiltersPullRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(issuesResponseJSON))
	}))
	defer server.Close()

	output, err := newIssueTestTool(server, "").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"})
	if err != nil {
		t.Fatal(err)
	}
	var decoded recentIssuesOutput
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Issues) != 1 || decoded.Issues[0].Number != 12 || len(decoded.Issues[0].Labels) != 2 {
		t.Fatalf("unexpected output: %#v", decoded)
	}
}
