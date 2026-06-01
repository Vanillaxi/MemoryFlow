package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const commitResponseJSON = `[{"sha":"abc","html_url":"https://github.com/vanillaxi/MemoryFlow/commit/abc","commit":{"message":"add tool calling mvp","author":{"name":"Vanilla","date":"2026-06-01T00:00:00Z"}},"author":{"login":"vanillaxi"}}]`

func newTestTool(server *httptest.Server, token string) *GetRecentCommitsTool {
	return NewGetRecentCommitsTool(token, 10, 7, server.URL, server.Client())
}

func TestGetRecentCommitsToolWithoutTokenOmitsAuthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		_, _ = w.Write([]byte(commitResponseJSON))
	}))
	defer server.Close()

	if _, err := newTestTool(server, "").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"}); err != nil {
		t.Fatal(err)
	}
}

func TestGetRecentCommitsToolWithTokenSetsAuthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(commitResponseJSON))
	}))
	defer server.Close()

	output, err := newTestTool(server, "secret-token").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "secret-token") {
		t.Fatalf("token leaked in output: %s", output)
	}
}

func TestGetRecentCommitsToolRequiresRepository(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	defer server.Close()

	_, err := newTestTool(server, "").Call(context.Background(), map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "owner/repo format") || requests.Load() != 0 {
		t.Fatalf("err=%v requests=%d", err, requests.Load())
	}
}

func TestGetRecentCommitsToolRejectsInvalidRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unexpected request")
	}))
	defer server.Close()

	_, err := newTestTool(server, "").Call(context.Background(), map[string]any{"repository": "invalid"})
	if err == nil || !strings.Contains(err.Error(), "owner/repo format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetRecentCommitsToolCapsLimitAndGeneratesSinceFromDays(t *testing.T) {
	before := time.Now().UTC().AddDate(0, 0, -3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("per_page"); got != "20" {
			t.Fatalf("per_page = %q, want 20", got)
		}
		assertSinceBetween(t, r.URL.Query().Get("since"), before, time.Now().UTC().AddDate(0, 0, -3))
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	_, err := newTestTool(server, "").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow", "limit": 100, "days": 3})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRecentCommitsToolSinceOverridesDays(t *testing.T) {
	const since = "2026-05-01T00:00:00Z"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("since"); got != since {
			t.Fatalf("since = %q", got)
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	_, err := newTestTool(server, "").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow", "days": 1, "since": since})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRecentCommitsToolReturnsGitHubNon2xxError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream echoed secret-token", http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := newTestTool(server, "secret-token").Call(context.Background(), map[string]any{"repository": "vanillaxi/MemoryFlow"})
	if err == nil || !strings.Contains(err.Error(), "status=401") || strings.Contains(err.Error(), "secret-token") || !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertSinceBetween(t *testing.T, raw string, before time.Time, after time.Time) {
	t.Helper()
	since, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("since = %q, expected RFC3339: %v", raw, err)
	}
	if since.Before(before.Truncate(time.Second)) || since.After(after.Add(time.Second)) {
		t.Fatalf("since = %s, want between %s and %s", since, before, after)
	}
}
