package golang

import "testing"

func TestMux_BasicCRUD(t *testing.T) {
	src := `
package main

import "github.com/gorilla/mux"

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/api/users", listUsers).Methods("GET")
	r.HandleFunc("/api/users", createUser).Methods("POST")
	r.HandleFunc("/api/users/{id}", getUser).Methods("GET")
	r.HandleFunc("/api/users/{id}", updateUser).Methods("PUT")
	r.HandleFunc("/api/users/{id}", deleteUser).Methods("DELETE")
}
`
	p := NewMux(".")
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
		{"GET", "/api/users/{id}", "getUser"},
		{"PUT", "/api/users/{id}", "updateUser"},
		{"DELETE", "/api/users/{id}", "deleteUser"},
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
		if ep.Controller != "r" {
			t.Errorf("[%d] controller: got %q, want r", i, ep.Controller)
		}
	}
}

func TestMux_MultipleMethods(t *testing.T) {
	src := `
package main

func main() {
	r.HandleFunc("/api/users/{id}", userHandler).Methods("GET", "PUT")
}
`
	p := NewMux(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].Method != "GET" {
		t.Errorf("[0] method: got %q, want GET", endpoints[0].Method)
	}
	if endpoints[1].Method != "PUT" {
		t.Errorf("[1] method: got %q, want PUT", endpoints[1].Method)
	}
}

func TestMux_WithoutMethods(t *testing.T) {
	src := `
package main

func main() {
	r.HandleFunc("/api/health", healthCheck)
}
`
	p := NewMux(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if endpoints[0].Method != "ALL" {
		t.Errorf("method: got %q, want ALL", endpoints[0].Method)
	}
}

func TestMux_NamedRoute(t *testing.T) {
	src := `
package main

func main() {
	r.HandleFunc("/api/users", listUsers).Methods("GET").Name("listUsers")
}
`
	p := NewMux(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if endpoints[0].Name != "listUsers" {
		t.Errorf("name: got %q, want listUsers", endpoints[0].Name)
	}
}

func TestMux_Subrouter(t *testing.T) {
	src := `
package main

func main() {
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/users", listUsers).Methods("GET")
	api.HandleFunc("/users/{id}", getUser).Methods("GET")
}
`
	p := NewMux(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].URI != "/api/users" {
		t.Errorf("[0] URI: got %q, want /api/users", endpoints[0].URI)
	}
	if endpoints[1].URI != "/api/users/{id}" {
		t.Errorf("[1] URI: got %q, want /api/users/{id}", endpoints[1].URI)
	}
}

func TestMux_NestedSubrouters(t *testing.T) {
	src := `
package main

func main() {
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	v1 := api.PathPrefix("/v1").Subrouter()
	v1.HandleFunc("/users", listUsers).Methods("GET")
	v1.HandleFunc("/items", listItems).Methods("GET")
}
`
	p := NewMux(".")
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

func TestMux_Handle(t *testing.T) {
	src := `
package main

func main() {
	r := mux.NewRouter()
	r.Handle("/api/users", usersHandler).Methods("GET")
}
`
	p := NewMux(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if endpoints[0].Method != "GET" {
		t.Errorf("method: got %q, want GET", endpoints[0].Method)
	}
	if endpoints[0].URI != "/api/users" {
		t.Errorf("URI: got %q, want /api/users", endpoints[0].URI)
	}
}

func TestMux_PathParams(t *testing.T) {
	src := `
package main

func main() {
	r.HandleFunc("/api/users/{userID}/posts/{postID}", getPost).Methods("GET")
}
`
	p := NewMux(".")
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

func TestMux_CommentsStripped(t *testing.T) {
	src := `
package main

// r.HandleFunc("/commented", handler).Methods("GET")
r.HandleFunc("/real", handler).Methods("GET")
`
	p := NewMux(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].URI != "/real" {
		t.Errorf("URI: got %q, want /real", endpoints[0].URI)
	}
}

func TestMux_NoRoutes(t *testing.T) {
	src := `
package utils

func helper() {
	fmt.Println("no routes")
}
`
	p := NewMux(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 0 {
		t.Fatalf("expected 0 endpoints, got %d", len(endpoints))
	}
}

func TestMux_SubrouterWithStripPrefix(t *testing.T) {
	src := `
package main

func main() {
	r := mux.NewRouter()
	s := r.PathPrefix("/api").Headers("Content-Type", "application/json").Subrouter()
	s.HandleFunc("/users", listUsers).Methods("GET")
}
`
	p := NewMux(".")
	endpoints := p.parseFile(src)

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if endpoints[0].URI != "/api/users" {
		t.Errorf("URI: got %q, want /api/users", endpoints[0].URI)
	}
}
