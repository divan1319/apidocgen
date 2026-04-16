package golang

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/pkg/models"
)

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
			if name == "vendor" || name == ".git" || name == "testdata" ||
				name == "node_modules" || name == "internal" {
				return filepath.SkipDir
			}
			return nil
		}
		if isGoFile(fi.Name()) {
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

		for i := range endpoints {
			endpoints[i].Language = "go"
		}

		sections = append(sections, models.RouteSection{
			FilePath:  f,
			Version:   inferVersion(f),
			Endpoints: endpoints,
		})
	}

	return sections, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func isGoFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

var versionPathRe = regexp.MustCompile(`(?i)[/\\](v\d+)[/\\]`)

func inferVersion(path string) string {
	m := versionPathRe.FindStringSubmatch(path)
	if m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

func stripGoComments(src string) string {
	src = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(src, " ")
	src = regexp.MustCompile(`//[^\n]*`).ReplaceAllString(src, "")
	return src
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

// extractGoPathParams extracts path parameters in {param} or :param style.
var goPathParamRe = regexp.MustCompile(`\{(\w+)(?:\.\.\.)?\}|:(\w+)`)

func extractGoPathParams(path string) []models.Param {
	matches := goPathParamRe.FindAllStringSubmatch(path, -1)
	if len(matches) == 0 {
		return nil
	}
	var params []models.Param
	for _, m := range matches {
		name := m[1]
		if name == "" {
			name = m[2]
		}
		params = append(params, models.Param{
			Name:     name,
			Type:     "string",
			Required: true,
			Rules:    "route",
		})
	}
	return params
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

// extractHandlerName pulls the last identifier from a handler argument.
// e.g. "controllers.GetUsers" → "controllers.GetUsers", "handleUsers" → "handleUsers".
var handlerNameRe = regexp.MustCompile(`([\w.]+)\s*\)`)

func extractHandlerName(stmt string) string {
	if m := handlerNameRe.FindStringSubmatch(stmt); m != nil {
		name := m[1]
		if !isGoKeyword(name) {
			return name
		}
	}
	return ""
}

var goKeywords = map[string]bool{
	"func": true, "nil": true, "true": true, "false": true,
	"return": true, "if": true, "else": true, "for": true,
	"range": true, "var": true, "const": true, "type": true,
	"struct": true, "interface": true, "map": true, "chan": true,
	"go": true, "defer": true, "select": true, "switch": true,
	"case": true, "default": true, "break": true, "continue": true,
	"package": true, "import": true,
}

func isGoKeyword(s string) bool {
	return goKeywords[s]
}
