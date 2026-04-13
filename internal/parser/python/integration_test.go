package python

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divan1319/apidocgen/internal/parser"
)

func TestParserRegistered(t *testing.T) {
	names := parser.Names()
	foundF, foundFl := false, false
	for _, n := range names {
		if n == "fastapi" {
			foundF = true
		}
		if n == "flask" {
			foundFl = true
		}
	}
	if !foundF || !foundFl {
		t.Fatalf("registrados: %v", names)
	}
	_, err := parser.Get("fastapi", ".")
	if err != nil {
		t.Fatal(err)
	}
	_, err = parser.Get("flask", ".")
	if err != nil {
		t.Fatal(err)
	}
}

func TestFastAPI_ResolveIncludes_Directory(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "pkg")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.py"), []byte("app=None\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "skip.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	p := NewFastAPI(root)
	files, err := p.ResolveIncludes([]string{sub})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "a.py" {
		t.Fatalf("files=%v", files)
	}
}

func TestFastAPI_TraceAndOptions(t *testing.T) {
	src := `
from fastapi import FastAPI
app = FastAPI()
@app.trace("/t")
def t():
    pass
@app.options("/o")
def o():
    pass
`
	p := NewFastAPI(".")
	eps := p.parseFile(src)
	if len(eps) != 2 {
		t.Fatalf("hay %d", len(eps))
	}
	methods := map[string]string{eps[0].Method: eps[0].URI, eps[1].Method: eps[1].URI}
	if methods["TRACE"] != "/t" || methods["OPTIONS"] != "/o" {
		t.Fatalf("%+v", eps)
	}
}

func TestFlask_OptionsHead(t *testing.T) {
	src := `
from flask import Flask
app = Flask(__name__)
@app.options("/cors")
def cors():
    pass
@app.head("/meta")
def meta():
    pass
`
	p := NewFlask(".")
	eps := p.parseFile(src)
	if len(eps) != 2 {
		t.Fatalf("hay %d", len(eps))
	}
}
