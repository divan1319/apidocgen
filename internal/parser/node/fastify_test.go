package node

import (
	"testing"
)

// ── Fastify basic CRUD ───────────────────────────────────────────────────────

func TestFastify_BasicCRUD(t *testing.T) {
	src := `
const fastify = require('fastify')();

fastify.get('/api/users', getUsers);
fastify.post('/api/users', createUser);
fastify.get('/api/users/:id', getUser);
fastify.put('/api/users/:id', updateUser);
fastify.delete('/api/users/:id', deleteUser);
`
	p := NewFastify(".")
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

func TestFastify_RouteObject(t *testing.T) {
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
	p := NewFastify(".")
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

func TestFastify_RouteObjectMethodArray(t *testing.T) {
	src := `
fastify.route({
    method: ['GET', 'HEAD'],
    url: '/api/health',
    handler: healthCheck
});
`
	p := NewFastify(".")
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

func TestFastify_RouteObjectPreHandler(t *testing.T) {
	src := `
fastify.route({
    method: 'DELETE',
    url: '/api/items/:id',
    preHandler: [authenticate, authorize],
    handler: deleteItem
});
`
	p := NewFastify(".")
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

func TestFastify_RouteObjectOnRequest(t *testing.T) {
	src := `
fastify.route({
    method: 'GET',
    url: '/api/protected',
    onRequest: [verifyToken, checkRole],
    handler: getProtected
});
`
	p := NewFastify(".")
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

func TestFastify_RouteObjectPathField(t *testing.T) {
	src := `
fastify.route({
    method: 'GET',
    path: '/api/legacy',
    handler: getLegacy
});
`
	p := NewFastify(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].URI != "/api/legacy" {
		t.Errorf("URI: got %q, want /api/legacy", endpoints[0].URI)
	}
}

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
	p := NewFastify(".")
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

// ── Fastify register prefix ──────────────────────────────────────────────────

func TestFastify_RegisterPrefix(t *testing.T) {
	src := `
const fastify = require('fastify')();

function userRoutes(fastify, opts, done) {
    fastify.get('/', getUsers);
    fastify.get('/:id', getUser);
    done();
}

fastify.register(userRoutes, { prefix: '/api/users' });
`
	p := NewFastify(".")
	endpoints := p.parseFile(src)

	if len(endpoints) < 2 {
		t.Fatalf("expected at least 2 endpoints, got %d", len(endpoints))
	}
}

func TestFastify_BuildPrefixes(t *testing.T) {
	src := `
fastify.register(userRoutes, { prefix: '/api/users' });
fastify.register(itemRoutes, { prefix: '/api/items' });
`
	prefixes := buildFastifyPrefixes(src)

	if prefixes["userRoutes"] != "/api/users" {
		t.Errorf("userRoutes prefix: got %q, want /api/users", prefixes["userRoutes"])
	}
	if prefixes["itemRoutes"] != "/api/items" {
		t.Errorf("itemRoutes prefix: got %q, want /api/items", prefixes["itemRoutes"])
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
	p := NewFastify(".")
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
	p := NewFastify(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d", len(endpoints))
	}
}
