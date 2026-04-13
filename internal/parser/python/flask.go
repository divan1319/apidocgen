package python

import (
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/pkg/models"
)

func init() {
	parser.Register("flask", func(root string) parser.Parser {
		return NewFlask(root)
	})
}

type FlaskParser struct {
	baseParser
}

func NewFlask(projectRoot string) *FlaskParser {
	return &FlaskParser{baseParser{projectRoot: projectRoot}}
}

func (p *FlaskParser) Language() string { return "flask" }

func (p *FlaskParser) ParseSections(files []string) ([]models.RouteSection, error) {
	return p.parseSections(files, p.parseFile)
}

func (p *FlaskParser) parseFile(src string) []models.Endpoint {
	src = stripPythonComments(src)
	own := parseBlueprintOwnPrefixes(src)
	edges := parseRegisterBlueprintEdges(src)
	routeBases := computeFlaskRouteBases(src, own, edges)
	endpoints := parseFlaskDecorators(src, routeBases)
	endpoints = append(endpoints, parseFlaskAddURLRule(src, routeBases)...)
	return deduplicateEndpoints(endpoints)
}

func parseBlueprintOwnPrefixes(src string) map[string]string {
	own := make(map[string]string)
	re := regexp.MustCompile(`(\w+)\s*=\s*Blueprint\s*\(`)
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
		own[v] = extractKeywordArg(inner, "url_prefix")
	}

	reFlask := regexp.MustCompile(`(\w+)\s*=\s*Flask\s*\(`)
	for _, m := range reFlask.FindAllStringSubmatch(src, -1) {
		own[m[1]] = ""
	}

	return own
}

type registerEdge struct {
	app, blueprint, urlPrefix string
}

func parseRegisterBlueprintEdges(src string) []registerEdge {
	var edges []registerEdge
	re := regexp.MustCompile(`(\w+)\.register_blueprint\s*\(`)
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
		app := src[m[2]:m[3]]
		inner := src[open+1 : close]
		args := splitTopLevelComma(inner)
		if len(args) == 0 {
			continue
		}
		bp := strings.TrimSpace(args[0])
		prefix := extractKeywordArg(inner, "url_prefix")
		edges = append(edges, registerEdge{app: app, blueprint: bp, urlPrefix: prefix})
	}
	return edges
}

func computeFlaskRouteBases(src string, own map[string]string, edges []registerEdge) map[string]string {
	routeBase := make(map[string]string)
	included := map[string]bool{}
	for _, e := range edges {
		included[e.blueprint] = true
	}

	reFlask := regexp.MustCompile(`(\w+)\s*=\s*Flask\s*\(`)
	for _, m := range reFlask.FindAllStringSubmatch(src, -1) {
		routeBase[m[1]] = ""
	}

	reBP := regexp.MustCompile(`(\w+)\s*=\s*Blueprint\s*\(`)
	for _, m := range reBP.FindAllStringSubmatch(src, -1) {
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
			pb, ok := routeBase[e.app]
			if !ok {
				continue
			}
			next := joinPath(joinPath(pb, strings.Trim(e.urlPrefix, "/")), strings.Trim(own[e.blueprint], "/"))
			if _, exists := routeBase[e.blueprint]; !exists {
				routeBase[e.blueprint] = next
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
	flaskVerbDeco = regexp.MustCompile(
		`@(\w+)\.(get|post|put|patch|delete|options|head)\s*\(`)
	flaskRouteDeco = regexp.MustCompile(`@(\w+)\.route\s*\(`)
)

func parseFlaskDecorators(src string, routeBases map[string]string) []models.Endpoint {
	lines := strings.Split(src, "\n")
	var endpoints []models.Endpoint

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if idx := flaskVerbDeco.FindStringIndex(line); idx != nil {
			m := flaskVerbDeco.FindStringSubmatch(line)
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

			fn := findFollowingDefName(lines, i)
			action := fn
			if action == "" {
				action = deriveActionName(method, uri)
			}

			ep := models.Endpoint{
				Method:     method,
				URI:        uri,
				Controller: receiver,
				Action:     action,
				RawSource:  collectDecoratorRawSource(lines, i, idx[0]),
			}
			ep.StaticMeta.RequestParams = extractPythonPathParams(uri)
			endpoints = append(endpoints, ep)
			continue
		}

		if idx := flaskRouteDeco.FindStringIndex(line); idx != nil {
			m := flaskRouteDeco.FindStringSubmatch(line)
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
					RawSource:  raw,
				}
				ep.StaticMeta.RequestParams = extractPythonPathParams(uri)
				endpoints = append(endpoints, ep)
			}
		}
	}

	return endpoints
}

var flaskAddURLRuleRe = regexp.MustCompile(`(\w+)\.add_url_rule\s*\(`)

func parseFlaskAddURLRule(src string, routeBases map[string]string) []models.Endpoint {
	var endpoints []models.Endpoint
	lines := strings.Split(src, "\n")

	for _, line := range lines {
		idx := flaskAddURLRuleRe.FindStringSubmatchIndex(line)
		if idx == nil {
			continue
		}
		receiver := line[idx[2]:idx[3]]
		if isTestReceiver(receiver) {
			continue
		}
		sub := line[idx[0]:idx[1]]
		pIdx := strings.LastIndex(sub, "(")
		if pIdx < 0 {
			continue
		}
		open := idx[0] + pIdx
		close := findMatchingParen(line, open)
		if close < 0 {
			continue
		}
		inner := line[open+1 : close]
		args := splitTopLevelComma(inner)
		if len(args) == 0 {
			continue
		}
		arg0 := strings.TrimSpace(args[0])
		path := firstStringArg(arg0)
		if path == "" {
			path = strings.Trim(arg0, `"'`)
		}
		if path == "" {
			continue
		}
		base := routeBases[receiver]
		uri := joinPath(strings.Trim(base, "/"), strings.Trim(path, "/"))

		methods := parseMethodsKeyword(inner)
		if len(methods) == 0 {
			methods = []string{"GET"}
		}

		for _, method := range methods {
			ep := models.Endpoint{
				Method:     method,
				URI:        uri,
				Controller: receiver,
				Action:     deriveActionName(method, uri),
				RawSource:  strings.TrimSpace(line[idx[0]:close+1]),
			}
			ep.StaticMeta.RequestParams = extractPythonPathParams(uri)
			endpoints = append(endpoints, ep)
		}
	}

	return endpoints
}
