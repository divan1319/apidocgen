package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFastAPI_BasicCRUD(t *testing.T) {
	src := `
from fastapi import FastAPI

app = FastAPI()

@app.get("/api/users")
async def list_users():
    pass

@app.post("/api/users")
def create_user():
    pass

@app.get("/api/users/{user_id}")
async def get_user(user_id: int):
    pass

@app.put("/api/users/{user_id}")
async def update_user(user_id: int):
    pass

@app.delete("/api/users/{user_id}")
async def delete_user(user_id: int):
    pass
`
	p := NewFastAPI(".")
	eps := p.parseFile(src)
	if len(eps) != 5 {
		t.Fatalf("esperaba 5 endpoints, hay %d", len(eps))
	}
	want := []struct{ method, uri, action string }{
		{"GET", "/api/users", "list_users"},
		{"POST", "/api/users", "create_user"},
		{"GET", "/api/users/{user_id}", "get_user"},
		{"PUT", "/api/users/{user_id}", "update_user"},
		{"DELETE", "/api/users/{user_id}", "delete_user"},
	}
	for i, w := range want {
		if eps[i].Method != w.method {
			t.Errorf("[%d] método: %q != %q", i, eps[i].Method, w.method)
		}
		if eps[i].URI != w.uri {
			t.Errorf("[%d] URI: %q != %q", i, eps[i].URI, w.uri)
		}
		if eps[i].Action != w.action {
			t.Errorf("[%d] acción: %q != %q", i, eps[i].Action, w.action)
		}
		if eps[i].Controller != "app" {
			t.Errorf("[%d] controller: %q", i, eps[i].Controller)
		}
		if len(eps[i].StaticMeta.RequestParams) == 0 && i >= 2 {
			t.Errorf("[%d] debería tener params de ruta", i)
		}
	}
}

func TestFastAPI_APIRouterPrefix(t *testing.T) {
	src := `
from fastapi import APIRouter

router = APIRouter(prefix="/items")

@router.get("/")
def list_items():
    pass

@router.get("/{item_id}")
def read_item(item_id: str):
    pass
`
	p := NewFastAPI(".")
	eps := p.parseFile(src)
	if len(eps) != 2 {
		t.Fatalf("esperaba 2, hay %d", len(eps))
	}
	if eps[0].URI != "/items" {
		t.Errorf("URI[0]: %q", eps[0].URI)
	}
	if eps[1].URI != "/items/{item_id}" {
		t.Errorf("URI[1]: %q", eps[1].URI)
	}
}

func TestFastAPI_IncludeRouter(t *testing.T) {
	src := `
from fastapi import FastAPI, APIRouter

app = FastAPI()
users = APIRouter(prefix="/profiles")

@users.get("/")
def list_users():
    pass

@users.get("/{uid}")
def get_user(uid: str):
    pass

app.include_router(users, prefix="/api/users")
`
	p := NewFastAPI(".")
	eps := p.parseFile(src)
	if len(eps) != 2 {
		t.Fatalf("esperaba 2, hay %d", len(eps))
	}
	if eps[0].URI != "/api/users/profiles" {
		t.Errorf("URI[0]: %q", eps[0].URI)
	}
	if eps[1].URI != "/api/users/profiles/{uid}" {
		t.Errorf("URI[1]: %q", eps[1].URI)
	}
}

func TestFastAPI_NestedIncludeRouters(t *testing.T) {
	src := `
from fastapi import FastAPI, APIRouter

app = FastAPI()
v1 = APIRouter()
items = APIRouter(prefix="/things")

@items.get("/x")
def get_x():
    pass

v1.include_router(items, prefix="/catalog")
app.include_router(v1, prefix="/api")
`
	p := NewFastAPI(".")
	eps := p.parseFile(src)
	if len(eps) != 1 {
		t.Fatalf("esperaba 1, hay %d", len(eps))
	}
	if eps[0].URI != "/api/catalog/things/x" {
		t.Errorf("URI: %q", eps[0].URI)
	}
}

func TestFastAPI_APIRouteMethods(t *testing.T) {
	src := `
from fastapi import FastAPI

app = FastAPI()

@app.api_route("/report", methods=["GET", "HEAD"])
async def report():
    pass

@app.api_route("/bulk", methods=("POST", "PUT"))
def bulk():
    pass
`
	p := NewFastAPI(".")
	eps := p.parseFile(src)
	if len(eps) != 4 {
		t.Fatalf("esperaba 4 métodos, hay %d", len(eps))
	}
	methods := map[string]bool{}
	for _, e := range eps[:2] {
		if e.URI != "/report" {
			t.Errorf("report URI: %q", e.URI)
		}
		methods[e.Method] = true
	}
	if !methods["GET"] || !methods["HEAD"] {
		t.Errorf("report methods: %+v", eps[:2])
	}
}

func TestFastAPI_WebSocket(t *testing.T) {
	src := `
from fastapi import FastAPI

app = FastAPI()

@app.websocket("/ws/live")
async def ws_live():
    pass
`
	p := NewFastAPI(".")
	eps := p.parseFile(src)
	if len(eps) != 1 || eps[0].Method != "WEBSOCKET" || eps[0].URI != "/ws/live" {
		t.Fatalf("got %+v", eps)
	}
}

func TestFastAPI_DependsMiddleware(t *testing.T) {
	src := `
from fastapi import FastAPI, Depends

app = FastAPI()

@app.get("/secure", dependencies=[Depends(require_token)])
async def secure():
    pass

@app.post("/admin", dependencies=[Depends(auth), Depends(role_admin)])
def admin():
    pass
`
	p := NewFastAPI(".")
	eps := p.parseFile(src)
	if len(eps) != 2 {
		t.Fatalf("hay %d", len(eps))
	}
	if len(eps[0].Middleware) != 1 || eps[0].Middleware[0] != "require_token" {
		t.Errorf("mw0: %v", eps[0].Middleware)
	}
	if len(eps[1].Middleware) != 2 {
		t.Errorf("mw1: %v", eps[1].Middleware)
	}
}

func TestFastAPI_SkipsTestClient(t *testing.T) {
	src := `
from fastapi import FastAPI
from fastapi.testclient import TestClient

app = FastAPI()

@client.get("/should-skip")
def x():
    pass

@app.get("/ok")
def ok():
    pass
`
	p := NewFastAPI(".")
	eps := p.parseFile(src)
	if len(eps) != 1 || eps[0].URI != "/ok" {
		t.Fatalf("got %+v", eps)
	}
}

func TestFastAPI_MultilineDecorator(t *testing.T) {
	src := `
from fastapi import FastAPI

app = FastAPI()

@app.get(
    "/long/path",
    summary="s",
)
async def long_path():
    pass
`
	p := NewFastAPI(".")
	eps := p.parseFile(src)
	if len(eps) != 1 || eps[0].URI != "/long/path" {
		t.Fatalf("got %+v", eps)
	}
}

func TestFastAPI_ParseSections_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.py")
	content := `
from fastapi import FastAPI
app = FastAPI()
@app.get("/ping")
def ping():
    return {"ok": true}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	p := NewFastAPI(dir)
	secs, err := p.ParseSections([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 1 || len(secs[0].Endpoints) != 1 {
		t.Fatalf("sections %+v", secs)
	}
	if secs[0].Endpoints[0].Language != "python" {
		t.Errorf("language: %q", secs[0].Endpoints[0].Language)
	}
}

func TestFastAPI_DeduplicateSameRoute(t *testing.T) {
	src := `
from fastapi import FastAPI
app = FastAPI()
@app.get("/dup")
def a():
    pass
@app.get("/dup")
def b():
    pass
`
	p := NewFastAPI(".")
	eps := p.parseFile(src)
	if len(eps) != 1 {
		t.Fatalf("dedupe: hay %d", len(eps))
	}
}
