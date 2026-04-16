package golang

import "testing"

func TestFiber_BasicCRUD(t *testing.T) {
	src := `
package main

import "github.com/gofiber/fiber/v2"

func main() {
	app := fiber.New()
	app.Get("/api/users", listUsers)
	app.Post("/api/users", createUser)
	app.Get("/api/users/:id", getUser)
	app.Put("/api/users/:id", updateUser)
	app.Delete("/api/users/:id", deleteUser)
}
`
	p := NewFiber(".")
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
			t.Errorf("[%d] controller: got %q, want app", i, ep.Controller)
		}
	}
}

func TestFiber_Group(t *testing.T) {
	src := `
package main

func main() {
	app := fiber.New()
	api := app.Group("/api")
	api.Get("/users", listUsers)
	api.Post("/users", createUser)
}
`
	p := NewFiber(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].URI != "/api/users" {
		t.Errorf("[0] URI: got %q, want /api/users", endpoints[0].URI)
	}
	if endpoints[1].URI != "/api/users" {
		t.Errorf("[1] URI: got %q, want /api/users", endpoints[1].URI)
	}
	if endpoints[0].Method != "GET" {
		t.Errorf("[0] method: got %q, want GET", endpoints[0].Method)
	}
	if endpoints[1].Method != "POST" {
		t.Errorf("[1] method: got %q, want POST", endpoints[1].Method)
	}
}

func TestFiber_NestedGroups(t *testing.T) {
	src := `
package main

func main() {
	app := fiber.New()
	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/users", listUsers)
	v1.Get("/items", listItems)
}
`
	p := NewFiber(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].URI != "/api/v1/users" {
		t.Errorf("[0] URI: got %q, want /api/v1/users", endpoints[0].URI)
	}
	if endpoints[1].URI != "/api/v1/items" {
		t.Errorf("[1] URI: got %q, want /api/v1/items", endpoints[1].URI)
	}
}

func TestFiber_Middleware(t *testing.T) {
	src := `
package main

func main() {
	app := fiber.New()
	app.Get("/api/admin", authMiddleware, getAdmin)
	app.Post("/api/admin/action", authMiddleware, roleMiddleware, doAction)
}
`
	p := NewFiber(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if len(endpoints[0].Middleware) != 1 || endpoints[0].Middleware[0] != "authMiddleware" {
		t.Errorf("[0] middleware: got %v, want [authMiddleware]", endpoints[0].Middleware)
	}

	if len(endpoints[1].Middleware) != 2 {
		t.Errorf("[1] middleware count: got %d, want 2", len(endpoints[1].Middleware))
	} else {
		if endpoints[1].Middleware[0] != "authMiddleware" {
			t.Errorf("[1] middleware[0]: got %q, want authMiddleware", endpoints[1].Middleware[0])
		}
		if endpoints[1].Middleware[1] != "roleMiddleware" {
			t.Errorf("[1] middleware[1]: got %q, want roleMiddleware", endpoints[1].Middleware[1])
		}
	}
}

func TestFiber_AllMethod(t *testing.T) {
	src := `app.All("/api/cors", corsHandler)`
	p := NewFiber(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].Method != "ALL" {
		t.Errorf("method: got %q, want ALL", endpoints[0].Method)
	}
}

func TestFiber_PatchOptionsHead(t *testing.T) {
	src := `
app.Patch("/api/users/:id", patchUser)
app.Options("/api/users", handleOptions)
app.Head("/api/health", healthHead)
`
	p := NewFiber(".")
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

func TestFiber_PathParams(t *testing.T) {
	src := `app.Get("/api/users/:userID/posts/:postID", getPost)`
	p := NewFiber(".")
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

func TestFiber_CommentsStripped(t *testing.T) {
	src := `
// app.Get("/commented", handler)
app.Get("/real", realHandler)
`
	p := NewFiber(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].URI != "/real" {
		t.Errorf("URI: got %q, want /real", endpoints[0].URI)
	}
}

func TestFiber_NoRoutes(t *testing.T) {
	src := `
package utils

func helper() {
	fmt.Println("no routes")
}
`
	p := NewFiber(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 0 {
		t.Fatalf("expected 0 endpoints, got %d", len(endpoints))
	}
}

func TestFiber_GroupWithMiddleware(t *testing.T) {
	src := `
package main

func main() {
	app := fiber.New()
	admin := app.Group("/admin", authRequired)
	admin.Get("/dashboard", getDashboard)
	admin.Post("/settings", updateSettings)
}
`
	p := NewFiber(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].URI != "/admin/dashboard" {
		t.Errorf("[0] URI: got %q, want /admin/dashboard", endpoints[0].URI)
	}
	if endpoints[1].URI != "/admin/settings" {
		t.Errorf("[1] URI: got %q, want /admin/settings", endpoints[1].URI)
	}
}
