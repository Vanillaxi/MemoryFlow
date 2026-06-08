package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	ToolWebFetch          = "web_fetch"
	defaultFetchTimeout   = 8 * time.Second
	maxResponseBodyBytes  = 1 << 20
	maxFetchContentRunes  = 12000
	maxFetchErrorBodySize = 500
)

type IPResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

type netResolver struct{}

type WebFetchTool struct {
	client   *http.Client
	resolver IPResolver
}

type webFetchOutput struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

func NewWebFetchTool(client *http.Client, resolver IPResolver) *WebFetchTool {
	return &WebFetchTool{
		client:   newFetchHTTPClient(client),
		resolver: newResolver(resolver),
	}
}

func (t *WebFetchTool) Name() string {
	return ToolWebFetch
}

func (t *WebFetchTool) Description() string {
	return "只读网页读取工具。url 必填，仅允许 http/https 公网地址；禁止 file、localhost 和内网 IP；返回 title、url、content。"
}

func (t *WebFetchTool) Call(ctx context.Context, args map[string]any) (string, error) {
	rawURL := strings.TrimSpace(stringArg(args, "url"))
	if rawURL == "" {
		return "", fmt.Errorf("%s: url is required", ToolWebFetch)
	}
	parsed, err := t.validateURL(ctx, rawURL)
	if err != nil {
		return "", fmt.Errorf("%s: %w", ToolWebFetch, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", fmt.Errorf("%s: build request failed: %w", ToolWebFetch, err)
	}
	req.Header.Set("User-Agent", "MemoryFlow-WebFetch/0.1 (+read-only)")
	req.Header.Set("Accept", "text/html,text/plain;q=0.8,*/*;q=0.2")

	resp, err := t.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: request failed: %w", ToolWebFetch, err)
	}
	defer resp.Body.Close()

	body, err := readLimitedBody(resp)
	if err != nil {
		return "", fmt.Errorf("%s: %w", ToolWebFetch, err)
	}
	title, content := extractReadableText(body)
	output := webFetchOutput{
		Title:   title,
		URL:     resp.Request.URL.String(),
		Content: truncateRunes(content, maxFetchContentRunes),
	}
	bytes, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("%s: encode result failed: %w", ToolWebFetch, err)
	}
	return string(bytes), nil
}

func (t *WebFetchTool) validateURL(ctx context.Context, rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("only http and https URLs are allowed")
	}
	if parsed.User != nil {
		return nil, errors.New("URLs with user info are not allowed")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return nil, errors.New("url host is required")
	}
	if isBlockedHost(host) {
		return nil, errors.New("localhost and unspecified hosts are not allowed")
	}
	if err := t.validateResolvedIPs(ctx, host); err != nil {
		return nil, err
	}
	return parsed, nil
}

func (t *WebFetchTool) validateResolvedIPs(ctx context.Context, host string) error {
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return errors.New("private, loopback, link-local, multicast, and unspecified IPs are not allowed")
		}
		return nil
	}
	addrs, err := t.ipResolver().LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve host failed: %w", err)
	}
	if len(addrs) == 0 {
		return errors.New("host did not resolve to any IP address")
	}
	for _, addr := range addrs {
		if isBlockedIP(addr.IP) {
			return errors.New("host resolves to a private, loopback, link-local, multicast, or unspecified IP")
		}
	}
	return nil
}

func (t *WebFetchTool) httpClient() *http.Client {
	if t == nil || t.client == nil {
		return newFetchHTTPClient(nil)
	}
	return t.client
}

func (t *WebFetchTool) ipResolver() IPResolver {
	if t == nil || t.resolver == nil {
		return netResolver{}
	}
	return t.resolver
}

func newFetchHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{Timeout: defaultFetchTimeout}
	} else {
		copied := *client
		client = &copied
		if client.Timeout == 0 {
			client.Timeout = defaultFetchTimeout
		}
	}
	client.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		tool := NewWebFetchTool(nil, nil)
		_, err := tool.validateURL(req.Context(), req.URL.String())
		return err
	}
	return client
}

func newResolver(resolver IPResolver) IPResolver {
	if resolver != nil {
		return resolver
	}
	return netResolver{}
}

func (netResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	return net.DefaultResolver.LookupIPAddr(ctx, host)
}

func isBlockedHost(host string) bool {
	normalized := strings.Trim(strings.ToLower(host), "[]")
	return normalized == "localhost" || normalized == "0.0.0.0"
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast()
}

func readLimitedBody(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}
	if len(body) > maxResponseBodyBytes {
		return nil, errors.New("response body exceeds 1MB limit")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("HTTP status=%d body=%s", resp.StatusCode, truncateRunes(string(body), maxFetchErrorBodySize))
	}
	return body, nil
}

func extractReadableText(body []byte) (string, string) {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return "", normalizeWhitespace(string(body))
	}
	var titleParts []string
	var textParts []string
	collectHTMLText(root, false, false, &titleParts, &textParts)
	return normalizeWhitespace(strings.Join(titleParts, " ")), normalizeWhitespace(strings.Join(textParts, " "))
}

func collectHTMLText(node *html.Node, skip bool, inTitle bool, titleParts *[]string, textParts *[]string) {
	if node.Type == html.ElementNode {
		name := strings.ToLower(node.Data)
		if name == "script" || name == "style" || name == "noscript" || name == "svg" {
			skip = true
		}
		if name == "title" {
			inTitle = true
		}
	}
	if !skip && node.Type == html.TextNode {
		text := strings.TrimSpace(node.Data)
		if text != "" {
			if inTitle {
				*titleParts = append(*titleParts, text)
			} else {
				*textParts = append(*textParts, text)
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		collectHTMLText(child, skip, inTitle, titleParts, textParts)
	}
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
