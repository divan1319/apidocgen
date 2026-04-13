package python

import (
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/pkg/models"
)

func init() {
	parser.Register("fastapi", func(root string) parser.Parser {
		return NewFastAPI(root)
	})
}

type FastAPIParser struct {
	baseParser
}

func NewFastAPI(projectRoot string) *FastAPIParser {
	return &FastAPIParser{baseParser{projectRoot: projectRoot}}
}

func (p *FastAPIParser) Language() string { return "fastapi" }

func (p *FastAPIParser) ParseSections(files []string) ([]models.RouteSection, error) {
	return p.parseSections(files, p.parseFile)
}

func (p *FastAPIParser) parseFile(src string) []models.Endpoint {
	src = stripPythonComments(src)
	own := parseAPIRouterOwnPrefixes(src)
	edges := parseIncludeRouterEdges(src)
	routeBases := computeFastAPIRouteBases(src, own, edges)
	return deduplicateEndpoints(parseFastAPIDecorators(src, routeBases))
}

func parseAPIRouterOwnPrefixes(src string) map[string]string {
	own := make(map[string]string)
	re := regexp.MustCompile(`(\w+)\s*=\s*APIRouter\s*\(`)
	for _, m := range re.FindAllStringSubmatchIndex(src, -1) {
		fullStart, fullEnd := m[0], m[1]
		sub := src[fullStart:fullEnd]
		pIdx := strings.LastIndex(sub, "(")
		if pIdx < 0 {
			continue
		}
		open := fullStart + pIdx
		close := findMatchingParen(src, open)
		if close < 0 {
			continue
		}
		inner := src[open+1 : close]
		v := src[m[2]:m[3]]
		own[v] = extractKeywordArg(inner, "prefix")
	}

	reFast := regexp.MustCompile(`(\w+)\s*=\s*FastAPI\s*\(`)
	for _, m := range reFast.FindAllStringSubmatch(src, -1) {
		own[m[1]] = ""
	}

	return own
}

type includeEdge struct {
	parent, child, prefix string
}

func parseIncludeRouterEdges(src string) []includeEdge {
	var edges []includeEdge
	re := regexp.MustCompile(`(\w+)\.include_router\s*\(`)
	for _, m := range re.FindAllStringSubmatchIndex(src, -1) {
		fullStart, fullEnd := m[0], m[1]
		sub := src[fullStart:fullEnd]
		pIdx := strings.LastIndex(sub, "(")
		if pIdx < 0 {
			continue
		}
		open := fullStart + pIdx
		close := findMatchingParen(src, open)
		if close < 0 {
			continue
		}
		parent := src[m[2]:m[3]]
		inner := src[open+1 : close]
		args := splitTopLevelComma(inner)
		if len(args) == 0 {
			continue
		}
		child := strings.TrimSpace(args[0])
		prefix := extractKeywordArg(inner, "prefix")
		edges = append(edges, includeEdge{parent: parent, child: child, prefix: prefix})
	}
	return edges
}

func computeFastAPIRouteBases(src string, own map[string]string, edges []includeEdge) map[string]string {
	routeBase := make(map[string]string)
	included := map[string]bool{}
	for _, e := range edges {
		included[e.child] = true
	}

	reFast := regexp.MustCompile(`(\w+)\s*=\s*FastAPI\s*\(`)
	for _, m := range reFast.FindAllStringSubmatch(src, -1) {
		routeBase[m[1]] = ""
	}

	reAPI := regexp.MustCompile(`(\w+)\s*=\s*APIRouter\s*\(`)
	for _, m := range reAPI.FindAllStringSubmatch(src, -1) {
		v := m[1]
		if !included[v] {
			pfx := strings.Trim(own[v], "/")
			if pfx == "" {
				routeBase[v] = ""
			} else {
				routeBase[v] = joinPath("", pfx)
			}
		}
	}

	for {
		changed := false
		for _, e := range edges {
			pb, ok := routeBase[e.parent]
			if !ok {
				continue
			}
			next := joinPath(joinPath(pb, strings.Trim(e.prefix, "/")), strings.Trim(own[e.child], "/"))
			if _, exists := routeBase[e.child]; !exists {
				routeBase[e.child] = next
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return routeBase
}

var (
	fastapiVerbDeco = regexp.MustCompile(
		`@(\w+)\.(get|post|put|patch|delete|options|head|trace|websocket)\s*\(`)
	fastapiAPIRouteDeco = regexp.MustCompile(`@(\w+)\.api_route\s*\(`)
)

func parseFastAPIDecorators(src string, routeBases map[string]string) []models.Endpoint {
	lines := strings.Split(src, "\n")
	var endpoints []models.Endpoint

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if idx := fastapiVerbDeco.FindStringIndex(line); idx != nil {
			m := fastapiVerbDeco.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			receiver := m[1]
			if isTestReceiver(receiver) {
				continue
			}
			openParen := strings.Index(line[idx[0]:], "(") + idx[0]
			inner := extractParenBlock(lines, i, openParen)
			if inner == "" {
				continue
			}
			path := firstStringArg(inner)
			if path == "" {
				continue
			}
			base := routeBases[receiver]
			uri := joinPath(strings.Trim(base, "/"), strings.Trim(path, "/"))

			method := strings.ToUpper(m[2])
			if method == "WEBSOCKET" {
				method = "WEBSOCKET"
			}

			fn := findFollowingDefName(lines, i)
			action := fn
			if action == "" {
				action = deriveActionName(method, uri)
			}

			mw := extractDependsNames(line)
			if len(mw) == 0 {
				mw = extractDependsNames(inner)
			}

			ep := models.Endpoint{
				Method:     method,
				URI:        uri,
				Controller: receiver,
				Action:     action,
				Middleware: mw,
				RawSource:  collectDecoratorRawSource(lines, i, idx[0]),
			}
			ep.StaticMeta.RequestParams = extractPythonPathParams(uri)
			endpoints = append(endpoints, ep)
			continue
		}

		if idx := fastapiAPIRouteDeco.FindStringIndex(line); idx != nil {
			m := fastapiAPIRouteDeco.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			receiver := m[1]
			if isTestReceiver(receiver) {
				continue
			}
			openParen := strings.Index(line[idx[0]:], "(") + idx[0]
			inner := extractParenBlock(lines, i, openParen)
			if inner == "" {
				continue
			}
			path := firstStringArg(inner)
			if path == "" {
				continue
			}
			base := routeBases[receiver]
			uri := joinPath(strings.Trim(base, "/"), strings.Trim(path, "/"))

			methods := parseMethodsKeyword(inner)
			if len(methods) == 0 {
				methods = []string{"GET"}
			}

			fn := findFollowingDefName(lines, i)
			mw := extractDependsNames(line)
			if len(mw) == 0 {
				mw = extractDependsNames(inner)
			}

			raw := collectDecoratorRawSource(lines, i, idx[0])
			for _, method := range methods {
				action := fn
				if action == "" {
					action = deriveActionName(method, uri)
				}
				ep := models.Endpoint{
					Method:     method,
					URI:        uri,
					Controller: receiver,
					Action:     action,
					Middleware: mw,
					RawSource:  raw,
				}
				ep.StaticMeta.RequestParams = extractPythonPathParams(uri)
				endpoints = append(endpoints, ep)
			}
		}
	}

	return endpoints
}

func parseMethodsKeyword(inner string) []string {
	idx := strings.Index(inner, "methods")
	if idx < 0 {
		return nil
	}
	rest := strings.TrimSpace(inner[idx:])
	rest = regexp.MustCompile(`^methods\s*=\s*`).ReplaceAllString(rest, "")
	rest = strings.TrimSpace(rest)
	if len(rest) == 0 {
		return nil
	}
	switch rest[0] {
	case '[':
		seg, ok := balancedSegment(rest, '[', ']')
		if !ok {
			return nil
		}
		return extractMethodsFromSequence(seg)
	case '(':
		seg, ok := balancedSegment(rest, '(', ')')
		if !ok {
			return nil
		}
		return extractMethodsFromSequence(seg)
	default:
		return nil
	}
}
