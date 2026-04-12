package dotnet

import (
	"testing"
)

// ── Class declaration variants ───────────────────────────────────────────────

func TestClass_SealedModifier(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public sealed class UsersController : ControllerBase
{
    [HttpGet]
    public IActionResult GetAll() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 1 {
		t.Fatalf("sealed class: expected 1 endpoint, got %d", len(eps))
	}
	if eps[0].URI != "/api/users" {
		t.Errorf("URI: got %q, want /api/users", eps[0].URI)
	}
}

func TestClass_AbstractController_Skipped(t *testing.T) {
	src := `
[ApiController]
public abstract class BaseApiController : ControllerBase
{
    [HttpGet("health")]
    public IActionResult Health() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	// abstract controllers should still be parsed — concrete subclasses inherit routes
	// but the endpoints declared directly on abstract are valid
	if len(eps) != 1 {
		t.Fatalf("abstract class: expected 1 endpoint, got %d", len(eps))
	}
}

func TestClass_InternalModifier(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
internal class InternalController : ControllerBase
{
    [HttpGet]
    public IActionResult GetAll() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 1 {
		t.Fatalf("internal class: expected 1 endpoint, got %d", len(eps))
	}
}

func TestClass_PartialModifier(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public partial class ItemsController : ControllerBase
{
    [HttpGet]
    public IActionResult GetAll() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 1 {
		t.Fatalf("partial class: expected 1 endpoint, got %d", len(eps))
	}
}

func TestClass_GenericController(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public class CrudController<T> : ControllerBase where T : class
{
    [HttpGet]
    public IActionResult GetAll() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 1 {
		t.Fatalf("generic class: expected 1 endpoint, got %d", len(eps))
	}
}

func TestClass_InheritsCustomBase(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public class OrdersController : BaseApiController
{
    [HttpGet]
    public IActionResult GetAll() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	// BaseApiController doesn't contain "ControllerBase" or "Controller" directly
	// BUT [ApiController] attribute should be enough
	if len(eps) != 1 {
		t.Fatalf("custom base with [ApiController]: expected 1 endpoint, got %d", len(eps))
	}
}

func TestClass_NoApiControllerAttr_InheritsController(t *testing.T) {
	src := `
[Route("api/[controller]")]
public class OldController : Controller
{
    [HttpGet]
    public IActionResult GetAll() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 1 {
		t.Fatalf("inherits Controller (no [ApiController]): expected 1 endpoint, got %d", len(eps))
	}
}

// ── Method signature variants ────────────────────────────────────────────────

func TestMethod_OverrideKeyword(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public class ItemsController : ControllerBase
{
    [HttpGet]
    public override async Task<IActionResult> GetAll() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 1 {
		t.Fatalf("override method: expected 1 endpoint, got %d", len(eps))
	}
}

func TestMethod_ReturnsPlainType(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public class ValuesController : ControllerBase
{
    [HttpGet]
    public string Get() { return "hello"; }

    [HttpGet("list")]
    public List<string> GetList() { return new(); }

    [HttpGet("count")]
    public int GetCount() { return 42; }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 3 {
		t.Fatalf("plain return types: expected 3 endpoints, got %d", len(eps))
	}
}

func TestMethod_NonAction_Skipped(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public class ItemsController : ControllerBase
{
    [HttpGet]
    public IActionResult GetAll() { return Ok(); }

    [NonAction]
    public IActionResult Helper() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 1 {
		t.Fatalf("[NonAction]: expected 1 endpoint (Helper skipped), got %d", len(eps))
	}
}

func TestMethod_StaticMethod_Skipped(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public class ItemsController : ControllerBase
{
    [HttpGet]
    public IActionResult GetAll() { return Ok(); }

    public static string Utility() { return "test"; }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	// static method has no [Http*] attribute so it's already skipped
	if len(eps) != 1 {
		t.Fatalf("static method: expected 1 endpoint, got %d", len(eps))
	}
}

// ── HTTP verb attribute variants ─────────────────────────────────────────────

func TestHttpVerb_WithNamedParams(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public class ItemsController : ControllerBase
{
    [HttpGet("{id}", Name = "GetItem")]
    public IActionResult Get(int id) { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 1 {
		t.Fatalf("[HttpGet with Name]: expected 1 endpoint, got %d", len(eps))
	}
	if eps[0].URI != "/api/items/{id}" {
		t.Errorf("URI: got %q, want /api/items/{id}", eps[0].URI)
	}
}

func TestHttpVerb_AbsoluteRoute(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public class ItemsController : ControllerBase
{
    [HttpGet("~/api/v2/special")]
    public IActionResult Special() { return Ok(); }

    [HttpGet]
    public IActionResult GetAll() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 2 {
		t.Fatalf("absolute route: expected 2 endpoints, got %d", len(eps))
	}
	if eps[0].URI != "/api/v2/special" {
		t.Errorf("absolute URI: got %q, want /api/v2/special", eps[0].URI)
	}
	if eps[1].URI != "/api/items" {
		t.Errorf("normal URI: got %q, want /api/items", eps[1].URI)
	}
}

func TestHttpVerb_MultipleVerbsOnMethod(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public class ItemsController : ControllerBase
{
    [HttpGet]
    [HttpHead]
    public IActionResult GetAll() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 2 {
		t.Fatalf("multiple verbs: expected 2 endpoints (GET + HEAD), got %d", len(eps))
	}
}

// ── Route/URI patterns ───────────────────────────────────────────────────────

func TestRoute_ActionPlaceholder(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]/[action]")]
public class ReportsController : ControllerBase
{
    [HttpGet]
    public IActionResult Summary() { return Ok(); }

    [HttpGet]
    public IActionResult Details() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 2 {
		t.Fatalf("[action] placeholder: expected 2 endpoints, got %d", len(eps))
	}
	if eps[0].URI != "/api/reports/Summary" {
		t.Errorf("[0] URI: got %q, want /api/reports/Summary", eps[0].URI)
	}
	if eps[1].URI != "/api/reports/Details" {
		t.Errorf("[1] URI: got %q, want /api/reports/Details", eps[1].URI)
	}
}

func TestRoute_VersionedRoute(t *testing.T) {
	src := `
[ApiController]
[Route("api/v{version:apiVersion}/[controller]")]
public class ProductsController : ControllerBase
{
    [HttpGet]
    public IActionResult GetAll() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 1 {
		t.Fatalf("versioned route: expected 1 endpoint, got %d", len(eps))
	}
	if eps[0].URI != "/api/v{version:apiVersion}/products" {
		t.Errorf("URI: got %q, want /api/v{version:apiVersion}/products", eps[0].URI)
	}
}

// ── Parameter patterns ───────────────────────────────────────────────────────

func TestParam_DefaultValues(t *testing.T) {
	params := parseMethodParams("[FromQuery] int page = 1, [FromQuery] int size = 10")
	if len(params) != 2 {
		t.Fatalf("default params: expected 2, got %d", len(params))
	}
	if params[0].name != "page" {
		t.Errorf("[0] name: got %q, want page", params[0].name)
	}
	if params[1].name != "size" {
		t.Errorf("[1] name: got %q, want size", params[1].name)
	}
}

func TestParam_NullableTypes(t *testing.T) {
	params := parseMethodParams("int? id, string? name, DateOnly? from")
	if len(params) != 3 {
		t.Fatalf("nullable params: expected 3, got %d", len(params))
	}
	if params[0].typeName != "int?" {
		t.Errorf("[0] type: got %q, want int?", params[0].typeName)
	}
}

// ── ProducesResponseType extraction ──────────────────────────────────────────

func TestMethod_ProducesResponseType(t *testing.T) {
	src := `
[ApiController]
[Route("api/[controller]")]
public class ItemsController : ControllerBase
{
    [HttpGet]
    [ProducesResponseType(typeof(List<ItemDto>), 200)]
    [ProducesResponseType(404)]
    public IActionResult GetAll() { return Ok(); }
}
`
	p := New(".")
	eps := p.parseControllerFile(src)
	if len(eps) != 1 {
		t.Fatalf("ProducesResponseType: expected 1 endpoint, got %d", len(eps))
	}
}
