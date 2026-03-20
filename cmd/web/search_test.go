package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

var param = &SearchParam{Query: "golang"}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestBuildRequest(t *testing.T) {
	req, err := buildRequest(param, defaultClientOption)
	if err != nil {
		t.Fatal(err)
	}

	url := req.URL.String()
	if url != `https://html.duckduckgo.com/html?api=%2Fd.js&dc=1&o=json&q=golang&s=0&v=1` {
		t.Fatal(url)
	}
}

func TestSearchWithOptionMockedServer(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Referrer"); got != "https://ref.example" {
			t.Fatalf("Referrer header = %q", got)
		}
		if got := r.Header.Get("User-Agent"); got != "ua-test" {
			t.Fatalf("User-Agent header = %q", got)
		}

		_, _ = io.WriteString(w, `
<html><body>
  <div class="result">
    <div class="result__title"><a>Golang Official</a></div>
    <a class="result__url" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fgo.dev%2F&amp;rut=x"></a>
    <div class="result__snippet">Go <b>language</b> site</div>
  </div>
</body></html>`)
	}))
	defer server.Close()

	origBuildSearchURL := buildSearchURL
	origBuildHTTPClient := buildHTTPClient
	buildSearchURL = func(param *SearchParam) (*url.URL, error) {
		return url.Parse(server.URL)
	}
	buildHTTPClient = func(opt *ClientOption) *http.Client {
		return server.Client()
	}
	t.Cleanup(func() {
		buildSearchURL = origBuildSearchURL
		buildHTTPClient = origBuildHTTPClient
	})

	result, err := SearchWithOption(param, &ClientOption{
		Referrer:  "https://ref.example",
		UserAgent: "ua-test",
		Timeout:   time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(*result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(*result))
	}
	if got := (*result)[0].Title; got != "Golang Official" {
		t.Fatalf("title = %q", got)
	}
	if got := (*result)[0].Link; got != "https://go.dev/" {
		t.Fatalf("link = %q", got)
	}
	if got := strings.TrimSpace((*result)[0].Snippet); got != "Go language site" {
		t.Fatalf("snippet = %q", got)
	}
}

func TestSearchWithOptionRequestError(t *testing.T) {
	origBuildHTTPClient := buildHTTPClient
	buildHTTPClient = func(opt *ClientOption) *http.Client {
		return &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("boom")
		})}
	}
	t.Cleanup(func() {
		buildHTTPClient = origBuildHTTPClient
	})

	_, err := SearchWithOption(param, defaultClientOption)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to send request") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewSearchParamEmpty(t *testing.T) {
	_, err := NewSearchParam("   ")
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestExtractLinkMissingUddg(t *testing.T) {
	if got := extractLink("//duckduckgo.com/l/?rut=something"); got != "" {
		t.Fatalf("extractLink() = %q, want empty", got)
	}
}
