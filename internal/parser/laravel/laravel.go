package laravel

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/pkg/models"
)

// ── Parser ───────────────────────────────────────────────────────────────────

type LaravelParser struct {
	projectRoot string
}

func New(projectRoot string) *LaravelParser {
	return &LaravelParser{projectRoot: projectRoot}
}

func (p *LaravelParser) Language() string { return "laravel" }

// ── Include resolution ───────────────────────────────────────────────────────

var includePattern = regexp.MustCompile(
	`(?:include|require)(?:_once)?\s*['"]([^'"]+)['"]`,
)

// resolveIncludes returns a flat list: first the file itself, then any includes
// it references (recursively). The main entry file is always first.
func (p *LaravelParser) resolveIncludes(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(filePath)
	files := []string{filePath}

	for _, m := range includePattern.FindAllStringSubmatch(string(content), -1) {
		included := filepath.Join(dir, m[1])
		nested, err := p.resolveIncludes(included)
		if err != nil {
			continue
		}
		files = append(files, nested...)
	}

	return files, nil
}

// ResolveIncludes returns the ordered list of files that will be parsed,
// so the CLI can prompt the user to name each section before parsing starts.
func (p *LaravelParser) ResolveIncludes(files []string) ([]string, error) {
	var all []string
	seen := map[string]bool{}
	for _, f := range files {
		resolved, err := p.resolveIncludes(f)
		if err != nil {
			return nil, fmt.Errorf("resolving includes for %s: %w", f, err)
		}
		for _, r := range resolved {
			if !seen[r] {
				seen[r] = true
				all = append(all, r)
			}
		}
	}
	return all, nil
}

// ── Entry point ──────────────────────────────────────────────────────────────

// ParseSections parses each file independently and returns one RouteSection per file.
// The caller is responsible for setting each section's Name (from CLI prompt).
func (p *LaravelParser) ParseSections(files []string) ([]models.RouteSection, error) {
	var sections []models.RouteSection

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f, err)
		}

		endpoints := p.parseRouteFile(string(content))

		// Skip files with no routes (e.g. the main api.php that only has includes)
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

// inferVersion tries to detect v1/v2/vN from a file path.
func inferVersion(path string) string {
	re := regexp.MustCompile(`(?i)[/\\](v\d+)[/\\]`)
	m := re.FindStringSubmatch(path)
	if m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// ── Group context (inherited as we descend) ───────────────────────────────────

type groupContext struct {
	prefix     string
	controller string
	middleware []string
}

func (g groupContext) merge(child groupContext) groupContext {
	merged := groupContext{
		prefix:     joinPath(g.prefix, child.prefix),
		controller: g.controller,
		middleware: append(append([]string{}, g.middleware...), child.middleware...),
	}
	if child.controller != "" {
		merged.controller = child.controller
	}
	return merged
}

// ── Chain segment ─────────────────────────────────────────────────────────────

type segment struct {
	name string
	args string // raw content inside the parentheses
}

// ── Raw route (before context applied) ───────────────────────────────────────

type rawRoute struct {
	method     string
	uri        string
	handler    string
	isResource bool
}

// ── Block entry (either a single route or a nested group) ────────────────────

type blockEntry struct {
	route *rawRoute
	group *groupEntry
}

type groupEntry struct {
	ctx     groupContext
	entries []blockEntry
}

// ── File parser ───────────────────────────────────────────────────────────────

func (p *LaravelParser) parseRouteFile(content string) []models.Endpoint {
	content = stripComments(content)
	entries := parseBlock(content)
	return p.flatten(entries, groupContext{})
}

// ── Block parser ──────────────────────────────────────────────────────────────

func parseBlock(content string) []blockEntry {
	var entries []blockEntry
	i := 0
	n := len(content)

	for i < n {
		// skip whitespace
		for i < n && isSpace(content[i]) {
			i++
		}
		if i >= n {
			break
		}

		if ahead(content, i, "Route::") {
			newI, entry := parseStatement(content, i)
			if entry != nil {
				entries = append(entries, *entry)
			}
			i = newI
		} else {
			i++
		}
	}

	return entries
}

// ── Statement parser ──────────────────────────────────────────────────────────

func parseStatement(content string, start int) (end int, entry *blockEntry) {
	i := start + len("Route::")

	// Read first method name (get, post, prefix, group, middleware, controller, apiResource…)
	firstName := readIdent(content, &i)

	// Build the full chain of segments
	var chain []segment
	chain = append(chain, segment{name: firstName})

	for i < len(content) {
		skipWS(content, &i)
		if i >= len(content) {
			break
		}

		ch := content[i]

		if ch == '(' {
			args, after := readBalanced(content, i, '(', ')')
			chain[len(chain)-1].args = args
			i = after

		} else if ahead(content, i, "->") {
			i += 2
			name := readIdent(content, &i)
			if name != "" {
				chain = append(chain, segment{name: name})
			}

		} else if ch == ';' {
			i++
			break
		} else {
			// unexpected char — stop
			break
		}
	}

	end = i

	if len(chain) == 0 {
		return end, nil
	}

	first := chain[0]

	switch {
	case isHTTPVerb(first.name):
		r := parseSimpleRoute(first.name, first.args)
		if r != nil {
			// Absorb any ->name(...) chained after (e.g. ->name('route.name')) — ignore for now
			return end, &blockEntry{route: r}
		}

	case first.name == "apiResource" || first.name == "resource":
		r := parseResourceRoute(first.args)
		if r != nil {
			return end, &blockEntry{route: r}
		}

	default:
		// Group-style chain: prefix/middleware/controller/group in any order
		ctx, body, isGroup := extractGroupChain(chain)
		if isGroup {
			children := parseBlock(body)
			return end, &blockEntry{group: &groupEntry{ctx: ctx, entries: children}}
		}
	}

	return end, nil
}

// ── Group chain extraction ────────────────────────────────────────────────────

func extractGroupChain(chain []segment) (ctx groupContext, body string, ok bool) {
	for _, seg := range chain {
		switch seg.name {
		case "prefix":
			ctx.prefix = extractFirstString(seg.args)
		case "middleware":
			ctx.middleware = append(ctx.middleware, extractStrings(seg.args)...)
		case "controller":
			ctx.controller = extractController(seg.args)
		case "group":
			body = extractFunctionBody(seg.args)
			ok = true
		}
	}
	return
}

// ── Route parsing ─────────────────────────────────────────────────────────────

func parseSimpleRoute(method, args string) *rawRoute {
	// args: '/uri', [Controller::class, 'action']
	//   or: '/uri', 'Controller@action'
	//   or: '/uri', 'action'   (when controller is set at group level)
	args = strings.TrimSpace(args)

	// Find the first quoted string (the URI)
	uri, rest := splitFirstString(args)
	if uri == "" {
		return nil
	}

	// rest should start with a comma then the handler
	rest = strings.TrimLeft(rest, " ,")

	return &rawRoute{
		method:  strings.ToUpper(method),
		uri:     uri,
		handler: rest,
	}
}

func parseResourceRoute(args string) *rawRoute {
	args = strings.TrimSpace(args)
	uri, rest := splitFirstString(args)
	if uri == "" {
		return nil
	}
	rest = strings.TrimLeft(rest, " ,")
	return &rawRoute{
		method:     "RESOURCE",
		uri:        uri,
		handler:    rest,
		isResource: true,
	}
}

// ── apiResource expansion ─────────────────────────────────────────────────────

var resourceActions = []struct{ method, suffix, action string }{
	{"GET", "", "index"},
	{"POST", "", "store"},
	{"GET", "/{id}", "show"},
	{"PUT", "/{id}", "update"},
	{"PATCH", "/{id}", "update"},
	{"DELETE", "/{id}", "destroy"},
}

func expandResource(r rawRoute, ctx groupContext) []models.Endpoint {
	controller := extractController(r.handler)
	if controller == "" {
		controller = ctx.controller
	}
	base := joinPath(ctx.prefix, r.uri)

	var eps []models.Endpoint
	for _, a := range resourceActions {
		eps = append(eps, models.Endpoint{
			Method:     a.method,
			URI:        base + a.suffix,
			Controller: controller,
			Action:     a.action,
			Middleware: append([]string{}, ctx.middleware...),
		})
	}
	return eps
}

// ── Flatten tree → endpoints ──────────────────────────────────────────────────

func (p *LaravelParser) flatten(entries []blockEntry, parentCtx groupContext) []models.Endpoint {
	var result []models.Endpoint

	for _, e := range entries {
		switch {
		case e.route != nil:
			r := *e.route
			if r.isResource {
				eps := expandResource(r, parentCtx)
				for i := range eps {
					eps[i].RawSource = p.readControllerSource(eps[i].Controller, eps[i].Action)
					eps[i].StaticMeta = p.extractStaticMeta(eps[i].RawSource)
				}
				result = append(result, eps...)
			} else {
				ep := p.resolveEndpoint(r, parentCtx)
				result = append(result, ep)
			}

		case e.group != nil:
			childCtx := parentCtx.merge(e.group.ctx)
			result = append(result, p.flatten(e.group.entries, childCtx)...)
		}
	}

	return result
}

func (p *LaravelParser) resolveEndpoint(r rawRoute, ctx groupContext) models.Endpoint {
	uri := joinPath(ctx.prefix, r.uri)
	controller, action := parseHandler(r.handler)

	if controller == "" {
		controller = ctx.controller
	}

	ep := models.Endpoint{
		Method:     r.method,
		URI:        uri,
		Controller: controller,
		Action:     action,
		Middleware: append([]string{}, ctx.middleware...),
	}
	ep.RawSource = p.readControllerSource(controller, action)
	ep.StaticMeta = p.extractStaticMeta(ep.RawSource)
	return ep
}

// ── Handler parsing ───────────────────────────────────────────────────────────

// parseHandler handles:
//
//	[UserController::class, 'store']
//	'UserController@store'
//	'store'   (action only, controller from group)
func parseHandler(h string) (controller, action string) {
	h = strings.TrimSpace(h)

	if strings.HasPrefix(h, "[") {
		inner := strings.Trim(h, "[] ")
		parts := strings.SplitN(inner, ",", 2)
		if len(parts) == 2 {
			controller = cleanClass(parts[0])
			action = strings.Trim(strings.TrimSpace(parts[1]), `'"`)
		}
		return
	}

	unquoted := strings.Trim(h, `'"`)
	if strings.Contains(unquoted, "@") {
		parts := strings.SplitN(unquoted, "@", 2)
		controller = parts[0]
		if len(parts) == 2 {
			action = parts[1]
		}
		return
	}

	// Just an action string
	action = unquoted
	return
}

func extractController(s string) string {
	s = strings.TrimSpace(s)
	// Could be: SomeController::class  or  'SomeController'  or  [SomeController::class, 'action']
	if strings.HasPrefix(s, "[") {
		c, _ := parseHandler(s)
		return c
	}
	return cleanClass(s)
}

func cleanClass(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "::class", "")
	s = strings.Trim(s, `'"`)
	return strings.TrimSpace(s)
}

// ── Controller source reading ─────────────────────────────────────────────────

func (p *LaravelParser) readControllerSource(controller, action string) string {
	if controller == "" || action == "" || p.projectRoot == "" {
		return ""
	}
	rel := strings.ReplaceAll(controller, `\`, string(filepath.Separator))
	rel = strings.Replace(rel, "App"+string(filepath.Separator), "app"+string(filepath.Separator), 1)
	path := filepath.Join(p.projectRoot, rel+".php")

	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return extractMethodSource(string(content), action)
}

func extractMethodSource(src, method string) string {
	needle := "public function " + method + "("
	start := strings.Index(src, needle)
	if start == -1 {
		return src
	}
	depth, inMethod := 0, false
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
			inMethod = true
		case '}':
			depth--
			if inMethod && depth == 0 {
				return src[start : i+1]
			}
		}
	}
	return src[start:]
}

// ── Static meta ───────────────────────────────────────────────────────────────

var validatePattern = regexp.MustCompile(`\$request->validate\(\s*\[([\s\S]*?)\]\s*\)`)
var ruleLinePattern = regexp.MustCompile(`['"](\w+)['"]\s*=>\s*['"]([^'"]+)['"]`)

func (p *LaravelParser) extractStaticMeta(source string) models.StaticMeta {
	if source == "" {
		return models.StaticMeta{}
	}
	meta := models.StaticMeta{}
	if vm := validatePattern.FindStringSubmatch(source); len(vm) > 1 {
		for _, rl := range ruleLinePattern.FindAllStringSubmatch(vm[1], -1) {
			meta.RequestParams = append(meta.RequestParams, models.Param{
				Name:     rl[1],
				Type:     inferType(rl[2]),
				Required: strings.Contains(rl[2], "required"),
				Rules:    rl[2],
			})
		}
	}
	return meta
}

func inferType(rules string) string {
	switch {
	case strings.Contains(rules, "integer") || strings.Contains(rules, "numeric"):
		return "integer"
	case strings.Contains(rules, "boolean"):
		return "boolean"
	case strings.Contains(rules, "array"):
		return "array"
	case strings.Contains(rules, "file") || strings.Contains(rules, "image"):
		return "file"
	case strings.Contains(rules, "email"):
		return "string (email)"
	default:
		return "string"
	}
}

// ── String argument helpers ───────────────────────────────────────────────────

// extractFirstString returns the first quoted string value in s
func extractFirstString(s string) string {
	re := regexp.MustCompile(`['"]([^'"]+)['"]`)
	m := re.FindStringSubmatch(s)
	if m != nil {
		return m[1]
	}
	return ""
}

// extractStrings returns all quoted string values in s
func extractStrings(s string) []string {
	re := regexp.MustCompile(`['"]([^'"]+)['"]`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		if v := strings.TrimSpace(m[1]); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// splitFirstString finds the first quoted string and returns (value, rest-of-s-after-closing-quote)
func splitFirstString(s string) (value, rest string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' || s[i] == '"' {
			q := s[i]
			j := i + 1
			for j < len(s) && s[j] != q {
				if s[j] == '\\' {
					j++
				}
				j++
			}
			if j < len(s) {
				return s[i+1 : j], s[j+1:]
			}
		}
	}
	return "", s
}

// extractFunctionBody extracts the content inside { } from a function() { ... } expression
func extractFunctionBody(args string) string {
	start := strings.Index(args, "{")
	if start == -1 {
		return ""
	}
	depth := 0
	for i := start; i < len(args); i++ {
		switch args[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return args[start+1 : i]
			}
		}
	}
	return args[start+1:]
}

// ── Low-level text helpers ────────────────────────────────────────────────────

func isHTTPVerb(s string) bool {
	switch strings.ToLower(s) {
	case "get", "post", "put", "patch", "delete", "options", "any":
		return true
	}
	return false
}

func readIdent(s string, i *int) string {
	start := *i
	for *i < len(s) && isIdentChar(s[*i]) {
		(*i)++
	}
	return s[start:*i]
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

func skipWS(s string, i *int) {
	for *i < len(s) && isSpace(s[*i]) {
		(*i)++
	}
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func ahead(s string, i int, needle string) bool {
	return i+len(needle) <= len(s) && s[i:i+len(needle)] == needle
}

// readBalanced reads from the opening delimiter to the matching close.
// Returns inner content (without delimiters) and position after close.
func readBalanced(s string, start int, open, close byte) (inner string, after int) {
	depth := 0
	i := start
	for i < len(s) {
		switch {
		case s[i] == open:
			depth++
			if depth == 1 {
				// mark content start
			}
		case s[i] == close:
			depth--
			if depth == 0 {
				return s[start+1 : i], i + 1
			}
		case s[i] == '\'' || s[i] == '"':
			// skip string literal
			q := s[i]
			i++
			for i < len(s) && s[i] != q {
				if s[i] == '\\' {
					i++
				}
				i++
			}
		}
		i++
	}
	return s[start+1:], len(s)
}

// stripComments removes PHP // # and /* */ comments
func stripComments(s string) string {
	s = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`//[^\n]*`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`#[^\n]*`).ReplaceAllString(s, "")
	return s
}

// joinPath joins two URI segments
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
