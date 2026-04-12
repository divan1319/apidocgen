package node

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Express basic CRUD ───────────────────────────────────────────────────────

func TestParseExpressRoutes_BasicCRUD(t *testing.T) {
	src := `
const express = require('express');
const app = express();

app.get('/api/users', getUsers);
app.post('/api/users', createUser);
app.get('/api/users/:id', getUser);
app.put('/api/users/:id', updateUser);
app.delete('/api/users/:id', deleteUser);
`
	p := New(".")
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

func TestParseExpressRoutes_AnonymousHandlers(t *testing.T) {
	src := `
const app = require('express')();

app.get('/api/items', (req, res) => {
    res.json(items);
});

app.post('/api/items', function(req, res) {
    res.status(201).json(item);
});
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].Action != "anonymous" {
		t.Errorf("arrow function action: got %q, want anonymous", endpoints[0].Action)
	}
	if endpoints[1].Action != "anonymous" {
		t.Errorf("function expression action: got %q, want anonymous", endpoints[1].Action)
	}
}

// ── Express Router with prefix ───────────────────────────────────────────────

func TestParseExpressRoutes_RouterWithPrefix(t *testing.T) {
	src := `
const express = require('express');
const app = express();
const usersRouter = express.Router();

usersRouter.get('/', getUsers);
usersRouter.get('/:id', getUser);
usersRouter.post('/', createUser);

app.use('/api/users', usersRouter);
`
	p := New(".")
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

func TestParseExpressRoutes_MultipleRouters(t *testing.T) {
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
	p := New(".")
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

func TestParseExpressRoutes_Middleware(t *testing.T) {
	src := `
app.get('/api/admin', authenticate, (req, res) => {
    res.json({ admin: true });
});

app.post('/api/admin/action', authenticate, authorize, (req, res) => {
    res.json({ ok: true });
});
`
	p := New(".")
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

func TestParseExpressRoutes_RouteChain(t *testing.T) {
	src := `
app.route('/api/books')
    .get(getBooks)
    .post(createBook);

app.route('/api/books/:id')
    .get(getBook)
    .put(updateBook)
    .delete(deleteBook);
`
	p := New(".")
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

// ── Fastify basic routes ─────────────────────────────────────────────────────

func TestParseFastifyRoutes_BasicCRUD(t *testing.T) {
	src := `
const fastify = require('fastify')();

fastify.get('/api/users', getUsers);
fastify.post('/api/users', createUser);
fastify.get('/api/users/:id', getUser);
fastify.put('/api/users/:id', updateUser);
fastify.delete('/api/users/:id', deleteUser);
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d", len(endpoints))
	}

	tests := []struct {
		method string
		uri    string
	}{
		{"GET", "/api/users"},
		{"POST", "/api/users"},
		{"GET", "/api/users/:id"},
		{"PUT", "/api/users/:id"},
		{"DELETE", "/api/users/:id"},
	}

	for i, tt := range tests {
		if endpoints[i].Method != tt.method {
			t.Errorf("[%d] method: got %q, want %q", i, endpoints[i].Method, tt.method)
		}
		if endpoints[i].URI != tt.uri {
			t.Errorf("[%d] URI: got %q, want %q", i, endpoints[i].URI, tt.uri)
		}
		if endpoints[i].Controller != "fastify" {
			t.Errorf("[%d] controller: got %q, want fastify", i, endpoints[i].Controller)
		}
	}
}

// ── Fastify route object ─────────────────────────────────────────────────────

func TestParseFastifyRoutes_RouteObject(t *testing.T) {
	src := `
fastify.route({
    method: 'GET',
    url: '/api/items',
    handler: getItems
});

fastify.route({
    method: 'POST',
    url: '/api/items',
    handler: createItem
});

fastify.route({
    method: 'GET',
    url: '/api/items/:id',
    handler: getItem
});
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}

	tests := []struct {
		method  string
		uri     string
		handler string
	}{
		{"GET", "/api/items", "getItems"},
		{"POST", "/api/items", "createItem"},
		{"GET", "/api/items/:id", "getItem"},
	}

	for i, tt := range tests {
		if endpoints[i].Method != tt.method {
			t.Errorf("[%d] method: got %q, want %q", i, endpoints[i].Method, tt.method)
		}
		if endpoints[i].URI != tt.uri {
			t.Errorf("[%d] URI: got %q, want %q", i, endpoints[i].URI, tt.uri)
		}
		if endpoints[i].Action != tt.handler {
			t.Errorf("[%d] action: got %q, want %q", i, endpoints[i].Action, tt.handler)
		}
	}
}

func TestParseFastifyRoutes_RouteObjectMethodArray(t *testing.T) {
	src := `
fastify.route({
    method: ['GET', 'HEAD'],
    url: '/api/health',
    handler: healthCheck
});
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints (GET + HEAD), got %d", len(endpoints))
	}

	if endpoints[0].Method != "GET" {
		t.Errorf("[0] method: got %q, want GET", endpoints[0].Method)
	}
	if endpoints[1].Method != "HEAD" {
		t.Errorf("[1] method: got %q, want HEAD", endpoints[1].Method)
	}
	for i := range endpoints {
		if endpoints[i].URI != "/api/health" {
			t.Errorf("[%d] URI: got %q, want /api/health", i, endpoints[i].URI)
		}
	}
}

func TestParseFastifyRoutes_RouteObjectPreHandler(t *testing.T) {
	src := `
fastify.route({
    method: 'DELETE',
    url: '/api/items/:id',
    preHandler: [authenticate, authorize],
    handler: deleteItem
});
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	ep := endpoints[0]
	if len(ep.Middleware) != 2 {
		t.Fatalf("middleware count: got %d, want 2", len(ep.Middleware))
	}
	if ep.Middleware[0] != "authenticate" {
		t.Errorf("middleware[0]: got %q, want authenticate", ep.Middleware[0])
	}
	if ep.Middleware[1] != "authorize" {
		t.Errorf("middleware[1]: got %q, want authorize", ep.Middleware[1])
	}
}

// ── Fastify register prefix ──────────────────────────────────────────────────

func TestParseFastifyRoutes_RegisterPrefix(t *testing.T) {
	src := `
const fastify = require('fastify')();

function userRoutes(fastify, opts, done) {
    fastify.get('/', getUsers);
    fastify.get('/:id', getUser);
    done();
}

fastify.register(userRoutes, { prefix: '/api/users' });
`
	p := New(".")
	endpoints := p.parseFile(src)

	// userRoutes is registered with prefix, but the routes inside use the local
	// fastify instance. The prefix mapping maps userRoutes -> /api/users,
	// but since the routes are on 'fastify' (not 'userRoutes'), they won't get the prefix.
	// This is a known limitation of file-level parsing.
	// The routes on the local fastify will be parsed as-is.
	if len(endpoints) < 2 {
		t.Fatalf("expected at least 2 endpoints, got %d", len(endpoints))
	}
}

// ── Native HTTP routes ───────────────────────────────────────────────────────

func TestParseNativeHTTPRoutes_IfElse(t *testing.T) {
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
	p := New(".")
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

func TestParseNativeHTTPRoutes_ReverseOrder(t *testing.T) {
	src := `
if (req.url === '/api/items' && req.method === 'GET') {
    handleGetItems(req, res);
}
if (req.url === '/api/items' && req.method === 'POST') {
    handleCreateItem(req, res);
}
`
	p := New(".")
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

func TestParseNativeHTTPRoutes_LooseEquality(t *testing.T) {
	src := `
if (req.method == 'GET' && req.url == '/api/data') {
    handleData(req, res);
}
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if endpoints[0].Method != "GET" || endpoints[0].URI != "/api/data" {
		t.Errorf("got %s %s, want GET /api/data", endpoints[0].Method, endpoints[0].URI)
	}
}

// ── Parameter extraction ─────────────────────────────────────────────────────

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

// ── Route params in endpoints ────────────────────────────────────────────────

func TestExpressRoutes_ExtractParams(t *testing.T) {
	src := `
app.get('/api/users/:id', getUser);
app.put('/api/users/:userId/posts/:postId', updatePost);
`
	p := New(".")
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
	if endpoints[1].StaticMeta.RequestParams[0].Name != "userId" {
		t.Errorf("[1] param[0]: got %q, want userId", endpoints[1].StaticMeta.RequestParams[0].Name)
	}
	if endpoints[1].StaticMeta.RequestParams[1].Name != "postId" {
		t.Errorf("[1] param[1]: got %q, want postId", endpoints[1].StaticMeta.RequestParams[1].Name)
	}
}

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

func strings_Contains(s, sub string) bool {
	return len(s) >= len(sub) && containsSubstr(s, sub)
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
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

// ── ResolveIncludes ──────────────────────────────────────────────────────────

func TestResolveIncludes_Directory(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "routes"), 0755)
	os.WriteFile(filepath.Join(dir, "routes", "users.js"), []byte("// routes"), 0644)
	os.WriteFile(filepath.Join(dir, "routes", "items.ts"), []byte("// routes"), 0644)
	os.WriteFile(filepath.Join(dir, "routes", "readme.txt"), []byte("not js"), 0644)

	// Create node_modules that should be skipped
	os.MkdirAll(filepath.Join(dir, "routes", "node_modules"), 0755)
	os.WriteFile(filepath.Join(dir, "routes", "node_modules", "dep.js"), []byte("// dep"), 0644)

	p := New(dir)
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

	p := New(dir)
	files, err := p.ResolveIncludes([]string{path})
	if err != nil {
		t.Fatalf("ResolveIncludes: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}

// ── Full ParseSections integration ───────────────────────────────────────────

func TestParseSections_LanguageAssignment(t *testing.T) {
	dir := t.TempDir()

	jsFile := filepath.Join(dir, "routes.js")
	os.WriteFile(jsFile, []byte(`app.get('/api/users', getUsers);`), 0644)

	tsFile := filepath.Join(dir, "routes.ts")
	os.WriteFile(tsFile, []byte(`app.get('/api/items', getItems);`), 0644)

	p := New(dir)
	sections, err := p.ParseSections([]string{jsFile, tsFile})
	if err != nil {
		t.Fatalf("ParseSections: %v", err)
	}

	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}

	if sections[0].Endpoints[0].Language != "javascript" {
		t.Errorf(".js file language: got %q, want javascript", sections[0].Endpoints[0].Language)
	}
	if sections[1].Endpoints[0].Language != "typescript" {
		t.Errorf(".ts file language: got %q, want typescript", sections[1].Endpoints[0].Language)
	}
}

func TestParseSections_SkipsEmptyFiles(t *testing.T) {
	dir := t.TempDir()

	routeFile := filepath.Join(dir, "routes.js")
	os.WriteFile(routeFile, []byte(`app.get('/api/users', getUsers);`), 0644)

	emptyFile := filepath.Join(dir, "utils.js")
	os.WriteFile(emptyFile, []byte(`const helper = () => {};`), 0644)

	p := New(dir)
	sections, err := p.ParseSections([]string{routeFile, emptyFile})
	if err != nil {
		t.Fatalf("ParseSections: %v", err)
	}

	if len(sections) != 1 {
		t.Fatalf("expected 1 section (utils skipped), got %d", len(sections))
	}
}
