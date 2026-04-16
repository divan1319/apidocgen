package golang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divan1319/apidocgen/internal/parser"
)

func TestGoParserRegistration(t *testing.T) {
	names := parser.Names()
	expected := []string{"gohttp", "gomux", "fiber", "echo"}

	for _, name := range expected {
		found := false
		for _, n := range names {
			if n == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("parser %q not registered (registered: %v)", name, names)
		}
	}
}

func TestGoParser_Get(t *testing.T) {
	tests := []string{"gohttp", "gomux", "fiber", "echo"}

	for _, name := range tests {
		p, err := parser.Get(name, ".")
		if err != nil {
			t.Errorf("Get(%q): %v", name, err)
			continue
		}
		if p.Language() != name {
			t.Errorf("Language(): got %q, want %q", p.Language(), name)
		}
	}
}

func TestResolveIncludes_Dir(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "routes.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "routes_test.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0644)

	os.MkdirAll(filepath.Join(dir, "vendor", "lib"), 0755)
	os.WriteFile(filepath.Join(dir, "vendor", "lib", "dep.go"), []byte("package lib"), 0644)

	bp := &baseParser{projectRoot: dir}
	files, err := bp.ResolveIncludes([]string{dir})
	if err != nil {
		t.Fatalf("ResolveIncludes: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 .go files (excluding _test.go and vendor), got %d: %v", len(files), files)
	}

	for _, f := range files {
		ext := filepath.Ext(f)
		if ext != ".go" {
			t.Errorf("unexpected file: %s", f)
		}
	}
}

func TestResolveIncludes_SingleFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "routes.go")
	os.WriteFile(file, []byte("package main"), 0644)

	bp := &baseParser{projectRoot: dir}
	files, err := bp.ResolveIncludes([]string{file})
	if err != nil {
		t.Fatalf("ResolveIncludes: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}

func TestExtractGoPathParams(t *testing.T) {
	tests := []struct {
		path   string
		expect []string
	}{
		{"/api/users/{id}", []string{"id"}},
		{"/api/users/:id", []string{"id"}},
		{"/api/users/{userID}/posts/{postID}", []string{"userID", "postID"}},
		{"/api/files/{path...}", []string{"path"}},
		{"/api/users", nil},
	}

	for _, tt := range tests {
		params := extractGoPathParams(tt.path)
		if tt.expect == nil {
			if params != nil {
				t.Errorf("path %q: expected nil params, got %v", tt.path, params)
			}
			continue
		}
		if len(params) != len(tt.expect) {
			t.Errorf("path %q: expected %d params, got %d", tt.path, len(tt.expect), len(params))
			continue
		}
		for i, name := range tt.expect {
			if params[i].Name != name {
				t.Errorf("path %q param[%d]: got %q, want %q", tt.path, i, params[i].Name, name)
			}
		}
	}
}

func TestInferVersion(t *testing.T) {
	tests := []struct {
		path    string
		version string
	}{
		{"/project/api/v1/routes.go", "v1"},
		{"/project/api/v2/routes.go", "v2"},
		{"/project/routes.go", ""},
	}

	for _, tt := range tests {
		got := inferVersion(tt.path)
		if got != tt.version {
			t.Errorf("inferVersion(%q): got %q, want %q", tt.path, got, tt.version)
		}
	}
}
