package dotnet

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/pkg/models"
)

func init() {
	parser.Register("dotnet", func(root string) parser.Parser {
		return New(root)
	})
}

type DotnetParser struct {
	projectRoot string
}

func New(projectRoot string) *DotnetParser {
	return &DotnetParser{projectRoot: projectRoot}
}

func (p *DotnetParser) Language() string { return "dotnet" }

// ResolveIncludes expands directories into individual .cs files recursively.
// For .NET, each entry in files can be a .cs file or a directory.
func (p *DotnetParser) ResolveIncludes(files []string) ([]string, error) {
	var all []string
	seen := map[string]bool{}

	for _, f := range files {
		resolved, err := p.resolveEntry(f)
		if err != nil {
			return nil, fmt.Errorf("resolving %s: %w", f, err)
		}
		for _, r := range resolved {
			abs, _ := filepath.Abs(r)
			if !seen[abs] {
				seen[abs] = true
				all = append(all, r)
			}
		}
	}
	return all, nil
}

func (p *DotnetParser) resolveEntry(path string) ([]string, error) {
	path = p.resolveUnderRoot(path)
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}

	var files []string
	err = filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() && strings.HasSuffix(fi.Name(), ".cs") {
			files = append(files, p)
		}
		return nil
	})
	return files, err
}

// resolveUnderRoot joins relative paths to projectRoot so --routes Controllers/
// works when the shell cwd is not the .NET project (matches CLI --root).
func (p *DotnetParser) resolveUnderRoot(path string) string {
	if filepath.IsAbs(path) || p.projectRoot == "" {
		return path
	}
	return filepath.Join(p.projectRoot, path)
}

// ParseSections parses each .cs file and returns one RouteSection per file
// that contains at least one endpoint.
func (p *DotnetParser) ParseSections(files []string) ([]models.RouteSection, error) {
	var sections []models.RouteSection

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f, err)
		}

		src := string(content)
		var endpoints []models.Endpoint

		if isControllerFile(src) {
			endpoints = p.parseControllerFile(src)
		}

		endpoints = append(endpoints, p.parseMinimalAPIs(src)...)

		if len(endpoints) == 0 {
			continue
		}

		sections = append(sections, models.RouteSection{
			FilePath:  f,
			Version:   inferVersion(f),
			Endpoints: endpoints,
		})
	}

	return sections, nil
}

func inferVersion(path string) string {
	re := regexp.MustCompile(`(?i)[/\\](v\d+)[/\\]`)
	m := re.FindStringSubmatch(path)
	if m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// ── Controller-based routing ─────────────────────────────────────────────────

var (
	classPattern = regexp.MustCompile(
		`(?s)\[ApiController\].*?(?:\[Route\("([^"]+)"\)\])?\s*` +
			`(?:public\s+)?class\s+(\w+)\s*(?::\s*[\w.,<>\s]+)?\s*\{`)

	// Matches [Route("...")] that appears right before a class declaration
	routeAttrPattern = regexp.MustCompile(`\[Route\("([^"]+)"\)\]`)

	httpMethodPattern = regexp.MustCompile(
		`\[(Http(?:Get|Post|Put|Patch|Delete|Options|Head))(?:\("([^"]*)"[^)]*\))?\]`)

	authorizePattern  = regexp.MustCompile(`\[Authorize(?:\(([^)]*)\))?\]`)
	allowAnonPattern  = regexp.MustCompile(`\[AllowAnonymous\]`)
	producesPattern   = regexp.MustCompile(`\[Produces\("([^"]+)"\)\]`)
	consumesPattern   = regexp.MustCompile(`\[Consumes\("([^"]+)"\)\]`)

	methodSigPattern = regexp.MustCompile(
		`(?:public|internal)\s+(?:async\s+)?(?:(?:Task|ValueTask)<)?(?:IActionResult|ActionResult(?:<[\w<>,\s]+>)?|[\w<>,\s]+)>?\s+(\w+)\s*\(([^)]*)\)`)

	fromAttrPattern = regexp.MustCompile(
		`\[(From(?:Body|Query|Route|Header|Form|Services))(?:\([^)]*\))?\]\s*`)

	dataAnnotationRequired = regexp.MustCompile(`\[Required\]`)
	dataAnnotationRange    = regexp.MustCompile(`\[Range\(([^)]+)\)\]`)
	dataAnnotationMaxLen   = regexp.MustCompile(`\[(?:MaxLength|StringLength)\((\d+)\)\]`)
)

func isControllerFile(src string) bool {
	return strings.Contains(src, "[ApiController]") ||
		strings.Contains(src, ": ControllerBase") ||
		strings.Contains(src, ": Controller")
}

func (p *DotnetParser) parseControllerFile(src string) []models.Endpoint {
	src = stripCSharpComments(src)

	classLocs := findAllClasses(src)
	var endpoints []models.Endpoint

	for _, cls := range classLocs {
		routeTemplate := cls.routeTemplate
		controllerName := cls.name

		classMiddleware := extractClassMiddleware(cls.prelude)

		methods := extractMethods(cls.body)
		for _, m := range methods {
			eps := p.buildEndpointsFromMethod(m, routeTemplate, controllerName, classMiddleware)
			endpoints = append(endpoints, eps...)
		}
	}

	return endpoints
}

type classInfo struct {
	name          string
	routeTemplate string
	prelude       string // attributes before class
	body          string // content between { }
}

func findAllClasses(src string) []classInfo {
	var classes []classInfo

	classRe := regexp.MustCompile(
		`(?:(?:public|internal)\s+)?(?:(?:sealed|abstract|partial|static)\s+)*class\s+(\w+)(?:<[^>]+>)?\s*(?:\([^)]*\))?\s*(?::\s*([\w.,<>\s]+))?(?:\s+where\s+[^{]+)?\s*\{`)

	matches := classRe.FindAllStringSubmatchIndex(src, -1)

	for _, loc := range matches {
		className := src[loc[2]:loc[3]]

		var baseClasses string
		if loc[4] >= 0 {
			baseClasses = src[loc[4]:loc[5]]
		}

		// Extract the prelude (attributes before the class declaration)
		prelude := extractPrelude(src, loc[0])

		isController := strings.Contains(prelude, "[ApiController]") ||
			strings.Contains(baseClasses, "ControllerBase") ||
			strings.Contains(baseClasses, "Controller")

		if !isController {
			continue
		}

		bodyStart := loc[1] // position right after '{'
		body := extractBraceBlock(src, bodyStart-1)

		routeTemplate := ""
		if rm := routeAttrPattern.FindStringSubmatch(prelude); rm != nil {
			routeTemplate = rm[1]
		}

		classes = append(classes, classInfo{
			name:          className,
			routeTemplate: routeTemplate,
			prelude:       prelude,
			body:          body,
		})
	}

	return classes
}

// extractPrelude looks backwards from pos to collect all [...] attributes
// that appear immediately before the class/method declaration.
func extractPrelude(src string, pos int) string {
	end := pos
	i := pos - 1

	for i >= 0 {
		// Skip whitespace backwards
		for i >= 0 && (src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r') {
			i--
		}
		if i < 0 || src[i] != ']' {
			break
		}
		// Find matching '['
		depth := 0
		for i >= 0 {
			if src[i] == ']' {
				depth++
			} else if src[i] == '[' {
				depth--
				if depth == 0 {
					break
				}
			}
			i--
		}
		if i < 0 {
			break
		}
		i-- // move past the '['
	}

	start := i + 1
	if start < 0 {
		start = 0
	}
	return strings.TrimSpace(src[start:end])
}

func extractBraceBlock(src string, openPos int) string {
	depth := 0
	start := -1
	for i := openPos; i < len(src); i++ {
		if src[i] == '{' {
			if depth == 0 {
				start = i + 1
			}
			depth++
		} else if src[i] == '}' {
			depth--
			if depth == 0 {
				return src[start:i]
			}
		}
	}
	if start >= 0 {
		return src[start:]
	}
	return ""
}

func extractClassMiddleware(prelude string) []string {
	var mw []string
	if m := authorizePattern.FindStringSubmatch(prelude); m != nil {
		if m[1] != "" {
			mw = append(mw, "Authorize("+m[1]+")")
		} else {
			mw = append(mw, "Authorize")
		}
	}
	return mw
}

type methodInfo struct {
	prelude    string // attributes before the method
	name       string
	paramsRaw  string
	body       string
	fullSource string
}

func extractMethods(classBody string) []methodInfo {
	var methods []methodInfo

	re := regexp.MustCompile(
		`(?s)((?:\[[^\]]+\]\s*)*?)` +
			`(?:public|internal)\s+(?:virtual\s+)?(?:async\s+)?` +
			`(?:(?:Task|ValueTask)<)?(?:IActionResult|ActionResult(?:<[\w<>,\s]+>)?|[\w<>,.\[\]\s]+)>?\s+` +
			`(\w+)\s*\(([^)]*)\)\s*\{`)

	matches := re.FindAllStringSubmatchIndex(classBody, -1)

	for _, loc := range matches {
		prelude := classBody[loc[2]:loc[3]]
		methodName := classBody[loc[4]:loc[5]]
		paramsRaw := classBody[loc[6]:loc[7]]

		if !hasHTTPVerbAttribute(prelude) {
			continue
		}

		bodyStart := loc[1] - 1 // position of '{'
		body := extractBraceBlock(classBody, bodyStart)

		fullStart := loc[0]
		fullEnd := bodyStart + 1 + len(body) + 1
		if fullEnd > len(classBody) {
			fullEnd = len(classBody)
		}
		fullSource := classBody[fullStart:fullEnd]

		methods = append(methods, methodInfo{
			prelude:    prelude,
			name:       methodName,
			paramsRaw:  paramsRaw,
			body:       body,
			fullSource: fullSource,
		})
	}

	return methods
}

func hasHTTPVerbAttribute(s string) bool {
	return httpMethodPattern.MatchString(s)
}

func (p *DotnetParser) buildEndpointsFromMethod(
	m methodInfo,
	classRoute, controllerName string,
	classMiddleware []string,
) []models.Endpoint {
	verbMatches := httpMethodPattern.FindAllStringSubmatch(m.prelude, -1)
	if len(verbMatches) == 0 {
		return nil
	}

	methodMiddleware := extractMethodMiddleware(m.prelude)
	allMiddleware := mergeMiddleware(classMiddleware, methodMiddleware)

	params := parseMethodParams(m.paramsRaw)
	staticMeta := p.extractStaticMeta(m.body, params)

	var endpoints []models.Endpoint
	for _, vm := range verbMatches {
		httpMethod := normalizeHTTPVerb(vm[1])
		routeTemplate := vm[2]

		uri := buildURI(classRoute, routeTemplate, controllerName, m.name)

		endpoints = append(endpoints, models.Endpoint{
			Method:     httpMethod,
			URI:        uri,
			Controller: controllerName,
			Action:     m.name,
			Middleware: allMiddleware,
			RawSource:  m.fullSource,
			StaticMeta: staticMeta,
			Language:   "csharp",
		})
	}

	return endpoints
}

func normalizeHTTPVerb(attr string) string {
	switch attr {
	case "HttpGet":
		return "GET"
	case "HttpPost":
		return "POST"
	case "HttpPut":
		return "PUT"
	case "HttpPatch":
		return "PATCH"
	case "HttpDelete":
		return "DELETE"
	case "HttpOptions":
		return "OPTIONS"
	case "HttpHead":
		return "HEAD"
	default:
		return strings.ToUpper(strings.TrimPrefix(attr, "Http"))
	}
}

func buildURI(classRoute, methodRoute, controllerName, actionName string) string {
	// ~/path overrides the class route entirely
	if strings.HasPrefix(methodRoute, "~/") {
		uri := methodRoute[1:] // strip the '~', keep the '/'
		return uri
	}

	if classRoute == "" && methodRoute == "" {
		return "/"
	}

	controllerSegment := strings.TrimSuffix(controllerName, "Controller")
	controllerSegment = strings.ToLower(controllerSegment)

	uri := classRoute
	uri = strings.ReplaceAll(uri, "[controller]", controllerSegment)
	uri = strings.ReplaceAll(uri, "[action]", actionName)

	if methodRoute != "" {
		uri = joinPath(uri, methodRoute)
	}

	if !strings.HasPrefix(uri, "/") {
		uri = "/" + uri
	}
	return uri
}

func extractMethodMiddleware(prelude string) []string {
	var mw []string

	if m := authorizePattern.FindStringSubmatch(prelude); m != nil {
		if m[1] != "" {
			mw = append(mw, "Authorize("+m[1]+")")
		} else {
			mw = append(mw, "Authorize")
		}
	}

	if allowAnonPattern.MatchString(prelude) {
		mw = append(mw, "AllowAnonymous")
	}

	if m := producesPattern.FindStringSubmatch(prelude); m != nil {
		mw = append(mw, "Produces("+m[1]+")")
	}
	if m := consumesPattern.FindStringSubmatch(prelude); m != nil {
		mw = append(mw, "Consumes("+m[1]+")")
	}

	return mw
}

func mergeMiddleware(class, method []string) []string {
	seen := map[string]bool{}
	var result []string

	for _, m := range method {
		if m == "AllowAnonymous" {
			return []string{"AllowAnonymous"}
		}
	}

	for _, m := range class {
		if !seen[m] {
			seen[m] = true
			result = append(result, m)
		}
	}
	for _, m := range method {
		if !seen[m] {
			seen[m] = true
			result = append(result, m)
		}
	}
	return result
}

type paramInfo struct {
	name     string
	typeName string
	from     string // "Body", "Query", "Route", "Header", "Form", ""
}

func parseMethodParams(raw string) []paramInfo {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parts := splitParams(raw)
	var params []paramInfo

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Strip default value: "int page = 1" → "int page"
		if eqIdx := strings.Index(part, "="); eqIdx >= 0 {
			// Make sure it's not inside an attribute like [Range(1,10)]
			beforeEq := part[:eqIdx]
			if !strings.Contains(beforeEq, "[") || strings.Contains(beforeEq, "]") {
				part = strings.TrimSpace(part[:eqIdx])
			}
		}

		from := ""
		if m := fromAttrPattern.FindStringSubmatch(part); m != nil {
			from = strings.TrimPrefix(m[1], "From")
			part = fromAttrPattern.ReplaceAllString(part, "")
			part = strings.TrimSpace(part)
		}

		tokens := strings.Fields(part)
		if len(tokens) < 2 {
			continue
		}

		typeName := tokens[len(tokens)-2]
		paramName := tokens[len(tokens)-1]

		// Remove attributes like [Required] from type
		typeName = regexp.MustCompile(`\[[\w(),"=\s]+\]\s*`).ReplaceAllString(typeName, "")

		params = append(params, paramInfo{
			name:     paramName,
			typeName: typeName,
			from:     from,
		})
	}

	return params
}

func splitParams(raw string) []string {
	var parts []string
	depth := 0
	start := 0

	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '<', '[':
			depth++
		case '>', ']':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, raw[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, raw[start:])
	return parts
}

func (p *DotnetParser) extractStaticMeta(body string, params []paramInfo) models.StaticMeta {
	meta := models.StaticMeta{}

	skipTypes := map[string]bool{
		"CancellationToken":  true,
		"HttpContext":        true,
		"HttpRequest":       true,
		"HttpResponse":      true,
		"ClaimsPrincipal":   true,
	}

	for _, param := range params {
		if param.from == "Services" {
			continue
		}
		baseType := strings.TrimSuffix(param.typeName, "?")
		if skipTypes[baseType] {
			continue
		}

		mp := models.Param{
			Name: param.name,
			Type: mapCSharpType(param.typeName),
		}

		if param.from == "Body" || param.from == "Form" {
			mp.Required = true
		}
		if param.from == "Route" {
			mp.Required = true
		}

		if param.from != "" {
			mp.Rules = "From" + param.from
		}

		meta.RequestParams = append(meta.RequestParams, mp)
	}

	// Also look for data annotation attributes in the body for DTO validation
	meta.RequestParams = append(meta.RequestParams, extractDTOValidation(body)...)

	return meta
}

func extractDTOValidation(body string) []models.Param {
	// Look for ModelState.IsValid checks to indicate validation is used
	if !strings.Contains(body, "ModelState") && !strings.Contains(body, "Validate") {
		return nil
	}
	return nil
}

func mapCSharpType(t string) string {
	t = strings.TrimSuffix(t, "?") // Remove nullable marker

	switch strings.ToLower(t) {
	case "int", "int32", "long", "int64", "short", "int16":
		return "integer"
	case "float", "double", "decimal":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "string":
		return "string"
	case "guid":
		return "string (uuid)"
	case "datetime", "datetimeoffset":
		return "string (datetime)"
	case "dateonly":
		return "string (date)"
	case "timeonly", "timespan":
		return "string (time)"
	case "iformfile":
		return "file"
	default:
		if strings.HasPrefix(t, "List<") || strings.HasPrefix(t, "IEnumerable<") ||
			strings.HasSuffix(t, "[]") || strings.HasPrefix(t, "ICollection<") {
			return "array"
		}
		if strings.HasPrefix(t, "Dictionary<") || strings.HasPrefix(t, "IDictionary<") {
			return "object"
		}
		return "object"
	}
}

// ── Minimal APIs ─────────────────────────────────────────────────────────────

var (
	mapMethodPattern = regexp.MustCompile(
		`(\w+)\.Map(Get|Post|Put|Patch|Delete)\(\s*"([^"]+)"`)

	mapGroupPattern = regexp.MustCompile(
		`(?:var\s+)?(\w+)\s*=\s*\w+\.MapGroup\(\s*"([^"]+)"\s*\)`)

	requireAuthPattern = regexp.MustCompile(`\.RequireAuthorization\(\)`)
	allowAnonAPIPattern = regexp.MustCompile(`\.AllowAnonymous\(\)`)
	withTagsPattern    = regexp.MustCompile(`\.WithTags\("([^"]+)"\)`)
)

func (p *DotnetParser) parseMinimalAPIs(src string) []models.Endpoint {
	src = stripCSharpComments(src)

	if !hasMinimalAPIs(src) {
		return nil
	}

	groups := parseMapGroups(src)

	var endpoints []models.Endpoint

	lines := strings.Split(src, "\n")
	for lineIdx, line := range lines {
		trimmed := strings.TrimSpace(line)

		m := mapMethodPattern.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}

		callerVar := m[1]
		httpMethod := strings.ToUpper(m[2])
		routePath := m[3]

		prefix := resolveGroupPrefixByVar(callerVar, groups)
		uri := joinPath(prefix, routePath)

		// Build the full statement (may span multiple lines)
		fullStmt := collectStatement(lines, lineIdx)

		var middleware []string
		if requireAuthPattern.MatchString(fullStmt) {
			middleware = append(middleware, "Authorize")
		}
		if allowAnonAPIPattern.MatchString(fullStmt) {
			middleware = append(middleware, "AllowAnonymous")
		}

		handler := extractLambdaOrMethod(fullStmt)

		endpoints = append(endpoints, models.Endpoint{
			Method:     httpMethod,
			URI:        uri,
			Controller: "MinimalAPI",
			Action:     extractHandlerName(fullStmt, httpMethod, routePath),
			Middleware: middleware,
			RawSource:  handler,
			Language:   "csharp",
		})
	}

	return endpoints
}

func hasMinimalAPIs(src string) bool {
	return strings.Contains(src, ".MapGet(") ||
		strings.Contains(src, ".MapPost(") ||
		strings.Contains(src, ".MapPut(") ||
		strings.Contains(src, ".MapPatch(") ||
		strings.Contains(src, ".MapDelete(")
}

type mapGroup struct {
	varName string
	prefix  string
}

func parseMapGroups(src string) []mapGroup {
	var groups []mapGroup
	for _, m := range mapGroupPattern.FindAllStringSubmatch(src, -1) {
		groups = append(groups, mapGroup{
			varName: m[1],
			prefix:  m[2],
		})
	}
	return groups
}

func resolveGroupPrefixByVar(varName string, groups []mapGroup) string {
	for _, g := range groups {
		if g.varName == varName {
			return g.prefix
		}
	}
	return ""
}

func collectStatement(lines []string, startIdx int) string {
	var sb strings.Builder
	for i := startIdx; i < len(lines); i++ {
		sb.WriteString(lines[i])
		sb.WriteString("\n")
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasSuffix(trimmed, ";") || strings.HasSuffix(trimmed, "});") {
			break
		}
	}
	return sb.String()
}

func extractLambdaOrMethod(stmt string) string {
	// Try to find the lambda body: => { ... } or => expression
	if idx := strings.Index(stmt, "=>"); idx >= 0 {
		return strings.TrimSpace(stmt[idx:])
	}
	return stmt
}

func extractHandlerName(stmt, method, path string) string {
	// Try named method reference: .MapGet("/path", MethodName)
	re := regexp.MustCompile(`Map\w+\([^,]+,\s*(\w+)\s*\)`)
	if m := re.FindStringSubmatch(stmt); m != nil {
		return m[1]
	}

	// Derive from method + path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var name string
	for _, part := range parts {
		if part == "" || strings.HasPrefix(part, "{") {
			continue
		}
		name += strings.Title(part)
	}
	if name == "" {
		name = "Root"
	}
	return method + "_" + name
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func stripCSharpComments(s string) string {
	s = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`//[^\n]*`).ReplaceAllString(s, "")
	return s
}

func joinPath(a, b string) string {
	a = strings.Trim(a, "/")
	b = strings.Trim(b, "/")
	if a == "" && b == "" {
		return "/"
	}
	if a == "" {
		return "/" + b
	}
	if b == "" {
		return "/" + a
	}
	return "/" + a + "/" + b
}
