package python

import (
	"testing"
)

func TestExtractPythonPathParams_FastAPIStyle(t *testing.T) {
	params := extractPythonPathParams("/items/{item_id}")
	if len(params) != 1 || params[0].Name != "item_id" {
		t.Fatalf("got %+v", params)
	}
}

func TestExtractPythonPathParams_FastAPITyped(t *testing.T) {
	params := extractPythonPathParams("/items/{item_id:int}")
	if len(params) != 1 || params[0].Name != "item_id" {
		t.Fatalf("got %+v", params)
	}
}

func TestExtractPythonPathParams_FlaskStyle(t *testing.T) {
	params := extractPythonPathParams("/users/<int:user_id>")
	if len(params) != 1 || params[0].Name != "user_id" {
		t.Fatalf("got %+v", params)
	}
}

func TestExtractPythonPathParams_Mixed(t *testing.T) {
	params := extractPythonPathParams("/a/{x}/b/<slug>")
	names := map[string]bool{}
	for _, p := range params {
		names[p.Name] = true
	}
	if !names["x"] || !names["slug"] {
		t.Fatalf("got %+v", params)
	}
}

func TestJoinPath(t *testing.T) {
	if joinPath("api", "v1") != "/api/v1" {
		t.Fatalf("got %q", joinPath("api", "v1"))
	}
	if joinPath("", "users") != "/users" {
		t.Fatalf("got %q", joinPath("", "users"))
	}
}

func TestStripPythonComments(t *testing.T) {
	src := "x = 1  # c\ny = 2\n"
	out := stripPythonComments(src)
	if out != "x = 1  \ny = 2\n" {
		t.Fatalf("got %q", out)
	}
}
