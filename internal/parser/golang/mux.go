package golang

import (
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/pkg/models"
)

func init() {
	parser.Register("gomux", func(root string) parser.Parser {
		return NewMux(root)
	})
}

type MuxParser struct {
	baseParser
}

func NewMux(projectRoot string) *MuxParser {
	return &MuxParser{baseParser{projectRoot: projectRoot}}
}

func (p *MuxParser) Language() string { return "gomux" }

func (p *MuxParser) ParseSections(files []string) ([]models.RouteSection, error) {
	return p.parseSections(files, p.parseFile)
}

func (p *MuxParser) parseFile(src string) []models.Endpoint {
	src = stripGoComments(src)

	prefixes := buildMuxPrefixes(src)
	var endpoints []models.Endpoint
	endpoints = append(endpoints, parseMuxHandleFunc(src, prefixes)...)
	endpoints = append(endpoints, parseMuxHandle(src, prefixes)...)

	return deduplicateEndpoints(endpoints)
}

// ── Subrouter prefix resolution ──────────────────────────────────────────────

// Matches: sub := r.PathPrefix("/api").Subrouter()
var subrouterRe = regexp.MustCompile(
	`(\w+)\s*(?::=|=)\s*(\w+)\.PathPrefix\s*\(\s*"([^"]+)"\s*\)(?:\.\w+\([^)]*\))*\.Subrouter\s*\(`)

func buildMuxPrefixes(src string) map[string]string {
	prefixes := make(map[string]string)

	for _, m := range subrouterRe.FindAllStringSubmatch(src, -1) {
		subVar := m[1]
		parentVar := m[2]
		prefix := m[3]

		parentPrefix := prefixes[parentVar]
		prefixes[subVar] = joinPath(parentPrefix, prefix)
	}

	return prefixes
}

// ── r.HandleFunc("/path", handler).Methods("GET") ───────────────────────────

// Captures the full chain including .Methods(...) and .Name(...)
var muxHandleFuncRe = regexp.MustCompile(
	`(\w+)\.HandleFunc\s*\(\s*"([^"]+)"\s*,\s*([\w.]+)\s*\)([^\n]*)`)

func parseMuxHandleFunc(src string, prefixes map[string]string) []models.Endpoint {
	var endpoints []models.Endpoint

	for _, m := range muxHandleFuncRe.FindAllStringSubmatch(src, -1) {
		callerVar := m[1]
		routePath := m[2]
		handler := m[3]
		chain := m[4]

		prefix := prefixes[callerVar]
		uri := joinPath(prefix, routePath)
		methods := extractMuxMethods(chain)
		name := extractMuxName(chain)

		if len(methods) == 0 {
			ep := models.Endpoint{
				Method:     "ALL",
				URI:        uri,
				Name:       name,
				Controller: callerVar,
				Action:     handler,
				RawSource:  strings.TrimSpace(m[0]),
			}
			ep.StaticMeta.RequestParams = extractGoPathParams(uri)
			endpoints = append(endpoints, ep)
		} else {
			for _, method := range methods {
				ep := models.Endpoint{
					Method:     method,
					URI:        uri,
					Name:       name,
					Controller: callerVar,
					Action:     handler,
					RawSource:  strings.TrimSpace(m[0]),
				}
				ep.StaticMeta.RequestParams = extractGoPathParams(uri)
				endpoints = append(endpoints, ep)
			}
		}
	}

	return endpoints
}

// ── r.Handle("/path", handler).Methods("GET") ───────────────────────────────

var muxHandleRe = regexp.MustCompile(
	`(\w+)\.Handle\s*\(\s*"([^"]+)"\s*,\s*([\w.()]+)\s*\)([^\n]*)`)

func parseMuxHandle(src string, prefixes map[string]string) []models.Endpoint {
	var endpoints []models.Endpoint

	for _, m := range muxHandleRe.FindAllStringSubmatch(src, -1) {
		callerVar := m[1]
		routePath := m[2]
		handler := m[3]
		chain := m[4]

		prefix := prefixes[callerVar]
		uri := joinPath(prefix, routePath)
		methods := extractMuxMethods(chain)
		name := extractMuxName(chain)

		if len(methods) == 0 {
			ep := models.Endpoint{
				Method:     "ALL",
				URI:        uri,
				Name:       name,
				Controller: callerVar,
				Action:     handler,
				RawSource:  strings.TrimSpace(m[0]),
			}
			ep.StaticMeta.RequestParams = extractGoPathParams(uri)
			endpoints = append(endpoints, ep)
		} else {
			for _, method := range methods {
				ep := models.Endpoint{
					Method:     method,
					URI:        uri,
					Name:       name,
					Controller: callerVar,
					Action:     handler,
					RawSource:  strings.TrimSpace(m[0]),
				}
				ep.StaticMeta.RequestParams = extractGoPathParams(uri)
				endpoints = append(endpoints, ep)
			}
		}
	}

	return endpoints
}

// ── Chain method extraction ──────────────────────────────────────────────────

// Matches: .Methods("GET", "POST")
var muxMethodsRe = regexp.MustCompile(
	`\.Methods\s*\(\s*([^)]+)\)`)

var quotedStringRe = regexp.MustCompile(`"(\w+)"`)

func extractMuxMethods(chain string) []string {
	m := muxMethodsRe.FindStringSubmatch(chain)
	if m == nil {
		return nil
	}

	var methods []string
	for _, qm := range quotedStringRe.FindAllStringSubmatch(m[1], -1) {
		methods = append(methods, strings.ToUpper(qm[1]))
	}
	return methods
}

// Matches: .Name("routeName")
var muxNameRe = regexp.MustCompile(`\.Name\s*\(\s*"([^"]+)"\s*\)`)

func extractMuxName(chain string) string {
	if m := muxNameRe.FindStringSubmatch(chain); m != nil {
		return m[1]
	}
	return ""
}
