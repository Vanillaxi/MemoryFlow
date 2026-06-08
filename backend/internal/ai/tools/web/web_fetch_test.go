package web

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"testing"
)

type fakeResolver struct {
	ips []net.IPAddr
	err error
}

func (r fakeResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return r.ips, r.err
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestWebFetchReadsHTTPSPage(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://example.com/docs" {
			t.Fatalf("unexpected url: %s", req.URL.String())
		}
		body := `<html><head><title>Docs</title><style>.x{}</style><script>alert(1)</script></head><body><h1>Hello</h1><p>Readable content.</p></body></html>`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser{strings.NewReader(body)},
			Request:    req,
		}, nil
	})}
	tool := NewWebFetchTool(client, fakeResolver{ips: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}})

	result, err := tool.Call(context.Background(), map[string]any{"url": "https://example.com/docs"})
	if err != nil {
		t.Fatal(err)
	}
	var output webFetchOutput
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		t.Fatal(err)
	}
	if output.Title != "Docs" || output.URL != "https://example.com/docs" {
		t.Fatalf("unexpected output: %#v", output)
	}
	if !strings.Contains(output.Content, "Hello Readable content.") || strings.Contains(output.Content, "alert") || strings.Contains(output.Content, ".x") {
		t.Fatalf("unexpected content: %q", output.Content)
	}
}

func TestWebFetchRejectsBlockedURLs(t *testing.T) {
	tool := NewWebFetchTool(nil, fakeResolver{ips: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}})
	blocked := []string{
		"file:///etc/passwd",
		"http://localhost:8080",
		"http://127.0.0.1:8080",
		"http://0.0.0.0",
		"http://[::1]/",
		"http://10.0.0.1",
		"http://172.16.0.1",
		"http://192.168.0.1",
	}
	for _, rawURL := range blocked {
		if _, err := tool.Call(context.Background(), map[string]any{"url": rawURL}); err == nil {
			t.Fatalf("expected %s to be rejected", rawURL)
		}
	}
}

func TestWebFetchRejectsHostResolvingToPrivateIP(t *testing.T) {
	tool := NewWebFetchTool(nil, fakeResolver{ips: []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}})
	if _, err := tool.Call(context.Background(), map[string]any{"url": "https://example.com"}); err == nil {
		t.Fatal("expected private resolved IP to be rejected")
	}
}

type ioNopCloser struct {
	*strings.Reader
}

func (c ioNopCloser) Close() error {
	return nil
}
