package node

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
	parser.Register("node", func(root string) parser.Parser {
		return New(root)
	})
}

type NodeParser struct {
	projectRoot string
}

func New(projectRoot string) *NodeParser {
	return &NodeParser{projectRoot: projectRoot}
}

func (p *NodeParser) Language() string { return "node" }

// ── ResolveIncludes ──────────────────────────────────────────────────────────

func (p *NodeParser) ResolveIncludes(files []string) ([]string, error) {
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

func (p *NodeParser) resolveEntry(path string) ([]string, error) {
	path = p.resolveUnderRoot(path)
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}

	var files []string
	err = filepath.Walk(path, func(walkPath string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			name := fi.Name()
			if name == "node_modules" || name == "dist" || name == "build" ||
				name == ".git" || name == "coverage" || name == "__tests__" {
				return filepath.SkipDir
			}
			return nil
		}
		if isJSFile(fi.Name()) {
			files = append(files, walkPath)
		}
		return nil
	})
	return files, err
}

var jsExtensions = map[string]bool{
	".js": true, ".ts": true, ".mjs": true, ".mts": true, ".cjs": true, ".cts": true,
}

func isJSFile(name string) bool {
	return jsExtensions[filepath.Ext(name)]
}

func (p *NodeParser) resolveUnderRoot(path string) string {
	if filepath.IsAbs(path) || p.projectRoot == "" {
		return path
	}
	return filepath.Join(p.projectRoot, path)
}

// ── ParseSections ────────────────────────────────────────────────────────────

func (p *NodeParser) ParseSections(files []string) ([]models.RouteSection, error) {
	var sections []models.RouteSection

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f, err)
		}

		endpoints := p.parseFile(string(content))
		if len(endpoints) == 0 {
			continue
		}

		lang := fileLanguage(f)
		for i := range endpoints {
			endpoints[i].Language = lang
		}

		sections = append(sections, models.RouteSection{
			FilePath:  f,
			Version:   inferVersion(f),
			Endpoints: endpoints,
		})
	}

	return sections, nil
}

func fileLanguage(path string) string {
	ext := filepath.Ext(path)
	if ext == ".ts" || ext == ".mts" || ext == ".cts" {
		return "typescript"
	}
	return "javascript"
}

func (p *NodeParser) parseFile(src string) []models.Endpoint {
	src = stripJSComments(src)

	routerPrefixes := buildRouterPrefixes(src)

	var endpoints []models.Endpoint
	endpoints = append(endpoints, parseMethodRoutes(src, routerPrefixes)...)
	endpoints = append(endpoints, parseRouteChains(src, routerPrefixes)...)
	endpoints = append(endpoints, parseFastifyRouteObjects(src, routerPrefixes)...)
	endpoints = append(endpoints, parseNativeHTTPRoutes(src)...)

	return deduplicateEndpoints(endpoints)
}

// ── Router prefix resolution ─────────────────────────────────────────────────

var (
	routerMountRe = regexp.MustCompile(
		`(\w+)\.use\s*\(\s*['"]([^'"]+)['"]\s*,\s*(\w+)`)

	fastifyRegisterRe = regexp.MustCompile(
		`(\w+)\.register\s*\(\s*(\w+)\s*,\s*\{[^}]*prefix\s*:\s*['"]([^'"]+)['"]`)
)

func buildRouterPrefixes(src string) map[string]string {
	prefixes := make(map[string]string)

	for _, m := range routerMountRe.FindAllStringSubmatch(src, -1) {
		routerVar := m[3]
		prefix := m[2]
		prefixes[routerVar] = prefix
	}

	for _, m := range fastifyRegisterRe.FindAllStringSubmatch(src, -1) {
		handlerVar := m[2]
		prefix := m[3]
		prefixes[handlerVar] = prefix
	}

	return prefixes
}

// ── Express/Fastify method routes: var.method('/path', ...) ──────────────────

var methodRouteRe = regexp.MustCompile(
	`(\w+)\.(get|post|put|patch|delete|options|head|all)\s*\(\s*['"]([^'"]+)['"]`)

func parseMethodRoutes(src string, prefixes map[string]string) []models.Endpoint {
	var endpoints []models.Endpoint

	lines := strings.Split(src, "\n")
	for lineIdx, line := range lines {
		for _, m := range methodRouteRe.FindAllStringSubmatch(line, -1) {
			callerVar := m[1]
			method := strings.ToUpper(m[2])
			routePath := m[3]

			if isTestVariable(callerVar) {
				continue
			}

			prefix := prefixes[callerVar]
			uri := joinPath(prefix, routePath)

			fullStmt := collectStatement(lines, lineIdx)
			middleware := extractMiddlewareFromArgs(fullStmt, routePath)
			handler := extractHandlerName(fullStmt)

			ep := models.Endpoint{
				Method:     method,
				URI:        uri,
				Controller: callerVar,
				Action:     handler,
				Middleware: middleware,
				RawSource:  strings.TrimSpace(fullStmt),
			}
			ep.StaticMeta.RequestParams = extractRouteParams(uri)

			endpoints = append(endpoints, ep)
		}
	}

	return endpoints
}

// ── Express route chains: var.route('/path').get().post() ────────────────────

var (
	routeChainStartRe = regexp.MustCompile(
		`(\w+)\.route\s*\(\s*['"]([^'"]+)['"]\s*\)`)

	chainedVerbRe = regexp.MustCompile(
		`\.(get|post|put|patch|delete|options|head|all)\s*\(`)
)

func parseRouteChains(src string, prefixes map[string]string) []models.Endpoint {
	var endpoints []models.Endpoint

	lines := strings.Split(src, "\n")
	for lineIdx, line := range lines {
		m := routeChainStartRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		callerVar := m[1]
		routePath := m[2]

		if isTestVariable(callerVar) {
			continue
		}

		prefix := prefixes[callerVar]
		uri := joinPath(prefix, routePath)

		fullStmt := collectStatement(lines, lineIdx)

		for _, vm := range chainedVerbRe.FindAllStringSubmatch(fullStmt, -1) {
			method := strings.ToUpper(vm[1])
			ep := models.Endpoint{
				Method:     method,
				URI:        uri,
				Controller: callerVar,
				Action:     deriveActionName(method, uri),
				RawSource:  strings.TrimSpace(fullStmt),
			}
			ep.StaticMeta.RequestParams = extractRouteParams(uri)

			endpoints = append(endpoints, ep)
		}
	}

	return endpoints
}

// ── Fastify route objects: var.route({ method, url, handler }) ───────────────

var (
	fastifyRouteCallRe = regexp.MustCompile(
		`(\w+)\.route\s*\(\s*\{`)

	routeObjMethodRe      = regexp.MustCompile(`method\s*:\s*['"](\w+)['"]`)
	routeObjMethodArrayRe = regexp.MustCompile(`method\s*:\s*\[([^\]]+)\]`)
	routeObjUrlRe         = regexp.MustCompile(`(?:url|path)\s*:\s*['"]([^'"]+)['"]`)
	routeObjHandlerNameRe = regexp.MustCompile(`handler\s*:\s*(\w+)`)
	routeObjPreHandlerRe  = regexp.MustCompile(`(?:preHandler|onRequest)\s*:\s*\[([^\]]+)\]`)
)

func parseFastifyRouteObjects(src string, prefixes map[string]string) []models.Endpoint {
	var endpoints []models.Endpoint

	matches := fastifyRouteCallRe.FindAllStringSubmatchIndex(src, -1)
	for _, loc := range matches {
		callerVar := src[loc[2]:loc[3]]
		if isTestVariable(callerVar) {
			continue
		}

		braceStart := strings.Index(src[loc[0]:], "{") + loc[0]
		body := extractBraceBlock(src, braceStart)
		if body == "" {
			continue
		}

		prefix := prefixes[callerVar]

		endPos := braceStart + 1 + len(body) + 1
		if endPos > len(src) {
			endPos = len(src)
		}
		fullSource := strings.TrimSpace(src[loc[0]:endPos])

		urlMatch := routeObjUrlRe.FindStringSubmatch(body)
		if urlMatch == nil {
			continue
		}
		routePath := urlMatch[1]
		uri := joinPath(prefix, routePath)

		handler := ""
		if hm := routeObjHandlerNameRe.FindStringSubmatch(body); hm != nil {
			handler = hm[1]
		}

		var middleware []string
		if pm := routeObjPreHandlerRe.FindStringSubmatch(body); pm != nil {
			for _, part := range strings.Split(pm[1], ",") {
				part = strings.TrimSpace(part)
				if identRe.MatchString(part) {
					middleware = append(middleware, part)
				}
			}
		}

		var methods []string
		if am := routeObjMethodArrayRe.FindStringSubmatch(body); am != nil {
			strRe := regexp.MustCompile(`['"](\w+)['"]`)
			for _, ms := range strRe.FindAllStringSubmatch(am[1], -1) {
				methods = append(methods, strings.ToUpper(ms[1]))
			}
		} else if mm := routeObjMethodRe.FindStringSubmatch(body); mm != nil {
			methods = append(methods, strings.ToUpper(mm[1]))
		}

		for _, method := range methods {
			ep := models.Endpoint{
				Method:     method,
				URI:        uri,
				Controller: callerVar,
				Action:     handler,
				Middleware: middleware,
				RawSource:  fullSource,
			}
			ep.StaticMeta.RequestParams = extractRouteParams(uri)
			endpoints = append(endpoints, ep)
		}
	}

	return endpoints
}

// ── Native HTTP routes ───────────────────────────────────────────────────────

var (
	// req.method === 'GET' && req.url === '/path'
	nativeMethodUrlRe = regexp.MustCompile(
		`req(?:uest)?\.method\s*===?\s*['"](\w+)['"]\s*&&\s*(?:req(?:uest)?\.url|url)\s*===?\s*['"]([^'"]+)['"]`)

	// req.url === '/path' && req.method === 'GET'
	nativeUrlMethodRe = regexp.MustCompile(
		`(?:req(?:uest)?\.url|url)\s*===?\s*['"]([^'"]+)['"]\s*&&\s*req(?:uest)?\.method\s*===?\s*['"](\w+)['"]`)

	// switch(req.url) { case '/path': ... } with enclosing method check
	nativeSwitchUrlRe = regexp.MustCompile(
		`switch\s*\(\s*req(?:uest)?\.url\s*\)\s*\{`)
	nativeSwitchMethodRe = regexp.MustCompile(
		`switch\s*\(\s*req(?:uest)?\.method\s*\)\s*\{`)

	caseStringRe = regexp.MustCompile(`case\s+['"]([^'"]+)['"]\s*:`)
)

func parseNativeHTTPRoutes(src string) []models.Endpoint {
	var endpoints []models.Endpoint
	seen := map[string]bool{}

	addEndpoint := func(method, uri, rawSource string) {
		key := method + " " + uri
		if seen[key] {
			return
		}
		seen[key] = true
		ep := models.Endpoint{
			Method:     strings.ToUpper(method),
			URI:        uri,
			Controller: "http",
			Action:     deriveActionName(method, uri),
			RawSource:  rawSource,
		}
		ep.StaticMeta.RequestParams = extractRouteParams(uri)
		endpoints = append(endpoints, ep)
	}

	// Pattern 1: req.method === 'GET' && req.url === '/path'
	for _, m := range nativeMethodUrlRe.FindAllStringSubmatch(src, -1) {
		addEndpoint(m[1], m[2], m[0])
	}

	// Pattern 2: req.url === '/path' && req.method === 'GET'
	for _, m := range nativeUrlMethodRe.FindAllStringSubmatch(src, -1) {
		addEndpoint(m[2], m[1], m[0])
	}

	// Pattern 3: switch(req.method) with case 'GET': inside, combined with URL check
	endpoints = append(endpoints, parseSwitchBlocks(src)...)

	return endpoints
}

func parseSwitchBlocks(src string) []models.Endpoint {
	var endpoints []models.Endpoint
	seen := map[string]bool{}

	// Look for switch(req.url) blocks
	urlSwitchLocs := nativeSwitchUrlRe.FindAllStringIndex(src, -1)
	for _, loc := range urlSwitchLocs {
		bracePos := loc[1] - 1
		body := extractBraceBlock(src, bracePos)

		// Each case is a URL
		for _, cm := range caseStringRe.FindAllStringSubmatch(body, -1) {
			url := cm[1]

			// Look for method checks inside this case block
			caseStart := strings.Index(body, cm[0])
			caseBody := extractCaseBody(body, caseStart+len(cm[0]))

			methodMatches := caseStringRe.FindAllStringSubmatch(caseBody, -1)
			if len(methodMatches) > 0 && nativeSwitchMethodRe.MatchString(caseBody) {
				for _, mm := range methodMatches {
					method := mm[1]
					key := method + " " + url
					if !seen[key] {
						seen[key] = true
						ep := models.Endpoint{
							Method:     strings.ToUpper(method),
							URI:        url,
							Controller: "http",
							Action:     deriveActionName(method, url),
							RawSource:  cm[0],
						}
						ep.StaticMeta.RequestParams = extractRouteParams(url)
						endpoints = append(endpoints, ep)
					}
				}
			} else if !strings.HasPrefix(url, "/") {
				continue
			} else {
				// No nested method switch — assume ALL methods
				key := "ALL " + url
				if !seen[key] {
					seen[key] = true
					ep := models.Endpoint{
						Method:     "ALL",
						URI:        url,
						Controller: "http",
						Action:     deriveActionName("ALL", url),
						RawSource:  cm[0],
					}
					ep.StaticMeta.RequestParams = extractRouteParams(url)
					endpoints = append(endpoints, ep)
				}
			}
		}
	}

	// Look for switch(req.method) blocks with URL context
	methodSwitchLocs := nativeSwitchMethodRe.FindAllStringIndex(src, -1)
	for _, loc := range methodSwitchLocs {
		// Check for a nearby URL condition before this switch
		before := src[:loc[0]]
		urlRe := regexp.MustCompile(`req(?:uest)?\.url\s*===?\s*['"]([^'"]+)['"]\s*`)
		urlMatches := urlRe.FindAllStringSubmatch(before, -1)

		var contextUrl string
		if len(urlMatches) > 0 {
			contextUrl = urlMatches[len(urlMatches)-1][1]
		}
		if contextUrl == "" {
			continue
		}

		bracePos := loc[1] - 1
		body := extractBraceBlock(src, bracePos)
		for _, cm := range caseStringRe.FindAllStringSubmatch(body, -1) {
			method := cm[1]
			key := method + " " + contextUrl
			if !seen[key] {
				seen[key] = true
				ep := models.Endpoint{
					Method:     strings.ToUpper(method),
					URI:        contextUrl,
					Controller: "http",
					Action:     deriveActionName(method, contextUrl),
					RawSource:  cm[0],
				}
				ep.StaticMeta.RequestParams = extractRouteParams(contextUrl)
				endpoints = append(endpoints, ep)
			}
		}
	}

	return endpoints
}

func extractCaseBody(body string, start int) string {
	depth := 0
	for i := start; i < len(body); i++ {
		switch body[i] {
		case '{':
			depth++
		case '}':
			depth--
		}
		if depth < 0 {
			return body[start:i]
		}
		// Next case or default
		if depth == 0 && i+5 < len(body) {
			rest := body[i:]
			if strings.HasPrefix(rest, "case ") || strings.HasPrefix(rest, "default:") {
				return body[start:i]
			}
		}
	}
	return body[start:]
}

// ── Parameter extraction ─────────────────────────────────────────────────────

var routeParamRe = regexp.MustCompile(`:(\w+)`)

func extractRouteParams(path string) []models.Param {
	matches := routeParamRe.FindAllStringSubmatch(path, -1)
	if len(matches) == 0 {
		return nil
	}
	var params []models.Param
	for _, m := range matches {
		params = append(params, models.Param{
			Name:     m[1],
			Type:     "string",
			Required: true,
			Rules:    "route",
		})
	}
	return params
}

// ── Middleware extraction ─────────────────────────────────────────────────────

func extractMiddlewareFromArgs(stmt, routePath string) []string {
	idx := strings.Index(stmt, routePath)
	if idx < 0 {
		return nil
	}

	afterPath := stmt[idx+len(routePath):]
	quoteIdx := strings.IndexAny(afterPath, `'"`)
	if quoteIdx < 0 {
		return nil
	}
	remaining := afterPath[quoteIdx+1:]

	remaining = strings.TrimSpace(remaining)
	if len(remaining) == 0 || remaining[0] != ',' {
		return nil
	}
	remaining = remaining[1:]

	parts := splitTopLevel(remaining, ',')
	if len(parts) <= 1 {
		return nil
	}

	var middleware []string
	for _, part := range parts[:len(parts)-1] {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.HasPrefix(part, "[") {
			inner := strings.Trim(part, "[] ")
			for _, mw := range strings.Split(inner, ",") {
				mw = strings.TrimSpace(mw)
				name := extractIdentifier(mw)
				if name != "" && !isHandlerKeyword(name) {
					middleware = append(middleware, name)
				}
			}
		} else {
			name := extractIdentifier(part)
			if name != "" && !isHandlerKeyword(name) {
				middleware = append(middleware, name)
			}
		}
	}

	return middleware
}

func splitTopLevel(s string, sep byte) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			depth--
			if depth < 0 {
				parts = append(parts, s[start:i])
				return parts
			}
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

var identRe = regexp.MustCompile(`^\w+$`)

func extractIdentifier(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") {
		return ""
	}
	re := regexp.MustCompile(`^(\w+)`)
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

var handlerKeywords = map[string]bool{
	"async": true, "function": true, "req": true, "res": true,
	"reply": true, "request": true, "response": true, "next": true,
	"return": true, "await": true, "const": true, "let": true, "var": true,
	"true": true, "false": true, "null": true, "undefined": true,
	"new": true, "this": true, "err": true, "error": true,
}

func isHandlerKeyword(s string) bool {
	return handlerKeywords[s]
}

// ── Handler name extraction ──────────────────────────────────────────────────

var namedHandlerRe = regexp.MustCompile(`,\s*([\w.]+)\s*\)\s*;?\s*$`)

func extractHandlerName(stmt string) string {
	stmt = strings.TrimSpace(stmt)

	if m := namedHandlerRe.FindStringSubmatch(stmt); m != nil {
		name := m[1]
		if !isHandlerKeyword(name) {
			return name
		}
	}

	if strings.Contains(stmt, "=>") ||
		strings.Contains(stmt, "function(") ||
		strings.Contains(stmt, "function (") {
		return "anonymous"
	}

	return ""
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func stripJSComments(s string) string {
	s = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`//[^\n]*`).ReplaceAllString(s, "")
	return s
}

func inferVersion(path string) string {
	re := regexp.MustCompile(`(?i)[/\\](v\d+)[/\\]`)
	m := re.FindStringSubmatch(path)
	if m != nil {
		return strings.ToLower(m[1])
	}
	return ""
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

func collectStatement(lines []string, startIdx int) string {
	var sb strings.Builder
	depth := 0

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		sb.WriteString(line)
		sb.WriteString("\n")

		for _, ch := range line {
			switch ch {
			case '(', '{', '[':
				depth++
			case ')', '}', ']':
				depth--
			}
		}

		trimmed := strings.TrimSpace(line)
		if depth == 0 && strings.HasSuffix(trimmed, ";") {
			break
		}
		if depth <= 0 && i > startIdx &&
			(strings.HasSuffix(trimmed, ");") ||
				strings.HasSuffix(trimmed, "});") ||
				strings.HasSuffix(trimmed, ");")) {
			break
		}
	}

	return sb.String()
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

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func deriveActionName(method, path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var name string
	for _, part := range parts {
		if part == "" || strings.HasPrefix(part, ":") || strings.HasPrefix(part, "{") {
			continue
		}
		name += capitalize(part)
	}
	if name == "" {
		name = "Root"
	}
	return strings.ToUpper(method) + "_" + name
}

func deduplicateEndpoints(endpoints []models.Endpoint) []models.Endpoint {
	seen := map[string]bool{}
	var result []models.Endpoint
	for _, ep := range endpoints {
		key := ep.Method + " " + ep.URI
		if !seen[key] {
			seen[key] = true
			result = append(result, ep)
		}
	}
	return result
}

var testVariables = map[string]bool{
	"test": true, "expect": true, "describe": true, "it": true,
	"jest": true, "chai": true, "assert": true, "mocha": true,
	"supertest": true,
}

func isTestVariable(name string) bool {
	return testVariables[name]
}
