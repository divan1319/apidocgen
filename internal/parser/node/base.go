package node

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/pkg/models"
)

// ── Base parser (shared by Express, Fastify, NativeHTTP) ─────────────────────

type baseParser struct {
	projectRoot string
}

func (b *baseParser) ResolveIncludes(files []string) ([]string, error) {
	var all []string
	seen := map[string]bool{}

	for _, f := range files {
		resolved, err := b.resolveEntry(f)
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

func (b *baseParser) resolveEntry(path string) ([]string, error) {
	path = b.resolveUnderRoot(path)
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

func (b *baseParser) resolveUnderRoot(path string) string {
	if filepath.IsAbs(path) || b.projectRoot == "" {
		return path
	}
	return filepath.Join(b.projectRoot, path)
}

func (b *baseParser) parseSections(files []string, parse func(string) []models.Endpoint) ([]models.RouteSection, error) {
	var sections []models.RouteSection

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f, err)
		}

		endpoints := parse(string(content))
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

// ── Shared file helpers ──────────────────────────────────────────────────────

var jsExtensions = map[string]bool{
	".js": true, ".ts": true, ".mjs": true, ".mts": true, ".cjs": true, ".cts": true,
}

func isJSFile(name string) bool {
	return jsExtensions[filepath.Ext(name)]
}

func fileLanguage(path string) string {
	ext := filepath.Ext(path)
	if ext == ".ts" || ext == ".mts" || ext == ".cts" {
		return "typescript"
	}
	return "javascript"
}

func inferVersion(path string) string {
	re := regexp.MustCompile(`(?i)[/\\](v\d+)[/\\]`)
	m := re.FindStringSubmatch(path)
	if m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// ── Shared route parsing helpers ─────────────────────────────────────────────

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

// ── Text helpers ─────────────────────────────────────────────────────────────

func stripJSComments(s string) string {
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
