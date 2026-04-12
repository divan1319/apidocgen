package models

type RouteSection struct {
	Name      string
	FilePath  string
	Version   string
	Endpoints []Endpoint
}

type SectionDoc struct {
	Name     string
	Version  string
	FilePath string
	Docs     []EndpointDoc
}

type Endpoint struct {
	Method     string
	URI        string
	Name       string
	Controller string
	Action     string
	Middleware []string
	RawSource  string
	StaticMeta StaticMeta
	Language   string // "php", "csharp", etc. — used for code fences in AI prompts
}

type StaticMeta struct {
	RequestParams []Param
	ResponseCodes []int
	Description   string
}

type Param struct {
	Name     string
	Type     string
	Required bool
	Rules    string
}

type EndpointDoc struct {
	Endpoint    Endpoint
	Summary     string
	Description string
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
	Body        string
}

type RequestExample struct {
	Headers map[string]string
	Body    string
}
