package node

import (
	"os"
	"path/filepath"
	"testing"
)

// ── Express basic CRUD ───────────────────────────────────────────────────────

func TestExpress_BasicCRUD(t *testing.T) {
	src := `
const express = require('express');
const app = express();

app.get('/api/users', getUsers);
app.post('/api/users', createUser);
app.get('/api/users/:id', getUser);
app.put('/api/users/:id', updateUser);
app.delete('/api/users/:id', deleteUser);
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d", len(endpoints))
	}

	tests := []struct {
		method string
		uri    string
		action string
	}{
		{"GET", "/api/users", "getUsers"},
		{"POST", "/api/users", "createUser"},
		{"GET", "/api/users/:id", "getUser"},
		{"PUT", "/api/users/:id", "updateUser"},
		{"DELETE", "/api/users/:id", "deleteUser"},
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
		if ep.Controller != "app" {
			t.Errorf("[%d] controller: got %q, want %q", i, ep.Controller, "app")
		}
	}
}

func TestExpress_AnonymousHandlers(t *testing.T) {
	src := `
const app = require('express')();

app.get('/api/items', (req, res) => {
    res.json(items);
});

app.post('/api/items', function(req, res) {
    res.status(201).json(item);
});
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].Action != "anonymous" {
		t.Errorf("arrow function: got %q, want anonymous", endpoints[0].Action)
	}
	if endpoints[1].Action != "anonymous" {
		t.Errorf("function expression: got %q, want anonymous", endpoints[1].Action)
	}
}

// ── Express Router with prefix ───────────────────────────────────────────────

func TestExpress_RouterWithPrefix(t *testing.T) {
	src := `
const express = require('express');
const app = express();
const usersRouter = express.Router();

usersRouter.get('/', getUsers);
usersRouter.get('/:id', getUser);
usersRouter.post('/', createUser);

app.use('/api/users', usersRouter);
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}

	expected := []struct {
		method string
		uri    string
	}{
		{"GET", "/api/users"},
		{"GET", "/api/users/:id"},
		{"POST", "/api/users"},
	}

	for i, tt := range expected {
		if endpoints[i].Method != tt.method {
			t.Errorf("[%d] method: got %q, want %q", i, endpoints[i].Method, tt.method)
		}
		if endpoints[i].URI != tt.uri {
			t.Errorf("[%d] URI: got %q, want %q", i, endpoints[i].URI, tt.uri)
		}
	}
}

func TestExpress_MultipleRouters(t *testing.T) {
	src := `
const usersRouter = require('express').Router();
const productsRouter = require('express').Router();

usersRouter.get('/', getUsers);
usersRouter.get('/:id', getUser);

productsRouter.get('/', getProducts);
productsRouter.post('/', createProduct);

app.use('/api/users', usersRouter);
app.use('/api/products', productsRouter);
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 4 {
		t.Fatalf("expected 4 endpoints, got %d", len(endpoints))
	}

	expected := []struct {
		method string
		uri    string
	}{
		{"GET", "/api/users"},
		{"GET", "/api/users/:id"},
		{"GET", "/api/products"},
		{"POST", "/api/products"},
	}

	for i, tt := range expected {
		if endpoints[i].Method != tt.method {
			t.Errorf("[%d] method: got %q, want %q", i, endpoints[i].Method, tt.method)
		}
		if endpoints[i].URI != tt.uri {
			t.Errorf("[%d] URI: got %q, want %q", i, endpoints[i].URI, tt.uri)
		}
	}
}

// ── Express middleware ───────────────────────────────────────────────────────

func TestExpress_Middleware(t *testing.T) {
	src := `
app.get('/api/admin', authenticate, (req, res) => {
    res.json({ admin: true });
});

app.post('/api/admin/action', authenticate, authorize, (req, res) => {
    res.json({ ok: true });
});
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if len(endpoints[0].Middleware) != 1 || endpoints[0].Middleware[0] != "authenticate" {
		t.Errorf("GET middleware: got %v, want [authenticate]", endpoints[0].Middleware)
	}

	if len(endpoints[1].Middleware) != 2 {
		t.Errorf("POST middleware count: got %d, want 2", len(endpoints[1].Middleware))
	} else {
		if endpoints[1].Middleware[0] != "authenticate" {
			t.Errorf("POST middleware[0]: got %q, want authenticate", endpoints[1].Middleware[0])
		}
		if endpoints[1].Middleware[1] != "authorize" {
			t.Errorf("POST middleware[1]: got %q, want authorize", endpoints[1].Middleware[1])
		}
	}
}

// ── Express route chaining ───────────────────────────────────────────────────

func TestExpress_RouteChain(t *testing.T) {
	src := `
app.route('/api/books')
    .get(getBooks)
    .post(createBook);

app.route('/api/books/:id')
    .get(getBook)
    .put(updateBook)
    .delete(deleteBook);
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d", len(endpoints))
	}

	expected := []struct {
		method string
		uri    string
	}{
		{"GET", "/api/books"},
		{"POST", "/api/books"},
		{"GET", "/api/books/:id"},
		{"PUT", "/api/books/:id"},
		{"DELETE", "/api/books/:id"},
	}

	for i, tt := range expected {
		if endpoints[i].Method != tt.method {
			t.Errorf("[%d] method: got %q, want %q", i, endpoints[i].Method, tt.method)
		}
		if endpoints[i].URI != tt.uri {
			t.Errorf("[%d] URI: got %q, want %q", i, endpoints[i].URI, tt.uri)
		}
	}
}

// ── Express route params extraction ──────────────────────────────────────────

func TestExpress_ExtractParams(t *testing.T) {
	src := `
app.get('/api/users/:id', getUser);
app.put('/api/users/:userId/posts/:postId', updatePost);
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if len(endpoints[0].StaticMeta.RequestParams) != 1 {
		t.Fatalf("[0] params: got %d, want 1", len(endpoints[0].StaticMeta.RequestParams))
	}
	if endpoints[0].StaticMeta.RequestParams[0].Name != "id" {
		t.Errorf("[0] param name: got %q, want id", endpoints[0].StaticMeta.RequestParams[0].Name)
	}

	if len(endpoints[1].StaticMeta.RequestParams) != 2 {
		t.Fatalf("[1] params: got %d, want 2", len(endpoints[1].StaticMeta.RequestParams))
	}
}

// ── Express edge cases ───────────────────────────────────────────────────────

func TestExpress_AllMethod(t *testing.T) {
	src := `app.all('/api/cors-check', handleCORS);`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].Method != "ALL" {
		t.Errorf("method: got %q, want ALL", endpoints[0].Method)
	}
}

func TestExpress_PatchOptionsHead(t *testing.T) {
	src := `
app.patch('/api/users/:id', patchUser);
app.options('/api/users', handleOptions);
app.head('/api/health', healthHead);
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].Method != "PATCH" {
		t.Errorf("[0] method: got %q, want PATCH", endpoints[0].Method)
	}
	if endpoints[1].Method != "OPTIONS" {
		t.Errorf("[1] method: got %q, want OPTIONS", endpoints[1].Method)
	}
	if endpoints[2].Method != "HEAD" {
		t.Errorf("[2] method: got %q, want HEAD", endpoints[2].Method)
	}
}

func TestExpress_QuoteStyles(t *testing.T) {
	src := `
app.get('/api/single', handler1);
app.get("/api/double", handler2);
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].URI != "/api/single" {
		t.Errorf("[0] URI: got %q, want /api/single", endpoints[0].URI)
	}
	if endpoints[1].URI != "/api/double" {
		t.Errorf("[1] URI: got %q, want /api/double", endpoints[1].URI)
	}
}

func TestExpress_ControllerStyleHandler(t *testing.T) {
	src := `
app.get('/api/users', usersController.getAll);
app.post('/api/users', usersController.create);
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].Action != "usersController.getAll" {
		t.Errorf("[0] action: got %q, want usersController.getAll", endpoints[0].Action)
	}
	if endpoints[1].Action != "usersController.create" {
		t.Errorf("[1] action: got %q, want usersController.create", endpoints[1].Action)
	}
}

func TestExpress_RouterWithoutMount(t *testing.T) {
	src := `
const router = require('express').Router();
router.get('/users', getUsers);
router.post('/users', createUser);
router.get('/users/:id', getUser);
module.exports = router;
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}

	for _, ep := range endpoints {
		if ep.Controller != "router" {
			t.Errorf("controller: got %q, want router", ep.Controller)
		}
	}
}

func TestExpress_MultilineHandler(t *testing.T) {
	src := `
app.get('/api/users',
    authenticate,
    (req, res) => {
        const users = db.getAll();
        res.json(users);
    }
);
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if endpoints[0].Method != "GET" || endpoints[0].URI != "/api/users" {
		t.Errorf("got %s %s, want GET /api/users", endpoints[0].Method, endpoints[0].URI)
	}
}

func TestExpress_PluginPattern(t *testing.T) {
	src := `
module.exports = function(app) {
    app.get('/api/auth/login', login);
    app.post('/api/auth/register', register);
    app.post('/api/auth/logout', logout);
};
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}
}

func TestExpress_NestedRouterPrefix(t *testing.T) {
	src := `
const express = require('express');
const apiRouter = express.Router();
const v1Router = express.Router();

v1Router.get('/users', getUsers);
v1Router.get('/items', getItems);

apiRouter.use('/v1', v1Router);
app.use('/api', apiRouter);
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].URI != "/v1/users" {
		t.Errorf("[0] URI: got %q, want /v1/users", endpoints[0].URI)
	}
	if endpoints[1].URI != "/v1/items" {
		t.Errorf("[1] URI: got %q, want /v1/items", endpoints[1].URI)
	}
}

func TestExpress_CommentsStripped(t *testing.T) {
	src := `
// app.get('/api/commented-out', handler);
/*
app.get('/api/block-commented', handler);
*/
app.get('/api/real-route', handler);
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].URI != "/api/real-route" {
		t.Errorf("URI: got %q, want /api/real-route", endpoints[0].URI)
	}
}

func TestExpress_TestCodeSkipped(t *testing.T) {
	src := `
const app = express();
app.get('/api/real', handler);

describe('API tests', () => {
    test.get('/api/fake');
    it.get('/api/also-fake');
});
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint (test code skipped), got %d", len(endpoints))
	}
	if endpoints[0].URI != "/api/real" {
		t.Errorf("URI: got %q, want /api/real", endpoints[0].URI)
	}
}

func TestExpress_NoRoutesInFile(t *testing.T) {
	src := `
const express = require('express');
const helper = () => { console.log('no routes here'); };
module.exports = helper;
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 0 {
		t.Fatalf("expected 0 endpoints, got %d", len(endpoints))
	}
}

func TestExpress_TypeScriptAnnotations(t *testing.T) {
	src := `
import express, { Request, Response } from 'express';

const app = express();

app.get('/api/users', (req: Request, res: Response) => {
    const users: User[] = await getUsers();
    res.json(users);
});

app.post('/api/users', (req: Request, res: Response) => {
    const user: User = req.body;
    res.status(201).json(user);
});
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].Method != "GET" || endpoints[0].URI != "/api/users" {
		t.Errorf("[0]: got %s %s", endpoints[0].Method, endpoints[0].URI)
	}
	if endpoints[1].Method != "POST" || endpoints[1].URI != "/api/users" {
		t.Errorf("[1]: got %s %s", endpoints[1].Method, endpoints[1].URI)
	}
}

func TestExpress_BuildRouterPrefixes(t *testing.T) {
	src := `
const usersRouter = express.Router();
const itemsRouter = express.Router();

app.use('/api/users', usersRouter);
app.use('/api/items', itemsRouter);
`
	prefixes := buildExpressPrefixes(src)

	if prefixes["usersRouter"] != "/api/users" {
		t.Errorf("usersRouter prefix: got %q, want /api/users", prefixes["usersRouter"])
	}
	if prefixes["itemsRouter"] != "/api/items" {
		t.Errorf("itemsRouter prefix: got %q, want /api/items", prefixes["itemsRouter"])
	}
}

// ── ParseSections integration ────────────────────────────────────────────────

func TestExpress_ParseSections_Language(t *testing.T) {
	dir := t.TempDir()

	jsFile := filepath.Join(dir, "routes.js")
	os.WriteFile(jsFile, []byte(`app.get('/api/users', getUsers);`), 0644)

	tsFile := filepath.Join(dir, "routes.ts")
	os.WriteFile(tsFile, []byte(`app.get('/api/items', getItems);`), 0644)

	p := NewExpress(dir)
	sections, err := p.ParseSections([]string{jsFile, tsFile})
	if err != nil {
		t.Fatalf("ParseSections: %v", err)
	}

	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}

	if sections[0].Endpoints[0].Language != "javascript" {
		t.Errorf(".js language: got %q, want javascript", sections[0].Endpoints[0].Language)
	}
	if sections[1].Endpoints[0].Language != "typescript" {
		t.Errorf(".ts language: got %q, want typescript", sections[1].Endpoints[0].Language)
	}
}

func TestExpress_ParseSections_SkipsEmptyFiles(t *testing.T) {
	dir := t.TempDir()

	routeFile := filepath.Join(dir, "routes.js")
	os.WriteFile(routeFile, []byte(`app.get('/api/users', getUsers);`), 0644)

	emptyFile := filepath.Join(dir, "utils.js")
	os.WriteFile(emptyFile, []byte(`const helper = () => {};`), 0644)

	p := NewExpress(dir)
	sections, err := p.ParseSections([]string{routeFile, emptyFile})
	if err != nil {
		t.Fatalf("ParseSections: %v", err)
	}

	if len(sections) != 1 {
		t.Fatalf("expected 1 section (utils skipped), got %d", len(sections))
	}
}

func TestExpress_DottedHandler(t *testing.T) {
	src := `
const userCtrl = require('./controllers/users');
app.get('/api/users', userCtrl.list);
app.get('/api/users/:id', userCtrl.show);
app.post('/api/users', userCtrl.create);
app.put('/api/users/:id', userCtrl.update);
app.delete('/api/users/:id', userCtrl.destroy);
`
	p := NewExpress(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d", len(endpoints))
	}

	expectedActions := []string{
		"userCtrl.list", "userCtrl.show", "userCtrl.create",
		"userCtrl.update", "userCtrl.destroy",
	}
	for i, want := range expectedActions {
		if endpoints[i].Action != want {
			t.Errorf("[%d] action: got %q, want %q", i, endpoints[i].Action, want)
		}
	}
}
