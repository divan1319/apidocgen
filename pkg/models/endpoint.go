package models

// Endpoint is the language-agnostic representation extracted by any parser.
// The AI layer uses RawSource to infer what the parser couldn't determine statically.
type Endpoint struct {
	Method      string            // GET, POST, PUT, DELETE, PATCH
	URI         string            // /api/users/{id}
	Name        string            // route name if available
	Controller  string            // App\Http\Controllers\UserController
	Action      string            // store, update, show...
	Middleware  []string          // auth, throttle:60,1 ...
	RawSource   string            // raw source code of controller method + request class
	StaticMeta  StaticMeta        // what the parser could infer without AI
}

// StaticMeta holds everything the parser could determine without AI assistance.
type StaticMeta struct {
	RequestParams  []Param  // extracted from Form Request or inline validate()
	ResponseCodes  []int    // HTTP codes found in return response()->json(...)
	Description    string   // from PHPDoc/docblock if present
}

// Param represents a single input parameter to an endpoint.
type Param struct {
	Name     string // field name
	Type     string // string, integer, boolean, array, file...
	Required bool
	Rules    string // raw validation rules: "required|email|max:255"
}

// EndpointDoc is what Claude returns after analyzing an Endpoint.
// This is what gets rendered into HTML.
type EndpointDoc struct {
	Endpoint    Endpoint
	Summary     string        // short one-liner
	Description string        // fuller explanation
	Parameters  []ParamDoc
	Responses   []ResponseDoc
	Example     RequestExample
}

type ParamDoc struct {
	Param
	Description string
}

type ResponseDoc struct {
	Code        int
	Description string
	Body        string // JSON example
}

type RequestExample struct {
	Headers map[string]string
	Body    string // JSON
}
