package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFlask_BasicRoute(t *testing.T) {
	src := `
from flask import Flask

app = Flask(__name__)

@app.route("/users")
def users():
    pass

@app.route("/items", methods=["POST"])
def create_item():
    pass

@app.route("/mixed", methods=["GET", "PUT"])
def mixed():
    pass
`
	p := NewFlask(".")
	eps := p.parseFile(src)
	if len(eps) != 4 {
		t.Fatalf("esperaba 4, hay %d", len(eps))
	}
	if eps[0].Method != "GET" || eps[0].URI != "/users" {
		t.Errorf("0: %+v", eps[0])
	}
	if eps[1].Method != "POST" || eps[1].URI != "/items" {
		t.Errorf("1: %+v", eps[1])
	}
}

func TestFlask_VerbShorthand(t *testing.T) {
	src := `
from flask import Flask

app = Flask(__name__)

@app.get("/api/hello")
def hello():
    pass

@app.post("/api/hello")
def hello_post():
    pass

@app.delete("/api/hello/<int:idx>")
def hello_del(idx):
    pass
`
	p := NewFlask(".")
	eps := p.parseFile(src)
	if len(eps) != 3 {
		t.Fatalf("hay %d", len(eps))
	}
	if eps[0].URI != "/api/hello" || eps[0].Method != "GET" {
		t.Errorf("%+v", eps[0])
	}
	if eps[2].URI != "/api/hello/<int:idx>" {
		t.Errorf("%+v", eps[2])
	}
}

func TestFlask_BlueprintUrlPrefix(t *testing.T) {
	src := `
from flask import Flask, Blueprint

app = Flask(__name__)
bp = Blueprint("admin", __name__, url_prefix="/admin")

@bp.route("/dash")
def dash():
    pass

@bp.route("/users/<user_id>")
def user(user_id):
    pass
`
	p := NewFlask(".")
	eps := p.parseFile(src)
	if len(eps) != 2 {
		t.Fatalf("hay %d", len(eps))
	}
	if eps[0].URI != "/admin/dash" {
		t.Errorf("URI0: %q", eps[0].URI)
	}
	if eps[1].URI != "/admin/users/<user_id>" {
		t.Errorf("URI1: %q", eps[1].URI)
	}
}

func TestFlask_RegisterBlueprintStackedPrefix(t *testing.T) {
	src := `
from flask import Flask, Blueprint

app = Flask(__name__)
api = Blueprint("api", __name__, url_prefix="/v1")

@api.get("/status")
def status():
    pass

app.register_blueprint(api, url_prefix="/service")
`
	p := NewFlask(".")
	eps := p.parseFile(src)
	if len(eps) != 1 {
		t.Fatalf("hay %d", len(eps))
	}
	if eps[0].URI != "/service/v1/status" {
		t.Errorf("URI: %q", eps[0].URI)
	}
}

func TestFlask_AddURLRule(t *testing.T) {
	src := `
from flask import Flask

app = Flask(__name__)

app.add_url_rule("/legacy", view_func=legacy, methods=["GET", "POST"])
`
	p := NewFlask(".")
	eps := p.parseFile(src)
	if len(eps) != 2 {
		t.Fatalf("hay %d", len(eps))
	}
	for _, e := range eps {
		if e.URI != "/legacy" {
			t.Errorf("URI %q", e.URI)
		}
		if e.Method != "GET" && e.Method != "POST" {
			t.Errorf("método %q", e.Method)
		}
	}
}

func TestFlask_SkipsTestReceiver(t *testing.T) {
	src := `
from flask import Flask
app = Flask(__name__)

@client.get("/skip")
def s():
    pass

@app.get("/real")
def r():
    pass
`
	p := NewFlask(".")
	eps := p.parseFile(src)
	if len(eps) != 1 || eps[0].URI != "/real" {
		t.Fatalf("%+v", eps)
	}
}

func TestFlask_ParseSections_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "web.py")
	content := `
from flask import Flask
app = Flask(__name__)
@app.route("/health")
def health():
    return "ok"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	p := NewFlask(dir)
	secs, err := p.ParseSections([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 1 || len(secs[0].Endpoints) != 1 {
		t.Fatalf("%+v", secs)
	}
	if secs[0].Endpoints[0].Language != "python" {
		t.Errorf("lang %q", secs[0].Endpoints[0].Language)
	}
}

func TestFlask_RouteMethodsTuple(t *testing.T) {
	src := `
from flask import Flask
app = Flask(__name__)

@app.route("/t", methods=("DELETE",))
def only_del():
    pass
`
	p := NewFlask(".")
	eps := p.parseFile(src)
	if len(eps) != 1 || eps[0].Method != "DELETE" {
		t.Fatalf("%+v", eps)
	}
}
