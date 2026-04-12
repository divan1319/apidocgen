package node

import (
	"testing"
)

// ── Native HTTP if/else ──────────────────────────────────────────────────────

func TestNativeHTTP_IfElse(t *testing.T) {
	src := `
const http = require('http');

const server = http.createServer((req, res) => {
    if (req.method === 'GET' && req.url === '/api/users') {
        res.writeHead(200);
        res.end(JSON.stringify(users));
    } else if (req.method === 'POST' && req.url === '/api/users') {
        res.writeHead(201);
        res.end();
    } else if (req.method === 'GET' && req.url === '/api/users/profile') {
        res.writeHead(200);
        res.end();
    }
});
`
	p := NewNativeHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}

	tests := []struct {
		method string
		uri    string
	}{
		{"GET", "/api/users"},
		{"POST", "/api/users"},
		{"GET", "/api/users/profile"},
	}

	for i, tt := range tests {
		if endpoints[i].Method != tt.method {
			t.Errorf("[%d] method: got %q, want %q", i, endpoints[i].Method, tt.method)
		}
		if endpoints[i].URI != tt.uri {
			t.Errorf("[%d] URI: got %q, want %q", i, endpoints[i].URI, tt.uri)
		}
		if endpoints[i].Controller != "http" {
			t.Errorf("[%d] controller: got %q, want http", i, endpoints[i].Controller)
		}
	}
}

// ── Reverse order: req.url first ─────────────────────────────────────────────

func TestNativeHTTP_ReverseOrder(t *testing.T) {
	src := `
if (req.url === '/api/items' && req.method === 'GET') {
    handleGetItems(req, res);
}
if (req.url === '/api/items' && req.method === 'POST') {
    handleCreateItem(req, res);
}
`
	p := NewNativeHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].Method != "GET" || endpoints[0].URI != "/api/items" {
		t.Errorf("[0]: got %s %s, want GET /api/items", endpoints[0].Method, endpoints[0].URI)
	}
	if endpoints[1].Method != "POST" || endpoints[1].URI != "/api/items" {
		t.Errorf("[1]: got %s %s, want POST /api/items", endpoints[1].Method, endpoints[1].URI)
	}
}

// ── Loose equality (==) ──────────────────────────────────────────────────────

func TestNativeHTTP_LooseEquality(t *testing.T) {
	src := `
if (req.method == 'GET' && req.url == '/api/data') {
    handleData(req, res);
}
`
	p := NewNativeHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if endpoints[0].Method != "GET" || endpoints[0].URI != "/api/data" {
		t.Errorf("got %s %s, want GET /api/data", endpoints[0].Method, endpoints[0].URI)
	}
}

// ── switch/case on URL ───────────────────────────────────────────────────────

func TestNativeHTTP_SwitchUrl(t *testing.T) {
	src := `
const server = http.createServer((req, res) => {
    switch (req.url) {
        case '/api/users':
            handleUsers(req, res);
            break;
        case '/api/items':
            handleItems(req, res);
            break;
    }
});
`
	p := NewNativeHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].URI != "/api/users" {
		t.Errorf("[0] URI: got %q, want /api/users", endpoints[0].URI)
	}
	if endpoints[1].URI != "/api/items" {
		t.Errorf("[1] URI: got %q, want /api/items", endpoints[1].URI)
	}
}

// ── switch/case on method with URL context ───────────────────────────────────

func TestNativeHTTP_SwitchMethodWithUrlContext(t *testing.T) {
	src := `
const server = http.createServer((req, res) => {
    if (req.url === '/api/users') {
        switch (req.method) {
            case 'GET':
                getUsers(req, res);
                break;
            case 'POST':
                createUser(req, res);
                break;
        }
    }
});
`
	p := NewNativeHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	methods := map[string]bool{}
	for _, ep := range endpoints {
		if ep.URI != "/api/users" {
			t.Errorf("URI: got %q, want /api/users", ep.URI)
		}
		methods[ep.Method] = true
	}
	if !methods["GET"] {
		t.Error("missing GET method")
	}
	if !methods["POST"] {
		t.Error("missing POST method")
	}
}

// ── No routes in file ────────────────────────────────────────────────────────

func TestNativeHTTP_NoRoutes(t *testing.T) {
	src := `
const http = require('http');
const server = http.createServer((req, res) => {
    res.end('hello');
});
`
	p := NewNativeHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 0 {
		t.Fatalf("expected 0 endpoints, got %d", len(endpoints))
	}
}

// ── Comments stripped ────────────────────────────────────────────────────────

func TestNativeHTTP_CommentsStripped(t *testing.T) {
	src := `
// if (req.method === 'GET' && req.url === '/api/fake') {}
if (req.method === 'GET' && req.url === '/api/real') {
    handle(req, res);
}
`
	p := NewNativeHTTP(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].URI != "/api/real" {
		t.Errorf("URI: got %q, want /api/real", endpoints[0].URI)
	}
}
