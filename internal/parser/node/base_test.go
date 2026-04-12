package node

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divan1319/apidocgen/pkg/models"
)

// ── Utility function tests ───────────────────────────────────────────────────

func TestJoinPath(t *testing.T) {
	tests := []struct {
		a, b string
		want string
	}{
		{"", "", "/"},
		{"", "/users", "/users"},
		{"/api", "", "/api"},
		{"/api", "/users", "/api/users"},
		{"/api/", "/users/", "/api/users"},
		{"api", "users", "/api/users"},
		{"/api/v1", "/items/:id", "/api/v1/items/:id"},
	}

	for _, tt := range tests {
		got := joinPath(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("joinPath(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestDeriveActionName(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   string
	}{
		{"GET", "/api/users", "GET_ApiUsers"},
		{"POST", "/api/users", "POST_ApiUsers"},
		{"GET", "/", "GET_Root"},
		{"DELETE", "/api/users/:id", "DELETE_ApiUsers"},
		{"GET", "/api/v1/items", "GET_ApiV1Items"},
	}

	for _, tt := range tests {
		got := deriveActionName(tt.method, tt.path)
		if got != tt.want {
			t.Errorf("deriveActionName(%q, %q) = %q, want %q", tt.method, tt.path, got, tt.want)
		}
	}
}

func TestStripJSComments(t *testing.T) {
	src := `
// This is a comment
app.get('/api/users', getUsers); // inline comment
/* block comment
   spanning lines */
app.post('/api/items', createItem);
`
	result := stripJSComments(src)
	if strings.Contains(result, "This is a comment") {
		t.Error("line comment not stripped")
	}
	if strings.Contains(result, "inline comment") {
		t.Error("inline comment not stripped")
	}
	if strings.Contains(result, "block comment") {
		t.Error("block comment not stripped")
	}
	if !strings.Contains(result, "app.get") {
		t.Error("app.get should be preserved")
	}
	if !strings.Contains(result, "app.post") {
		t.Error("app.post should be preserved")
	}
}

func TestIsTestVariable(t *testing.T) {
	if !isTestVariable("test") {
		t.Error("test should be a test variable")
	}
	if !isTestVariable("supertest") {
		t.Error("supertest should be a test variable")
	}
	if isTestVariable("app") {
		t.Error("app should NOT be a test variable")
	}
	if isTestVariable("router") {
		t.Error("router should NOT be a test variable")
	}
	if isTestVariable("fastify") {
		t.Error("fastify should NOT be a test variable")
	}
}

func TestInferVersion(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/project/routes/v1/users.js", "v1"},
		{"/project/routes/v2/items.ts", "v2"},
		{"/project/routes/users.js", ""},
		{"/project/api/v3/routes/products.js", "v3"},
	}

	for _, tt := range tests {
		got := inferVersion(tt.path)
		if got != tt.want {
			t.Errorf("inferVersion(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestFileLanguage(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"routes/users.js", "javascript"},
		{"routes/users.ts", "typescript"},
		{"routes/users.mjs", "javascript"},
		{"routes/users.mts", "typescript"},
		{"routes/users.cjs", "javascript"},
		{"routes/users.cts", "typescript"},
	}

	for _, tt := range tests {
		got := fileLanguage(tt.path)
		if got != tt.want {
			t.Errorf("fileLanguage(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestExtractRouteParams(t *testing.T) {
	tests := []struct {
		path   string
		params []string
	}{
		{"/api/users", nil},
		{"/api/users/:id", []string{"id"}},
		{"/api/users/:userId/posts/:postId", []string{"userId", "postId"}},
		{"/:org/:repo/issues/:number", []string{"org", "repo", "number"}},
	}

	for _, tt := range tests {
		params := extractRouteParams(tt.path)
		if tt.params == nil {
			if params != nil {
				t.Errorf("extractRouteParams(%q): expected nil, got %v", tt.path, params)
			}
			continue
		}
		if len(params) != len(tt.params) {
			t.Errorf("extractRouteParams(%q): got %d params, want %d", tt.path, len(params), len(tt.params))
			continue
		}
		for i, name := range tt.params {
			if params[i].Name != name {
				t.Errorf("extractRouteParams(%q)[%d]: got %q, want %q", tt.path, i, params[i].Name, name)
			}
			if !params[i].Required {
				t.Errorf("extractRouteParams(%q)[%d]: should be required", tt.path, i)
			}
		}
	}
}

func TestExtractHandlerName(t *testing.T) {
	tests := []struct {
		stmt string
		want string
	}{
		{`app.get('/users', getUsers);`, "getUsers"},
		{`app.get('/users', controller.getAll);`, "controller.getAll"},
		{`app.get('/users', (req, res) => { res.json(); });`, "anonymous"},
		{`app.get('/users', function(req, res) { res.json(); });`, "anonymous"},
		{`app.get('/users', async (req, res) => { res.json(); });`, "anonymous"},
	}

	for _, tt := range tests {
		got := extractHandlerName(tt.stmt)
		if got != tt.want {
			t.Errorf("extractHandlerName(%q) = %q, want %q", tt.stmt, got, tt.want)
		}
	}
}

func TestSplitTopLevel(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{` auth, handler)`, 2},
		{` auth, validate, handler)`, 3},
		{` [auth, validate], handler)`, 2},
		{` (req, res) => {})`, 1},
		{` handler)`, 1},
	}

	for _, tt := range tests {
		parts := splitTopLevel(tt.input, ',')
		if len(parts) != tt.want {
			t.Errorf("splitTopLevel(%q): got %d parts, want %d — parts: %v",
				tt.input, len(parts), tt.want, parts)
		}
	}
}

func TestDeduplication(t *testing.T) {
	endpoints := deduplicateEndpoints([]models.Endpoint{
		{Method: "GET", URI: "/api/users"},
		{Method: "GET", URI: "/api/users"},
		{Method: "POST", URI: "/api/users"},
	})

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 unique endpoints, got %d", len(endpoints))
	}
}

// ── ResolveIncludes ──────────────────────────────────────────────────────────

func TestResolveIncludes_Directory(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "routes"), 0755)
	os.WriteFile(filepath.Join(dir, "routes", "users.js"), []byte("// routes"), 0644)
	os.WriteFile(filepath.Join(dir, "routes", "items.ts"), []byte("// routes"), 0644)
	os.WriteFile(filepath.Join(dir, "routes", "readme.txt"), []byte("not js"), 0644)

	os.MkdirAll(filepath.Join(dir, "routes", "node_modules"), 0755)
	os.WriteFile(filepath.Join(dir, "routes", "node_modules", "dep.js"), []byte("// dep"), 0644)

	p := NewExpress(dir)
	files, err := p.ResolveIncludes([]string{filepath.Join(dir, "routes")})
	if err != nil {
		t.Fatalf("ResolveIncludes: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files (js + ts), got %d: %v", len(files), files)
	}
}

func TestResolveIncludes_SingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.js")
	os.WriteFile(path, []byte("// routes"), 0644)

	p := NewFastify(dir)
	files, err := p.ResolveIncludes([]string{path})
	if err != nil {
		t.Fatalf("ResolveIncludes: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}
