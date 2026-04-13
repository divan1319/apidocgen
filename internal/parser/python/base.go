package python

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/pkg/models"
)

// ── Base parser (compartido FastAPI / Flask) ───────────────────────────────

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

	var out []string
	err = filepath.Walk(path, func(walkPath string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			name := fi.Name()
			if name == ".git" || name == "__pycache__" || name == ".venv" || name == "venv" ||
				name == "node_modules" || name == ".tox" || name == "dist" || name == "build" ||
				name == ".mypy_cache" || name == "site-packages" || name == ".eggs" {
				return filepath.SkipDir
			}
			if name == "tests" || name == "test" {
				return filepath.SkipDir
			}
			return nil
		}
		if isPyFile(fi.Name()) {
			out = append(out, walkPath)
		}
		return nil
	})
	return out, err
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
			endpoints[i].Language = "python"
		}

		sections = append(sections, models.RouteSection{
			FilePath:  f,
			Version:   inferVersion(f),
			Endpoints: endpoints,
		})
	}

	return sections, nil
}

func isPyFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".py")
}

func inferVersion(path string) string {
	re := regexp.MustCompile(`(?i)[/\\](v\d+)[/\\]`)
	m := re.FindStringSubmatch(path)
	if m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

func stripPythonComments(s string) string {
	// Elimina comentarios # hasta fin de línea (no contempla # dentro de strings).
	re := regexp.MustCompile(`(?m)#.*$`)
	return re.ReplaceAllString(s, "")
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

// extractPythonPathParams reconoce {id}, {id:int} (FastAPI) y <int:id>, <id> (Flask).
// Los dos puntos dentro de llaves no se tratan como estilo Express ":param".
func extractPythonPathParams(path string) []models.Param {
	seen := map[string]bool{}
	var params []models.Param

	braceRe := regexp.MustCompile(`\{([^}:]+)(?::[^}]*)?\}`)
	for _, m := range braceRe.FindAllStringSubmatch(path, -1) {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		params = append(params, models.Param{
			Name:     name,
			Type:     "string",
			Required: true,
			Rules:    "route",
		})
	}

	masked := braceRe.ReplaceAllString(path, "")
	angleRe := regexp.MustCompile(`<(?:\w+:)?(\w+)>`)
	for _, m := range angleRe.FindAllStringSubmatch(masked, -1) {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		params = append(params, models.Param{
			Name:     name,
			Type:     "string",
			Required: true,
			Rules:    "route",
		})
	}

	masked = angleRe.ReplaceAllString(masked, "")
	colonRe := regexp.MustCompile(`:(\w+)`)
	for _, m := range colonRe.FindAllStringSubmatch(masked, -1) {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		params = append(params, models.Param{
			Name:     name,
			Type:     "string",
			Required: true,
			Rules:    "route",
		})
	}

	if len(params) == 0 {
		return nil
	}
	return params
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
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, ":") {
			continue
		}
		if strings.Contains(part, "{") || strings.Contains(part, "<") {
			continue
		}
		name += capitalize(part)
	}
	if name == "" {
		name = "Root"
	}
	return strings.ToUpper(method) + "_" + name
}

var testReceiverVars = map[string]bool{
	"test": true, "client": true, "async_client": true, "test_client": true,
	"pytest": true, "mock": true,
}

func isTestReceiver(name string) bool {
	return testReceiverVars[name] || strings.HasPrefix(name, "test_")
}

var stringLiteralRe = regexp.MustCompile(`["']([^"'\\]*(?:\\.[^"'\\]*)*)["']`)

func firstStringArg(s string) string {
	m := stringLiteralRe.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return m[1]
}

// extractMethodsFromSequence parsea ['GET','POST'] o ("PUT",) en una cadena ya recortada.
func extractMethodsFromSequence(seq string) []string {
	seq = strings.TrimSpace(seq)
	var out []string
	inner := strings.Trim(seq, "[]()")
	for _, part := range strings.Split(inner, ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part == "" {
			continue
		}
		out = append(out, strings.ToUpper(part))
	}
	return out
}

// balancedSegment devuelve s[0:end+1] donde s[0] es open y end cierra el delimitador respetando anidación.
func balancedSegment(s string, open, close byte) (segment string, ok bool) {
	if len(s) == 0 || s[0] != open {
		return "", false
	}
	depth := 0
	for i := 0; i < len(s); i++ {
		if s[i] == open {
			depth++
		} else if s[i] == close {
			depth--
			if depth == 0 {
				return s[:i+1], true
			}
		}
	}
	return "", false
}

func findFollowingDefName(lines []string, startLine int) string {
	for i := startLine + 1; i < len(lines) && i < startLine+40; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "@") {
			continue
		}
		m := regexp.MustCompile(`^(?:async\s+)?def\s+(\w+)\s*\(`).FindStringSubmatch(line)
		if m != nil {
			return m[1]
		}
		break
	}
	return ""
}

func extractDependsNames(line string) []string {
	var names []string
	re := regexp.MustCompile(`Depends\s*\(\s*(\w+)\s*\)`)
	for _, m := range re.FindAllStringSubmatch(line, -1) {
		names = append(names, m[1])
	}
	return names
}

func findMatchingParen(s string, openPos int) int {
	if openPos < 0 || openPos >= len(s) || s[openPos] != '(' {
		return -1
	}
	depth := 0
	for i := openPos; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitTopLevelComma(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	if start <= len(s) {
		parts = append(parts, strings.TrimSpace(s[start:]))
	}
	return parts
}

func extractKeywordArg(callInner, key string) string {
	re := regexp.MustCompile(key + `\s*=\s*["']([^"']*)["']`)
	if m := re.FindStringSubmatch(callInner); m != nil {
		return m[1]
	}
	return ""
}

func extractParenBlock(lines []string, startLine, openParenCol int) string {
	var sb strings.Builder
	depth := 0
	for li := startLine; li < len(lines); li++ {
		line := lines[li]
		start := 0
		if li == startLine {
			start = openParenCol
		}
		for j := start; j < len(line); j++ {
			ch := line[j]
			sb.WriteByte(ch)
			if ch == '(' {
				depth++
			} else if ch == ')' {
				depth--
				if depth == 0 {
					inner := sb.String()
					if len(inner) >= 2 {
						return inner[1 : len(inner)-1]
					}
					return ""
				}
			}
		}
		if li < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}
	return ""
}

func collectDecoratorRawSource(lines []string, startLine, decoStart int) string {
	var sb strings.Builder
	depth := 0
	started := false
	for li := startLine; li < len(lines); li++ {
		line := lines[li]
		start := 0
		if li == startLine {
			start = decoStart
		}
		for j := start; j < len(line); j++ {
			ch := line[j]
			sb.WriteByte(ch)
			if ch == '(' {
				depth++
				started = true
			} else if ch == ')' && started {
				depth--
			}
			if started && depth == 0 {
				return strings.TrimSpace(sb.String())
			}
		}
		if li < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}
	return strings.TrimSpace(sb.String())
}
