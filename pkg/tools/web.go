package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	ddgSearchBase = "https://api.duckduckgo.com/"
	fetchMaxBytes = 512 * 1024
	fetchMaxChars = 4000
	webTimeout    = 10 * time.Second
)

func RegisterWeb(r *Registry) {
	registerWeb(r, &http.Client{Timeout: webTimeout}, ddgSearchBase)
}

func registerWeb(r *Registry, client *http.Client, searchBase string) {
	r.Register(NewNativeInfoTool(
		"web_search",
		"Search the web for facts you are missing. Returns a short text summary of the top results.",
		json.RawMessage(`{
			"type": "object",
			"properties": {"query": {"type": "string", "description": "what to search for"}},
			"required": ["query"]
		}`),
		webSearch(client, searchBase),
	))
	r.Register(NewNativeInfoTool(
		"fetch_url",
		"Fetch a web page by URL and return its readable text (HTML stripped).",
		json.RawMessage(`{
			"type": "object",
			"properties": {"url": {"type": "string", "description": "absolute http(s) URL"}},
			"required": ["url"]
		}`),
		fetchURL(client),
	))
}

func webSearch(client *http.Client, searchBase string) NativeFunc {
	return func(ctx context.Context, args json.RawMessage) (Result, error) {
		query, err := stringArg(args, "web_search", "query")
		if err != nil {
			return Result{IsError: true}, err
		}
		q := url.Values{"q": {query}, "format": {"json"}, "no_html": {"1"}, "no_redirect": {"1"}, "t": {"magi"}}
		body, err := httpGet(ctx, client, searchBase+"?"+q.Encode(), fetchMaxBytes)
		if err != nil {
			return Result{IsError: true}, err
		}
		return Result{Content: summarizeDDG(body)}, nil
	}
}

func fetchURL(client *http.Client) NativeFunc {
	return func(ctx context.Context, args json.RawMessage) (Result, error) {
		target, err := stringArg(args, "fetch_url", "url")
		if err != nil {
			return Result{IsError: true}, err
		}
		body, err := httpGet(ctx, client, target, fetchMaxBytes)
		if err != nil {
			return Result{IsError: true}, err
		}
		return Result{Content: truncate(htmlToText(body), fetchMaxChars)}, nil
	}
}

func stringArg(args json.RawMessage, tool, field string) (string, error) {
	var m map[string]string
	if err := json.Unmarshal(args, &m); err != nil {
		return "", fmt.Errorf("%s args: %w", tool, err)
	}
	v := strings.TrimSpace(m[field])
	if v == "" {
		return "", fmt.Errorf("%s: empty %s", tool, field)
	}
	return v, nil
}

func httpGet(ctx context.Context, client *http.Client, target string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "magi/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET %s: %s", target, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}

func summarizeDDG(body []byte) string {
	var d struct {
		AbstractText  string `json:"AbstractText"`
		RelatedTopics []struct {
			Text string `json:"Text"`
		} `json:"RelatedTopics"`
	}
	if err := json.Unmarshal(body, &d); err != nil {
		return truncate(strings.TrimSpace(string(body)), fetchMaxChars)
	}
	var parts []string
	if d.AbstractText != "" {
		parts = append(parts, d.AbstractText)
	}
	for _, rt := range d.RelatedTopics {
		if rt.Text != "" {
			parts = append(parts, "- "+rt.Text)
		}
	}
	if len(parts) == 0 {
		return "no results"
	}
	return truncate(strings.Join(parts, "\n"), fetchMaxChars)
}

var (
	reScriptStyle = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	reTag         = regexp.MustCompile(`(?s)<[^>]*>`)
	reSpace       = regexp.MustCompile(`\s+`)
)

func htmlToText(body []byte) string {
	s := reScriptStyle.ReplaceAll(body, []byte(" "))
	s = reTag.ReplaceAll(s, []byte(" "))
	s = reSpace.ReplaceAll(s, []byte(" "))
	return strings.TrimSpace(string(s))
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "…"
}
