package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebSearch_IsInfoAndReturnsText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "golang generics" {
			t.Errorf("query q = %q, want %q", r.URL.Query().Get("q"), "golang generics")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Heading":"Go","AbstractText":"Go supports generics since 1.18.","RelatedTopics":[{"Text":"Type parameters","FirstURL":"https://go.dev/x"},{"Text":"Constraints"}]}`))
	}))
	defer srv.Close()

	r := NewRegistry(nil)
	registerWeb(r, srv.Client(), srv.URL)

	if r.IsAction("web_search") {
		t.Fatalf("web_search must be an INFO tool, not an action")
	}
	res, err := r.Execute(context.Background(), "web_search", []byte(`{"query":"golang generics"}`))
	if err != nil || res.IsError {
		t.Fatalf("execute web_search: err=%v res=%+v", err, res)
	}
	if !strings.Contains(res.Content, "Go supports generics since 1.18.") {
		t.Fatalf("missing abstract; got: %q", res.Content)
	}
	if !strings.Contains(res.Content, "Type parameters") {
		t.Fatalf("missing related topic; got: %q", res.Content)
	}
}

func TestFetchURL_IsInfoAndStripsHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><style>.a{color:red}</style></head><body><h1>Title</h1><script>var secret=1;</script><p>Hello  world</p></body></html>`))
	}))
	defer srv.Close()

	r := NewRegistry(nil)
	registerWeb(r, srv.Client(), "https://unused.example")

	if r.IsAction("fetch_url") {
		t.Fatalf("fetch_url must be an INFO tool, not an action")
	}
	res, err := r.Execute(context.Background(), "fetch_url", []byte(`{"url":"`+srv.URL+`"}`))
	if err != nil || res.IsError {
		t.Fatalf("execute fetch_url: err=%v res=%+v", err, res)
	}
	if !strings.Contains(res.Content, "Title") || !strings.Contains(res.Content, "Hello world") {
		t.Fatalf("expected visible text, got: %q", res.Content)
	}
	if strings.Contains(res.Content, "secret") || strings.Contains(res.Content, "color:red") {
		t.Fatalf("script/style content leaked: %q", res.Content)
	}
	if strings.Contains(res.Content, "<") || strings.Contains(res.Content, ">") {
		t.Fatalf("html tags not stripped: %q", res.Content)
	}
}

func TestFetchURL_RejectsMissingURL(t *testing.T) {
	r := NewRegistry(nil)
	registerWeb(r, http.DefaultClient, "https://unused.example")
	res, err := r.Execute(context.Background(), "fetch_url", []byte(`{}`))
	if err == nil && !res.IsError {
		t.Fatalf("expected error for empty url, got %+v", res)
	}
}

func TestRegisterWeb_RegistersInfoTools(t *testing.T) {
	r := NewRegistry(nil)
	RegisterWeb(r)
	for _, name := range []string{"web_search", "fetch_url"} {
		if !r.Has(name) {
			t.Fatalf("RegisterWeb missing %q", name)
		}
		if r.IsAction(name) {
			t.Fatalf("%q must be INFO, not action", name)
		}
	}
}
