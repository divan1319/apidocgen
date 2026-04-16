package golang

import (
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/pkg/models"
)

func init() {
	parser.Register("gohttp", func(root string) parser.Parser {
		return NewNetHTTP(root)
	})
}

type NetHTTPParser struct {
	baseParser
}

func NewNetHTTP(projectRoot string) *NetHTTPParser {
	return &NetHTTPParser{baseParser{projectRoot: projectRoot}}
}

func (p *NetHTTPParser) Language() string { return "gohttp" }

func (p *NetHTTPParser) ParseSections(files []string) ([]models.RouteSection, error) {
	return p.parseSections(files, p.parseFile)
}

func (p *NetHTTPParser) parseFile(src string) []models.Endpoint {
	src = stripGoComments(src)

	var endpoints []models.Endpoint
	endpoints = append(endpoints, parseGoHTTPHandleFunc(src)...)
	endpoints = append(endpoints, parseGoHTTPHandle(src)...)
	endpoints = append(endpoints, parseGo122MethodRoutes(src)...)

	return deduplicateEndpoints(endpoints)
}

// ── http.HandleFunc / mux.HandleFunc (pre-1.22) ─────────────────────────────

// Matches: http.HandleFunc("/path", handler)
//          mux.HandleFunc("/path", handler)
//          serveMux.HandleFunc("/path", handler)
var handleFuncRe = regexp.MustCompile(
	`(\w+)\.HandleFunc\s*\(\s*"([^"]+)"\s*,\s*([\w.]+)`)

func parseGoHTTPHandleFunc(src string) []models.Endpoint {
	var endpoints []models.Endpoint

	for _, m := range handleFuncRe.FindAllStringSubmatch(src, -1) {
		callerVar := m[1]
		pattern := m[2]
		handler := m[3]

		method, path := parseGoHTTPPattern(pattern)

		ep := models.Endpoint{
			Method:     method,
			URI:        path,
			Controller: callerVar,
			Action:     handler,
			RawSource:  strings.TrimSpace(m[0]),
		}
		ep.StaticMeta.RequestParams = extractGoPathParams(path)
		endpoints = append(endpoints, ep)
	}

	return endpoints
}

// ── http.Handle / mux.Handle ─────────────────────────────────────────────────

var handleRe = regexp.MustCompile(
	`(\w+)\.Handle\s*\(\s*"([^"]+)"\s*,\s*([\w.]+)`)

func parseGoHTTPHandle(src string) []models.Endpoint {
	var endpoints []models.Endpoint

	for _, m := range handleRe.FindAllStringSubmatch(src, -1) {
		callerVar := m[1]
		pattern := m[2]
		handler := m[3]

		method, path := parseGoHTTPPattern(pattern)

		ep := models.Endpoint{
			Method:     method,
			URI:        path,
			Controller: callerVar,
			Action:     handler,
			RawSource:  strings.TrimSpace(m[0]),
		}
		ep.StaticMeta.RequestParams = extractGoPathParams(path)
		endpoints = append(endpoints, ep)
	}

	return endpoints
}

// ── Go 1.22+ patterns: "GET /api/users/{id}" ────────────────────────────────

// Matches patterns with explicit HTTP method: mux.HandleFunc("GET /path", ...)
var go122MethodRe = regexp.MustCompile(
	`(\w+)\.HandleFunc\s*\(\s*"(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\s+([^"]+)"\s*,\s*([\w.]+)`)

func parseGo122MethodRoutes(src string) []models.Endpoint {
	var endpoints []models.Endpoint

	for _, m := range go122MethodRe.FindAllStringSubmatch(src, -1) {
		callerVar := m[1]
		method := m[2]
		path := m[3]
		handler := m[4]

		ep := models.Endpoint{
			Method:     method,
			URI:        path,
			Controller: callerVar,
			Action:     handler,
			RawSource:  strings.TrimSpace(m[0]),
		}
		ep.StaticMeta.RequestParams = extractGoPathParams(path)
		endpoints = append(endpoints, ep)
	}

	return endpoints
}

// parseGoHTTPPattern splits a Go net/http pattern into method and path.
// Go 1.22+ supports "METHOD /path" patterns; older patterns are just "/path".
var goHTTPPatternRe = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\s+(.+)$`)

func parseGoHTTPPattern(pattern string) (method, path string) {
	pattern = strings.TrimSpace(pattern)
	if m := goHTTPPatternRe.FindStringSubmatch(pattern); m != nil {
		return m[1], m[2]
	}
	return "ALL", pattern
}
