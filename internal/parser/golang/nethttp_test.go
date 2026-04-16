package golang

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNetHTTP_HandleFunc(t *testing.T) {
	src := `
package main

import "net/http"

func main() {
	http.HandleFunc("/api/users", handleUsers)
	http.HandleFunc("/api/items", handleItems)
	http.HandleFunc("/health", healthCheck)
}
`
	p := NewNetHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}

	tests := []struct {
		method string
		uri    string
		action string
	}{
		{"ALL", "/api/users", "handleUsers"},
		{"ALL", "/api/items", "handleItems"},
		{"ALL", "/health", "healthCheck"},
	}

	for i, tt := range tests {
		ep := endpoints[i]
		if ep.Method != tt.method {
			t.Errorf("[%d] method: got %q, want %q", i, ep.Method, tt.method)
		}
		if ep.URI != tt.uri {
			t.Errorf("[%d] URI: got %q, want %q", i, ep.URI, tt.uri)
		}
		if ep.Action != tt.action {
			t.Errorf("[%d] action: got %q, want %q", i, ep.Action, tt.action)
		}
	}
}

func TestNetHTTP_ServeMux(t *testing.T) {
	src := `
package main

import "net/http"

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/users", handleUsers)
	mux.Handle("/api/static", http.FileServer)
}
`
	p := NewNetHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].Controller != "mux" {
		t.Errorf("[0] controller: got %q, want mux", endpoints[0].Controller)
	}
	if endpoints[0].URI != "/api/users" {
		t.Errorf("[0] URI: got %q, want /api/users", endpoints[0].URI)
	}
	if endpoints[1].URI != "/api/static" {
		t.Errorf("[1] URI: got %q, want /api/static", endpoints[1].URI)
	}
}

func TestNetHTTP_Go122Patterns(t *testing.T) {
	src := `
package main

import "net/http"

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/users", listUsers)
	mux.HandleFunc("POST /api/users", createUser)
	mux.HandleFunc("GET /api/users/{id}", getUser)
	mux.HandleFunc("PUT /api/users/{id}", updateUser)
	mux.HandleFunc("DELETE /api/users/{id}", deleteUser)
}
`
	p := NewNetHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d", len(endpoints))
	}

	tests := []struct {
		method string
		uri    string
		action string
	}{
		{"GET", "/api/users", "listUsers"},
		{"POST", "/api/users", "createUser"},
		{"GET", "/api/users/{id}", "getUser"},
		{"PUT", "/api/users/{id}", "updateUser"},
		{"DELETE", "/api/users/{id}", "deleteUser"},
	}

	for i, tt := range tests {
		ep := endpoints[i]
		if ep.Method != tt.method {
			t.Errorf("[%d] method: got %q, want %q", i, ep.Method, tt.method)
		}
		if ep.URI != tt.uri {
			t.Errorf("[%d] URI: got %q, want %q", i, ep.URI, tt.uri)
		}
		if ep.Action != tt.action {
			t.Errorf("[%d] action: got %q, want %q", i, ep.Action, tt.action)
		}
	}
}

func TestNetHTTP_Go122PathParams(t *testing.T) {
	src := `
package main

func main() {
	mux.HandleFunc("GET /api/users/{userID}/posts/{postID}", getPost)
}
`
	p := NewNetHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	params := endpoints[0].StaticMeta.RequestParams
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	if params[0].Name != "userID" {
		t.Errorf("param[0]: got %q, want userID", params[0].Name)
	}
	if params[1].Name != "postID" {
		t.Errorf("param[1]: got %q, want postID", params[1].Name)
	}
}

func TestNetHTTP_WildcardParam(t *testing.T) {
	src := `
package main

func main() {
	mux.HandleFunc("GET /files/{path...}", serveFile)
}
`
	p := NewNetHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	params := endpoints[0].StaticMeta.RequestParams
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(params))
	}
	if params[0].Name != "path" {
		t.Errorf("param: got %q, want path", params[0].Name)
	}
}

func TestNetHTTP_CommentsStripped(t *testing.T) {
	src := `
package main

// http.HandleFunc("/commented-out", handler)
/*
http.HandleFunc("/block-commented", handler)
*/
http.HandleFunc("/real", realHandler)
`
	p := NewNetHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].URI != "/real" {
		t.Errorf("URI: got %q, want /real", endpoints[0].URI)
	}
}

func TestNetHTTP_MixedPatterns(t *testing.T) {
	src := `
package main

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/legacy", legacyHandler)
	mux.HandleFunc("GET /api/v2/users", listUsersV2)
	mux.Handle("/static", staticHandler)
}
`
	p := NewNetHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].Method != "ALL" {
		t.Errorf("[0] method: got %q, want ALL", endpoints[0].Method)
	}
	if endpoints[1].Method != "GET" {
		t.Errorf("[1] method: got %q, want GET", endpoints[1].Method)
	}
	if endpoints[2].Method != "ALL" {
		t.Errorf("[2] method: got %q, want ALL", endpoints[2].Method)
	}
}

func TestNetHTTP_NoRoutes(t *testing.T) {
	src := `
package utils

func helper() string {
	return "no routes here"
}
`
	p := NewNetHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 0 {
		t.Fatalf("expected 0 endpoints, got %d", len(endpoints))
	}
}

func TestNetHTTP_ParseSections_Language(t *testing.T) {
	dir := t.TempDir()

	goFile := filepath.Join(dir, "routes.go")
	os.WriteFile(goFile, []byte(`
package main
http.HandleFunc("/api/users", handleUsers)
`), 0644)

	p := NewNetHTTP(dir)
	sections, err := p.ParseSections([]string{goFile})
	if err != nil {
		t.Fatalf("ParseSections: %v", err)
	}

	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}

	if sections[0].Endpoints[0].Language != "go" {
		t.Errorf("language: got %q, want go", sections[0].Endpoints[0].Language)
	}
}
