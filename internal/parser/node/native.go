package node

import (
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/pkg/models"
)

func init() {
	parser.Register("nodehttp", func(root string) parser.Parser {
		return NewNativeHTTP(root)
	})
}

type NativeHTTPParser struct {
	baseParser
}

func NewNativeHTTP(projectRoot string) *NativeHTTPParser {
	return &NativeHTTPParser{baseParser{projectRoot: projectRoot}}
}

func (p *NativeHTTPParser) Language() string { return "nodehttp" }

func (p *NativeHTTPParser) ParseSections(files []string) ([]models.RouteSection, error) {
	return p.parseSections(files, p.parseFile)
}

func (p *NativeHTTPParser) parseFile(src string) []models.Endpoint {
	src = stripJSComments(src)
	return deduplicateEndpoints(parseNativeHTTPRoutes(src))
}

// ── Native HTTP routes ───────────────────────────────────────────────────────

var (
	nativeMethodUrlRe = regexp.MustCompile(
		`req(?:uest)?\.method\s*===?\s*['"](\w+)['"]\s*&&\s*(?:req(?:uest)?\.url|url)\s*===?\s*['"]([^'"]+)['"]`)

	nativeUrlMethodRe = regexp.MustCompile(
		`(?:req(?:uest)?\.url|url)\s*===?\s*['"]([^'"]+)['"]\s*&&\s*req(?:uest)?\.method\s*===?\s*['"](\w+)['"]`)

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

	for _, m := range nativeMethodUrlRe.FindAllStringSubmatch(src, -1) {
		addEndpoint(m[1], m[2], m[0])
	}

	for _, m := range nativeUrlMethodRe.FindAllStringSubmatch(src, -1) {
		addEndpoint(m[2], m[1], m[0])
	}

	endpoints = append(endpoints, parseSwitchBlocks(src)...)

	return endpoints
}

func parseSwitchBlocks(src string) []models.Endpoint {
	var endpoints []models.Endpoint
	seen := map[string]bool{}

	urlSwitchLocs := nativeSwitchUrlRe.FindAllStringIndex(src, -1)
	for _, loc := range urlSwitchLocs {
		bracePos := loc[1] - 1
		body := extractBraceBlock(src, bracePos)

		for _, cm := range caseStringRe.FindAllStringSubmatch(body, -1) {
			url := cm[1]

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

	methodSwitchLocs := nativeSwitchMethodRe.FindAllStringIndex(src, -1)
	for _, loc := range methodSwitchLocs {
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
		if depth == 0 && i+5 < len(body) {
			rest := body[i:]
			if strings.HasPrefix(rest, "case ") || strings.HasPrefix(rest, "default:") {
				return body[start:i]
			}
		}
	}
	return body[start:]
}
