package node

import (
	"testing"

	"github.com/divan1319/apidocgen/pkg/models"
)

// ── Express ALL method ───────────────────────────────────────────────────────

func TestExpress_AllMethod(t *testing.T) {
	src := `
app.all('/api/cors-check', handleCORS);
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].Method != "ALL" {
		t.Errorf("method: got %q, want ALL", endpoints[0].Method)
	}
	if endpoints[0].URI != "/api/cors-check" {
		t.Errorf("URI: got %q, want /api/cors-check", endpoints[0].URI)
	}
}

// ── Express PATCH and OPTIONS ────────────────────────────────────────────────

func TestExpress_PatchAndOptions(t *testing.T) {
	src := `
app.patch('/api/users/:id', patchUser);
app.options('/api/users', handleOptions);
app.head('/api/health', healthHead);
`
	p := New(".")
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

// ── Express double quotes vs single quotes ───────────────────────────────────

func TestExpress_QuoteStyles(t *testing.T) {
	src := `
app.get('/api/single', handler1);
app.get("/api/double", handler2);
`
	p := New(".")
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

// ── Fastify with path field instead of url ───────────────────────────────────

func TestFastify_RouteObjectPathField(t *testing.T) {
	src := `
fastify.route({
    method: 'GET',
    path: '/api/legacy',
    handler: getLegacy
});
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].URI != "/api/legacy" {
		t.Errorf("URI: got %q, want /api/legacy", endpoints[0].URI)
	}
}

// ── Fastify onRequest middleware ──────────────────────────────────────────────

func TestFastify_OnRequestMiddleware(t *testing.T) {
	src := `
fastify.route({
    method: 'GET',
    url: '/api/protected',
    onRequest: [verifyToken, checkRole],
    handler: getProtected
});
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if len(endpoints[0].Middleware) != 2 {
		t.Fatalf("middleware count: got %d, want 2", len(endpoints[0].Middleware))
	}
	if endpoints[0].Middleware[0] != "verifyToken" {
		t.Errorf("middleware[0]: got %q, want verifyToken", endpoints[0].Middleware[0])
	}
	if endpoints[0].Middleware[1] != "checkRole" {
		t.Errorf("middleware[1]: got %q, want checkRole", endpoints[0].Middleware[1])
	}
}

// ── Mixed frameworks in same file ────────────────────────────────────────────

func TestMixedFrameworks(t *testing.T) {
	src := `
const express = require('express');
const http = require('http');

const app = express();

app.get('/api/express-route', handler);

if (req.method === 'GET' && req.url === '/api/native-route') {
    handleNative(req, res);
}
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].URI != "/api/express-route" {
		t.Errorf("[0] URI: got %q, want /api/express-route", endpoints[0].URI)
	}
	if endpoints[1].URI != "/api/native-route" {
		t.Errorf("[1] URI: got %q, want /api/native-route", endpoints[1].URI)
	}
}

// ── TypeScript annotations ───────────────────────────────────────────────────

func TestTypeScript_Annotations(t *testing.T) {
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
	p := New(".")
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

// ── No routes in file ────────────────────────────────────────────────────────

func TestNoRoutesInFile(t *testing.T) {
	src := `
const express = require('express');
const helper = () => {
    console.log('no routes here');
};
module.exports = helper;
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 0 {
		t.Fatalf("expected 0 endpoints, got %d", len(endpoints))
	}
}

// ── Comments stripped ────────────────────────────────────────────────────────

func TestCommentsStripped(t *testing.T) {
	src := `
// app.get('/api/commented-out', handler);
/*
app.get('/api/block-commented', handler);
*/
app.get('/api/real-route', handler);
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint (commented routes stripped), got %d", len(endpoints))
	}
	if endpoints[0].URI != "/api/real-route" {
		t.Errorf("URI: got %q, want /api/real-route", endpoints[0].URI)
	}
}

// ── Test code skipped ────────────────────────────────────────────────────────

func TestTestCodeSkipped(t *testing.T) {
	src := `
const app = express();
app.get('/api/real', handler);

describe('API tests', () => {
    test.get('/api/fake');
    it.get('/api/also-fake');
});
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint (test code skipped), got %d", len(endpoints))
	}
	if endpoints[0].URI != "/api/real" {
		t.Errorf("URI: got %q, want /api/real", endpoints[0].URI)
	}
}

// ── Deduplication ────────────────────────────────────────────────────────────

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

// ── Controller-style handlers ────────────────────────────────────────────────

func TestExpress_ControllerStyleHandler(t *testing.T) {
	src := `
app.get('/api/users', usersController.getAll);
app.post('/api/users', usersController.create);
`
	p := New(".")
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

// ── Express Router without explicit mount ────────────────────────────────────

func TestExpress_RouterWithoutMount(t *testing.T) {
	src := `
const router = require('express').Router();

router.get('/users', getUsers);
router.post('/users', createUser);
router.get('/users/:id', getUser);

module.exports = router;
`
	p := New(".")
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

// ── Fastify async handler ────────────────────────────────────────────────────

func TestFastify_AsyncHandler(t *testing.T) {
	src := `
fastify.get('/api/users', async (request, reply) => {
    const users = await db.getUsers();
    return users;
});

fastify.post('/api/users', async (request, reply) => {
    const user = await db.createUser(request.body);
    reply.code(201);
    return user;
});
`
	p := New(".")
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

// ── Native HTTP switch/case on URL ───────────────────────────────────────────

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
	p := New(".")
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

// ── Native HTTP switch/case on method with URL context ───────────────────────

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
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	for _, ep := range endpoints {
		if ep.URI != "/api/users" {
			t.Errorf("URI: got %q, want /api/users", ep.URI)
		}
	}

	methods := map[string]bool{}
	for _, ep := range endpoints {
		methods[ep.Method] = true
	}
	if !methods["GET"] {
		t.Error("missing GET method")
	}
	if !methods["POST"] {
		t.Error("missing POST method")
	}
}

// ── Express with multiline handler ───────────────────────────────────────────

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
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if endpoints[0].Method != "GET" || endpoints[0].URI != "/api/users" {
		t.Errorf("got %s %s, want GET /api/users", endpoints[0].Method, endpoints[0].URI)
	}
}

// ── Express plugin pattern ───────────────────────────────────────────────────

func TestExpress_PluginPattern(t *testing.T) {
	src := `
module.exports = function(app) {
    app.get('/api/auth/login', login);
    app.post('/api/auth/register', register);
    app.post('/api/auth/logout', logout);
};
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}
}

// ── Fastify plugin pattern ───────────────────────────────────────────────────

func TestFastify_PluginPattern(t *testing.T) {
	src := `
module.exports = async function(fastify, opts) {
    fastify.get('/', getUsers);
    fastify.get('/:id', getUser);
    fastify.post('/', createUser);
    fastify.put('/:id', updateUser);
    fastify.delete('/:id', deleteUser);
};
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d", len(endpoints))
	}
}

// ── Express complex nested routers ───────────────────────────────────────────

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
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	// v1Router gets prefix /v1 from apiRouter.use('/v1', v1Router)
	if endpoints[0].URI != "/v1/users" {
		t.Errorf("[0] URI: got %q, want /v1/users", endpoints[0].URI)
	}
	if endpoints[1].URI != "/v1/items" {
		t.Errorf("[1] URI: got %q, want /v1/items", endpoints[1].URI)
	}
}

// ── Handler extraction ───────────────────────────────────────────────────────

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

// ── Build router prefixes ────────────────────────────────────────────────────

func TestBuildRouterPrefixes(t *testing.T) {
	src := `
const usersRouter = express.Router();
const itemsRouter = express.Router();

app.use('/api/users', usersRouter);
app.use('/api/items', itemsRouter);
`
	prefixes := buildRouterPrefixes(src)

	if prefixes["usersRouter"] != "/api/users" {
		t.Errorf("usersRouter prefix: got %q, want /api/users", prefixes["usersRouter"])
	}
	if prefixes["itemsRouter"] != "/api/items" {
		t.Errorf("itemsRouter prefix: got %q, want /api/items", prefixes["itemsRouter"])
	}
}

func TestBuildRouterPrefixes_Fastify(t *testing.T) {
	src := `
fastify.register(userRoutes, { prefix: '/api/users' });
fastify.register(itemRoutes, { prefix: '/api/items' });
`
	prefixes := buildRouterPrefixes(src)

	if prefixes["userRoutes"] != "/api/users" {
		t.Errorf("userRoutes prefix: got %q, want /api/users", prefixes["userRoutes"])
	}
	if prefixes["itemRoutes"] != "/api/items" {
		t.Errorf("itemRoutes prefix: got %q, want /api/items", prefixes["itemRoutes"])
	}
}

// ── SplitTopLevel ────────────────────────────────────────────────────────────

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

// ── Fastify route object edge cases ──────────────────────────────────────────

func TestFastify_RouteObjectWithSchema(t *testing.T) {
	src := `
fastify.route({
    method: 'POST',
    url: '/api/users',
    schema: {
        body: {
            type: 'object',
            properties: {
                name: { type: 'string' },
                email: { type: 'string' }
            },
            required: ['name', 'email']
        },
        response: {
            201: {
                type: 'object',
                properties: {
                    id: { type: 'integer' },
                    name: { type: 'string' }
                }
            }
        }
    },
    handler: createUser
});
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	ep := endpoints[0]
	if ep.Method != "POST" || ep.URI != "/api/users" {
		t.Errorf("got %s %s, want POST /api/users", ep.Method, ep.URI)
	}
	if ep.Action != "createUser" {
		t.Errorf("action: got %q, want createUser", ep.Action)
	}
}

// ── Express controller.method as handler ─────────────────────────────────────

func TestExpress_DottedHandler(t *testing.T) {
	src := `
const userCtrl = require('./controllers/users');
app.get('/api/users', userCtrl.list);
app.get('/api/users/:id', userCtrl.show);
app.post('/api/users', userCtrl.create);
app.put('/api/users/:id', userCtrl.update);
app.delete('/api/users/:id', userCtrl.destroy);
`
	p := New(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d", len(endpoints))
	}

	expectedActions := []string{
		"userCtrl.list",
		"userCtrl.show",
		"userCtrl.create",
		"userCtrl.update",
		"userCtrl.destroy",
	}

	for i, want := range expectedActions {
		if endpoints[i].Action != want {
			t.Errorf("[%d] action: got %q, want %q", i, endpoints[i].Action, want)
		}
	}
}

