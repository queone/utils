package main

import (
	"testing"
)

var param = &SearchParam{
	Query: "golang",
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
func TestSearch(t *testing.T) {
	_, err := Search(param, 10)
	if err != nil {
		t.Fatal(err)
	}
}
