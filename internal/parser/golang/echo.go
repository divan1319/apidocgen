package golang

import (
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/pkg/models"
)

func init() {
	parser.Register("echo", func(root string) parser.Parser {
		return NewEcho(root)
	})
}

type EchoParser struct {
	baseParser
}

func NewEcho(projectRoot string) *EchoParser {
	return &EchoParser{baseParser{projectRoot: projectRoot}}
}

func (p *EchoParser) Language() string { return "echo" }

func (p *EchoParser) ParseSections(files []string) ([]models.RouteSection, error) {
	return p.parseSections(files, p.parseFile)
}

func (p *EchoParser) parseFile(src string) []models.Endpoint {
	src = stripGoComments(src)

	prefixes := buildEchoGroupPrefixes(src)
	var endpoints []models.Endpoint
	endpoints = append(endpoints, parseEchoRoutes(src, prefixes)...)

	return deduplicateEndpoints(endpoints)
}

// ── Group prefix resolution ──────────────────────────────────────────────────

// Matches: g := e.Group("/api")
//          v1 := g.Group("/v1")
//          admin := e.Group("/admin", middleware)
var echoGroupRe = regexp.MustCompile(
	`(\w+)\s*(?::=|=)\s*(\w+)\.Group\s*\(\s*"([^"]*)"`)

func buildEchoGroupPrefixes(src string) map[string]string {
	prefixes := make(map[string]string)

	for _, m := range echoGroupRe.FindAllStringSubmatch(src, -1) {
		groupVar := m[1]
		parentVar := m[2]
		prefix := m[3]

		parentPrefix := prefixes[parentVar]
		prefixes[groupVar] = joinPath(parentPrefix, prefix)
	}

	return prefixes
}

// ── Echo method routes ───────────────────────────────────────────────────────

// Matches: e.GET("/path", handler)
//          g.POST("/path", handler, middleware)
//          e.Any("/path", handler)
var echoRouteRe = regexp.MustCompile(
	`(\w+)\.(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD|Any)\s*\(\s*"([^"]*)"`)

func parseEchoRoutes(src string, prefixes map[string]string) []models.Endpoint {
	var endpoints []models.Endpoint

	lines := strings.Split(src, "\n")
	for lineIdx, line := range lines {
		for _, m := range echoRouteRe.FindAllStringSubmatch(line, -1) {
			callerVar := m[1]
			method := m[2]
			routePath := m[3]

			if method == "Any" {
				method = "ALL"
			} else {
				method = strings.ToUpper(method)
			}

			prefix := prefixes[callerVar]
			uri := joinPath(prefix, routePath)

			fullStmt := collectGoStatement(lines, lineIdx)
			handler := extractEchoHandler(fullStmt, routePath)
			middleware := extractEchoMiddleware(fullStmt, routePath)

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

// ── Handler/middleware extraction ─────────────────────────────────────────────

// In Echo, the handler is the 2nd arg and middleware follows:
// e.GET("/path", handler, mw1, mw2)
func extractEchoHandler(stmt, routePath string) string {
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

	first := strings.TrimSpace(args[0])
	if isGoKeyword(first) || first == "" {
		return ""
	}

	if strings.HasPrefix(first, "func(") || strings.HasPrefix(first, "func (") {
		return "anonymous"
	}

	re := regexp.MustCompile(`^[\w.]+$`)
	if re.MatchString(first) {
		return first
	}

	return "anonymous"
}

// In Echo, middleware is after the handler (args 3+):
// e.GET("/path", handler, mw1, mw2)
func extractEchoMiddleware(stmt, routePath string) []string {
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
	for _, arg := range args[1:] {
		arg = strings.TrimSpace(arg)
		re := regexp.MustCompile(`^[\w.]+$`)
		if re.MatchString(arg) && !isGoKeyword(arg) {
			middleware = append(middleware, arg)
		}
	}

	return middleware
}
