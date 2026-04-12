package dotnet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseControllerFile_BasicCRUD(t *testing.T) {
	src := `
using Microsoft.AspNetCore.Mvc;

namespace MyApp.Controllers;

[ApiController]
[Route("api/[controller]")]
public class UsersController : ControllerBase
{
    [HttpGet]
    public async Task<IActionResult> GetAll()
    {
        return Ok();
    }

    [HttpGet("{id}")]
    public async Task<IActionResult> Get(int id)
    {
        return Ok();
    }

    [HttpPost]
    public async Task<IActionResult> Create([FromBody] CreateUserDto dto)
    {
        return Created();
    }

    [HttpPut("{id}")]
    public async Task<IActionResult> Update(int id, [FromBody] UpdateUserDto dto)
    {
        return Ok();
    }

    [HttpDelete("{id}")]
    public async Task<IActionResult> Delete(int id)
    {
        return NoContent();
    }
}
`
	p := New(".")
	endpoints := p.parseControllerFile(src)

	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d", len(endpoints))
	}

	tests := []struct {
		method string
		uri    string
		action string
	}{
		{"GET", "/api/users", "GetAll"},
		{"GET", "/api/users/{id}", "Get"},
		{"POST", "/api/users", "Create"},
		{"PUT", "/api/users/{id}", "Update"},
		{"DELETE", "/api/users/{id}", "Delete"},
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
		if ep.Language != "csharp" {
			t.Errorf("[%d] language: got %q, want %q", i, ep.Language, "csharp")
		}
		if ep.Controller != "UsersController" {
			t.Errorf("[%d] controller: got %q, want %q", i, ep.Controller, "UsersController")
		}
	}
}

func TestParseControllerFile_Authorize(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
[Authorize]
public class AdminController : ControllerBase
{
    [HttpGet]
    public IActionResult GetAll()
    {
        return Ok();
    }

    [HttpGet("public")]
    [AllowAnonymous]
    public IActionResult GetPublic()
    {
        return Ok();
    }

    [HttpPost]
    [Authorize(Roles = "Admin")]
    public IActionResult Create([FromBody] CreateDto dto)
    {
        return Ok();
    }
}
`
	p := New(".")
	endpoints := p.parseControllerFile(src)

	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}

	// GetAll inherits [Authorize] from class
	if len(endpoints[0].Middleware) != 1 || endpoints[0].Middleware[0] != "Authorize" {
		t.Errorf("GetAll middleware: got %v, want [Authorize]", endpoints[0].Middleware)
	}

	// GetPublic has [AllowAnonymous] which overrides class auth
	if len(endpoints[1].Middleware) != 1 || endpoints[1].Middleware[0] != "AllowAnonymous" {
		t.Errorf("GetPublic middleware: got %v, want [AllowAnonymous]", endpoints[1].Middleware)
	}

	// Create has method-level [Authorize(Roles = "Admin")]
	found := false
	for _, m := range endpoints[2].Middleware {
		if m == `Authorize(Roles = "Admin")` {
			found = true
		}
	}
	if !found {
		t.Errorf("Create middleware: got %v, want Authorize(Roles = \"Admin\")", endpoints[2].Middleware)
	}
}

func TestParseControllerFile_CustomRoute(t *testing.T) {
	src := `
[ApiController]
[Route("api/v1/products")]
public class ProductsController : ControllerBase
{
    [HttpGet("search")]
    public IActionResult Search([FromQuery] string query)
    {
        return Ok();
    }

    [HttpGet("{id:int}")]
    public IActionResult GetById(int id)
    {
        return Ok();
    }
}
`
	p := New(".")
	endpoints := p.parseControllerFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].URI != "/api/v1/products/search" {
		t.Errorf("Search URI: got %q, want %q", endpoints[0].URI, "/api/v1/products/search")
	}

	if endpoints[1].URI != "/api/v1/products/{id:int}" {
		t.Errorf("GetById URI: got %q, want %q", endpoints[1].URI, "/api/v1/products/{id:int}")
	}
}

func TestParseMinimalAPIs(t *testing.T) {
	src := `
var app = builder.Build();

app.MapGet("/api/items", () => Results.Ok());
app.MapPost("/api/items", (CreateItemDto dto) => Results.Created());
app.MapGet("/api/items/{id}", (int id) => Results.Ok());
app.MapPut("/api/items/{id}", (int id, UpdateItemDto dto) => Results.Ok());
app.MapDelete("/api/items/{id}", (int id) => Results.NoContent());
`
	p := New(".")
	endpoints := p.parseMinimalAPIs(src)

	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d", len(endpoints))
	}

	tests := []struct {
		method string
		uri    string
	}{
		{"GET", "/api/items"},
		{"POST", "/api/items"},
		{"GET", "/api/items/{id}"},
		{"PUT", "/api/items/{id}"},
		{"DELETE", "/api/items/{id}"},
	}

	for i, tt := range tests {
		if endpoints[i].Method != tt.method {
			t.Errorf("[%d] method: got %q, want %q", i, endpoints[i].Method, tt.method)
		}
		if endpoints[i].URI != tt.uri {
			t.Errorf("[%d] URI: got %q, want %q", i, endpoints[i].URI, tt.uri)
		}
		if endpoints[i].Language != "csharp" {
			t.Errorf("[%d] language: got %q, want %q", i, endpoints[i].Language, "csharp")
		}
	}
}

func TestParseMinimalAPIs_MapGroup(t *testing.T) {
	src := `
var app = builder.Build();

var usersGroup = app.MapGroup("/api/users");
usersGroup.MapGet("/", () => Results.Ok());
usersGroup.MapGet("/{id}", (int id) => Results.Ok());
usersGroup.MapPost("/", (CreateUserDto dto) => Results.Created());
`
	p := New(".")
	endpoints := p.parseMinimalAPIs(src)

	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}

	expected := []string{"/api/users", "/api/users/{id}", "/api/users"}
	for i, uri := range expected {
		if endpoints[i].URI != uri {
			t.Errorf("[%d] URI: got %q, want %q", i, endpoints[i].URI, uri)
		}
	}
}

func TestParseMinimalAPIs_RequireAuthorization(t *testing.T) {
	src := `
app.MapGet("/api/secret", () => Results.Ok())
    .RequireAuthorization();

app.MapGet("/api/public", () => Results.Ok())
    .AllowAnonymous();
`
	p := New(".")
	endpoints := p.parseMinimalAPIs(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if len(endpoints[0].Middleware) != 1 || endpoints[0].Middleware[0] != "Authorize" {
		t.Errorf("secret middleware: got %v, want [Authorize]", endpoints[0].Middleware)
	}

	if len(endpoints[1].Middleware) != 1 || endpoints[1].Middleware[0] != "AllowAnonymous" {
		t.Errorf("public middleware: got %v, want [AllowAnonymous]", endpoints[1].Middleware)
	}
}

func TestBuildURI(t *testing.T) {
	tests := []struct {
		classRoute     string
		methodRoute    string
		controllerName string
		actionName     string
		want           string
	}{
		{"api/[controller]", "", "UsersController", "GetAll", "/api/users"},
		{"api/[controller]", "{id}", "UsersController", "Get", "/api/users/{id}"},
		{"api/v1/products", "search", "ProductsController", "Search", "/api/v1/products/search"},
		{"api/[controller]", "{id}/details", "OrdersController", "Details", "/api/orders/{id}/details"},
		{"", "", "TestController", "Get", "/"},
		{"", "custom", "TestController", "Custom", "/custom"},
		{"api/[controller]/[action]", "", "ReportsController", "Summary", "/api/reports/Summary"},
		{"api/[controller]", "~/api/v2/override", "ItemsController", "Override", "/api/v2/override"},
	}

	for _, tt := range tests {
		got := buildURI(tt.classRoute, tt.methodRoute, tt.controllerName, tt.actionName)
		if got != tt.want {
			t.Errorf("buildURI(%q, %q, %q, %q) = %q, want %q",
				tt.classRoute, tt.methodRoute, tt.controllerName, tt.actionName, got, tt.want)
		}
	}
}

func TestMapCSharpType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"int", "integer"},
		{"int?", "integer"},
		{"string", "string"},
		{"bool", "boolean"},
		{"Guid", "string (uuid)"},
		{"DateTime", "string (datetime)"},
		{"List<string>", "array"},
		{"IFormFile", "file"},
		{"CreateUserDto", "object"},
		{"decimal", "number"},
	}

	for _, tt := range tests {
		got := mapCSharpType(tt.input)
		if got != tt.want {
			t.Errorf("mapCSharpType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeHTTPVerb(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"HttpGet", "GET"},
		{"HttpPost", "POST"},
		{"HttpPut", "PUT"},
		{"HttpPatch", "PATCH"},
		{"HttpDelete", "DELETE"},
		{"HttpOptions", "OPTIONS"},
		{"HttpHead", "HEAD"},
	}

	for _, tt := range tests {
		got := normalizeHTTPVerb(tt.input)
		if got != tt.want {
			t.Errorf("normalizeHTTPVerb(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveIncludes_Directory(t *testing.T) {
	dir := t.TempDir()

	// Create some .cs files
	os.MkdirAll(filepath.Join(dir, "Controllers"), 0755)
	os.WriteFile(filepath.Join(dir, "Controllers", "UsersController.cs"), []byte("// test"), 0644)
	os.WriteFile(filepath.Join(dir, "Controllers", "OrdersController.cs"), []byte("// test"), 0644)
	os.WriteFile(filepath.Join(dir, "NotAController.txt"), []byte("// skip"), 0644)

	p := New(dir)
	files, err := p.ResolveIncludes([]string{filepath.Join(dir, "Controllers")})
	if err != nil {
		t.Fatalf("ResolveIncludes: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
}

func TestParseMethodParams(t *testing.T) {
	tests := []struct {
		raw  string
		want []paramInfo
	}{
		{
			"int id",
			[]paramInfo{{name: "id", typeName: "int", from: ""}},
		},
		{
			"[FromBody] CreateUserDto dto",
			[]paramInfo{{name: "dto", typeName: "CreateUserDto", from: "Body"}},
		},
		{
			"int id, [FromBody] UpdateDto dto",
			[]paramInfo{
				{name: "id", typeName: "int", from: ""},
				{name: "dto", typeName: "UpdateDto", from: "Body"},
			},
		},
		{
			"[FromQuery] string search, [FromQuery] int page",
			[]paramInfo{
				{name: "search", typeName: "string", from: "Query"},
				{name: "page", typeName: "int", from: "Query"},
			},
		},
	}

	for i, tt := range tests {
		got := parseMethodParams(tt.raw)
		if len(got) != len(tt.want) {
			t.Errorf("[%d] parseMethodParams(%q): got %d params, want %d", i, tt.raw, len(got), len(tt.want))
			continue
		}
		for j := range got {
			if got[j].name != tt.want[j].name {
				t.Errorf("[%d][%d] name: got %q, want %q", i, j, got[j].name, tt.want[j].name)
			}
			if got[j].typeName != tt.want[j].typeName {
				t.Errorf("[%d][%d] type: got %q, want %q", i, j, got[j].typeName, tt.want[j].typeName)
			}
			if got[j].from != tt.want[j].from {
				t.Errorf("[%d][%d] from: got %q, want %q", i, j, got[j].from, tt.want[j].from)
			}
		}
	}
}

func TestStripCSharpComments(t *testing.T) {
	src := `
// This is a comment
[HttpGet] // inline comment
public IActionResult Get()
{
    /* block comment */
    return Ok();
}
`
	result := stripCSharpComments(src)
	if containsString(result, "This is a comment") {
		t.Error("line comment not stripped")
	}
	if containsString(result, "inline comment") {
		t.Error("inline comment not stripped")
	}
	if containsString(result, "block comment") {
		t.Error("block comment not stripped")
	}
	if !containsString(result, "[HttpGet]") {
		t.Error("[HttpGet] should be preserved")
	}
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestParseControllerFile_PrimaryConstructor(t *testing.T) {
	src := `
using Microsoft.AspNetCore.Mvc;

namespace MyApp.Controllers;

[ApiController]
[Route("api/[controller]")]
public class CategoriesController(AppDbContext db) : ControllerBase
{
    [HttpGet]
    public async Task<ActionResult<IEnumerable<CategoryResponse>>> GetAll(CancellationToken cancellationToken)
    {
        return Ok(list);
    }

    [HttpGet("{id:int}")]
    public async Task<ActionResult<CategoryResponse>> GetById(int id, CancellationToken cancellationToken)
    {
        return Ok(item);
    }

    [HttpPost]
    public async Task<ActionResult<CategoryResponse>> Create(
        [FromBody] CategoryCreateRequest request,
        CancellationToken cancellationToken)
    {
        return CreatedAtAction(nameof(GetById), new { id = 1 }, response);
    }

    [HttpPut("{id:int}")]
    public async Task<IActionResult> Update(int id, [FromBody] CategoryUpdateRequest request, CancellationToken cancellationToken)
    {
        return NoContent();
    }

    [HttpDelete("{id:int}")]
    public async Task<IActionResult> Delete(int id, CancellationToken cancellationToken)
    {
        return NoContent();
    }
}
`
	p := New(".")
	endpoints := p.parseControllerFile(src)

	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d", len(endpoints))
	}

	tests := []struct {
		method string
		uri    string
		action string
	}{
		{"GET", "/api/categories", "GetAll"},
		{"GET", "/api/categories/{id:int}", "GetById"},
		{"POST", "/api/categories", "Create"},
		{"PUT", "/api/categories/{id:int}", "Update"},
		{"DELETE", "/api/categories/{id:int}", "Delete"},
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
	}

	// Verify multi-line params: Create should have a FromBody param
	createEp := endpoints[2]
	foundBody := false
	for _, p := range createEp.StaticMeta.RequestParams {
		if p.Name == "request" && p.Rules == "FromBody" {
			foundBody = true
		}
	}
	if !foundBody {
		t.Errorf("Create: expected FromBody param 'request', got %+v", createEp.StaticMeta.RequestParams)
	}
}

func TestParseControllerFile_ActionResult_Generic(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public class ItemsController : ControllerBase
{
    [HttpGet]
    public async Task<ActionResult<IEnumerable<ItemDto>>> GetAll()
    {
        return Ok(items);
    }

    [HttpGet("{id}")]
    public async Task<ActionResult<ItemDto>> GetById(int id)
    {
        return Ok(item);
    }
}
`
	p := New(".")
	endpoints := p.parseControllerFile(src)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	if endpoints[0].Method != "GET" || endpoints[0].URI != "/api/items" {
		t.Errorf("GetAll: got %s %s", endpoints[0].Method, endpoints[0].URI)
	}
	if endpoints[1].Method != "GET" || endpoints[1].URI != "/api/items/{id}" {
		t.Errorf("GetById: got %s %s", endpoints[1].Method, endpoints[1].URI)
	}
}
