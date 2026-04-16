package golang

import (
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/pkg/models"
)

func init() {
	parser.Register("fiber", func(root string) parser.Parser {
		return NewFiber(root)
	})
}

type FiberParser struct {
	baseParser
}

func NewFiber(projectRoot string) *FiberParser {
	return &FiberParser{baseParser{projectRoot: projectRoot}}
}

func (p *FiberParser) Language() string { return "fiber" }

func (p *FiberParser) ParseSections(files []string) ([]models.RouteSection, error) {
	return p.parseSections(files, p.parseFile)
}

func (p *FiberParser) parseFile(src string) []models.Endpoint {
	src = stripGoComments(src)

	prefixes := buildFiberGroupPrefixes(src)
	var endpoints []models.Endpoint
	endpoints = append(endpoints, parseFiberRoutes(src, prefixes)...)
	endpoints = append(endpoints, parseFiberUse(src, prefixes)...)

	return deduplicateEndpoints(endpoints)
}

// ── Group prefix resolution ──────────────────────────────────────────────────

// Matches: api := app.Group("/api")
//          v1 := api.Group("/v1")
//          v1 := app.Group("/v1", middleware)
var fiberGroupRe = regexp.MustCompile(
	`(\w+)\s*(?::=|=)\s*(\w+)\.Group\s*\(\s*"([^"]*)"`)

func buildFiberGroupPrefixes(src string) map[string]string {
	prefixes := make(map[string]string)

	for _, m := range fiberGroupRe.FindAllStringSubmatch(src, -1) {
		groupVar := m[1]
		parentVar := m[2]
		prefix := m[3]

		parentPrefix := prefixes[parentVar]
		prefixes[groupVar] = joinPath(parentPrefix, prefix)
	}

	return prefixes
}

// ── Fiber method routes ──────────────────────────────────────────────────────

// Matches: app.Get("/path", handler)
//          api.Post("/path", middleware, handler)
//          v1.Put("/path/:id", handler)
var fiberRouteRe = regexp.MustCompile(
	`(\w+)\.(Get|Post|Put|Patch|Delete|Options|Head|All)\s*\(\s*"([^"]*)"`)

func parseFiberRoutes(src string, prefixes map[string]string) []models.Endpoint {
	var endpoints []models.Endpoint

	lines := strings.Split(src, "\n")
	for lineIdx, line := range lines {
		for _, m := range fiberRouteRe.FindAllStringSubmatch(line, -1) {
			callerVar := m[1]
			method := strings.ToUpper(m[2])
			routePath := m[3]

			prefix := prefixes[callerVar]
			uri := joinPath(prefix, routePath)

			fullStmt := collectGoStatement(lines, lineIdx)
			handler := extractFiberHandler(fullStmt, routePath)
			middleware := extractFiberMiddleware(fullStmt, routePath)

			ep := models.Endpoint{
				Method:     method,
				URI:        uri,
				Controller: callerVar,
				Action:     handler,
				Middleware: middleware,
				RawSource:  strings.TrimSpace(fullStmt),
			}
			ep.StaticMeta.RequestParams = extractGoPathParams(uri)
			endpoints = append(endpoints, ep)
		}
	}

	return endpoints
}

// ── Fiber app.Use for route-level middleware ──────────────────────────────────

// Matches: app.Use("/api", middleware)
var fiberUseRouteRe = regexp.MustCompile(
	`(\w+)\.Use\s*\(\s*"([^"]+)"\s*,`)

func parseFiberUse(src string, prefixes map[string]string) []models.Endpoint {
	// We don't create endpoints for Use() - these are middleware mounts, not routes.
	// But we track them for prefix/middleware resolution purposes.
	_ = fiberUseRouteRe
	return nil
}

// ── Handler/middleware extraction ─────────────────────────────────────────────

func extractFiberHandler(stmt, routePath string) string {
	idx := strings.Index(stmt, routePath)
	if idx < 0 {
		return ""
	}

	after := stmt[idx+len(routePath):]
	quoteIdx := strings.IndexByte(after, '"')
	if quoteIdx < 0 {
		return ""
	}
	remaining := after[quoteIdx+1:]

	remaining = strings.TrimLeft(remaining, " \t")
	if len(remaining) == 0 || remaining[0] != ',' {
		return ""
	}
	remaining = remaining[1:]

	args := splitFiberArgs(remaining)
	if len(args) == 0 {
		return ""
	}

	last := strings.TrimSpace(args[len(args)-1])
	last = strings.TrimRight(last, ")")
	last = strings.TrimSpace(last)

	if isGoKeyword(last) || last == "" {
		return ""
	}

	if strings.HasPrefix(last, "func(") || strings.HasPrefix(last, "func (") {
		return "anonymous"
	}

	re := regexp.MustCompile(`^[\w.]+$`)
	if re.MatchString(last) {
		return last
	}

	return "anonymous"
}

func extractFiberMiddleware(stmt, routePath string) []string {
	idx := strings.Index(stmt, routePath)
	if idx < 0 {
		return nil
	}

	after := stmt[idx+len(routePath):]
	quoteIdx := strings.IndexByte(after, '"')
	if quoteIdx < 0 {
		return nil
	}
	remaining := after[quoteIdx+1:]

	remaining = strings.TrimLeft(remaining, " \t")
	if len(remaining) == 0 || remaining[0] != ',' {
		return nil
	}
	remaining = remaining[1:]

	args := splitFiberArgs(remaining)
	if len(args) <= 1 {
		return nil
	}

	var middleware []string
	for _, arg := range args[:len(args)-1] {
		arg = strings.TrimSpace(arg)
		re := regexp.MustCompile(`^[\w.]+$`)
		if re.MatchString(arg) && !isGoKeyword(arg) {
			middleware = append(middleware, arg)
		}
	}

	return middleware
}

func splitFiberArgs(s string) []string {
	var args []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			if depth == 0 {
				arg := strings.TrimSpace(s[start:i])
				if arg != "" {
					args = append(args, arg)
				}
				return args
			}
			depth--
		case ',':
			if depth == 0 {
				args = append(args, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		arg := strings.TrimSpace(s[start:])
		if arg != "" {
			args = append(args, arg)
		}
	}
	return args
}

// ── Statement collection ─────────────────────────────────────────────────────

func collectGoStatement(lines []string, startIdx int) string {
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

		if depth <= 0 && i > startIdx {
			break
		}

		trimmed := strings.TrimSpace(line)
		if depth == 0 && (strings.HasSuffix(trimmed, ")") || strings.HasSuffix(trimmed, ");")) {
			break
		}
	}

	return sb.String()
}
