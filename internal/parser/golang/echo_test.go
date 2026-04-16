package golang

import "testing"

func TestEcho_BasicCRUD(t *testing.T) {
	src := `
package main

import "github.com/labstack/echo/v4"

func main() {
	e := echo.New()
	e.GET("/api/users", listUsers)
	e.POST("/api/users", createUser)
	e.GET("/api/users/:id", getUser)
	e.PUT("/api/users/:id", updateUser)
	e.DELETE("/api/users/:id", deleteUser)
}
`
	p := NewEcho(".")
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
		if ep.Controller != "e" {
			t.Errorf("[%d] controller: got %q, want e", i, ep.Controller)
		}
	}
}

func TestEcho_Group(t *testing.T) {
	src := `
package main

func main() {
	e := echo.New()
	g := e.Group("/api")
	g.GET("/users", listUsers)
	g.POST("/users", createUser)
}
`
	p := NewEcho(".")
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

func TestEcho_NestedGroups(t *testing.T) {
	src := `
package main

func main() {
	e := echo.New()
	api := e.Group("/api")
	v1 := api.Group("/v1")
	v1.GET("/users", listUsers)
	v1.GET("/items", listItems)
}
`
	p := NewEcho(".")
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

func TestEcho_Middleware(t *testing.T) {
	src := `
package main

func main() {
	e := echo.New()
	e.GET("/api/admin", getAdmin, authMiddleware)
	e.POST("/api/admin/action", doAction, authMiddleware, roleMiddleware)
}
`
	p := NewEcho(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if len(endpoints[0].Middleware) != 1 || endpoints[0].Middleware[0] != "authMiddleware" {
		t.Errorf("[0] middleware: got %v, want [authMiddleware]", endpoints[0].Middleware)
	}

	if endpoints[0].Action != "getAdmin" {
		t.Errorf("[0] action: got %q, want getAdmin", endpoints[0].Action)
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

func TestEcho_AnyMethod(t *testing.T) {
	src := `e.Any("/api/cors", corsHandler)`
	p := NewEcho(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].Method != "ALL" {
		t.Errorf("method: got %q, want ALL", endpoints[0].Method)
	}
}

func TestEcho_PatchOptionsHead(t *testing.T) {
	src := `
e.PATCH("/api/users/:id", patchUser)
e.OPTIONS("/api/users", handleOptions)
e.HEAD("/api/health", healthHead)
`
	p := NewEcho(".")
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

func TestEcho_PathParams(t *testing.T) {
	src := `e.GET("/api/users/:userID/posts/:postID", getPost)`
	p := NewEcho(".")
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

func TestEcho_CommentsStripped(t *testing.T) {
	src := `
// e.GET("/commented", handler)
e.GET("/real", realHandler)
`
	p := NewEcho(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].URI != "/real" {
		t.Errorf("URI: got %q, want /real", endpoints[0].URI)
	}
}

func TestEcho_NoRoutes(t *testing.T) {
	src := `
package utils

func helper() {
	fmt.Println("no routes")
}
`
	p := NewEcho(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 0 {
		t.Fatalf("expected 0 endpoints, got %d", len(endpoints))
	}
}

func TestEcho_GroupWithMiddleware(t *testing.T) {
	src := `
package main

func main() {
	e := echo.New()
	admin := e.Group("/admin", authRequired)
	admin.GET("/dashboard", getDashboard)
	admin.POST("/settings", updateSettings)
}
`
	p := NewEcho(".")
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

func TestEcho_ControllerStyle(t *testing.T) {
	src := `
package main

func main() {
	e := echo.New()
	e.GET("/api/users", controllers.ListUsers)
	e.POST("/api/users", controllers.CreateUser)
}
`
	p := NewEcho(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].Action != "controllers.ListUsers" {
		t.Errorf("[0] action: got %q, want controllers.ListUsers", endpoints[0].Action)
	}
	if endpoints[1].Action != "controllers.CreateUser" {
		t.Errorf("[1] action: got %q, want controllers.CreateUser", endpoints[1].Action)
	}
}
