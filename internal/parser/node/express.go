package node

import (
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/pkg/models"
)

func init() {
	parser.Register("express", func(root string) parser.Parser {
		return NewExpress(root)
	})
}

type ExpressParser struct {
	baseParser
}

func NewExpress(projectRoot string) *ExpressParser {
	return &ExpressParser{baseParser{projectRoot: projectRoot}}
}

func (p *ExpressParser) Language() string { return "express" }

func (p *ExpressParser) ParseSections(files []string) ([]models.RouteSection, error) {
	return p.parseSections(files, p.parseFile)
}

func (p *ExpressParser) parseFile(src string) []models.Endpoint {
	src = stripJSComments(src)

	prefixes := buildExpressPrefixes(src)

	var endpoints []models.Endpoint
	endpoints = append(endpoints, parseMethodRoutes(src, prefixes)...)
	endpoints = append(endpoints, parseRouteChains(src, prefixes)...)

	return deduplicateEndpoints(endpoints)
}

// ── Express router prefix resolution: app.use('/prefix', router) ─────────────

var routerMountRe = regexp.MustCompile(
	`(\w+)\.use\s*\(\s*['"]([^'"]+)['"]\s*,\s*(\w+)`)

func buildExpressPrefixes(src string) map[string]string {
	prefixes := make(map[string]string)

	for _, m := range routerMountRe.FindAllStringSubmatch(src, -1) {
		routerVar := m[3]
		prefix := m[2]
		prefixes[routerVar] = prefix
	}

	return prefixes
}

// ── Express route chains: app.route('/path').get().post() ────────────────────

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
