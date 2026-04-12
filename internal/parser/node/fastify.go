package node

import (
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/pkg/models"
)

func init() {
	parser.Register("fastify", func(root string) parser.Parser {
		return NewFastify(root)
	})
}

type FastifyParser struct {
	baseParser
}

func NewFastify(projectRoot string) *FastifyParser {
	return &FastifyParser{baseParser{projectRoot: projectRoot}}
}

func (p *FastifyParser) Language() string { return "fastify" }

func (p *FastifyParser) ParseSections(files []string) ([]models.RouteSection, error) {
	return p.parseSections(files, p.parseFile)
}

func (p *FastifyParser) parseFile(src string) []models.Endpoint {
	src = stripJSComments(src)

	prefixes := buildFastifyPrefixes(src)

	var endpoints []models.Endpoint
	endpoints = append(endpoints, parseMethodRoutes(src, prefixes)...)
	endpoints = append(endpoints, parseFastifyRouteObjects(src, prefixes)...)

	return deduplicateEndpoints(endpoints)
}

// ── Fastify prefix resolution: fastify.register(plugin, { prefix: '/api' }) ──

var fastifyRegisterRe = regexp.MustCompile(
	`(\w+)\.register\s*\(\s*(\w+)\s*,\s*\{[^}]*prefix\s*:\s*['"]([^'"]+)['"]`)

func buildFastifyPrefixes(src string) map[string]string {
	prefixes := make(map[string]string)

	for _, m := range fastifyRegisterRe.FindAllStringSubmatch(src, -1) {
		handlerVar := m[2]
		prefix := m[3]
		prefixes[handlerVar] = prefix
	}

	return prefixes
}

// ── Fastify route objects: fastify.route({ method, url, handler }) ───────────

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
