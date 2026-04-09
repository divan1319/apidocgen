package laravel

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/pkg/models"
)

type LaravelParser struct {
	// projectRoot is used to resolve controller file paths
	projectRoot string
}

func New(projectRoot string) *LaravelParser {
	return &LaravelParser{projectRoot: projectRoot}
}

func (p *LaravelParser) Language() string { return "laravel" }

// routePattern matches both styles:
//
//	Route::get('/users', [UserController::class, 'index']);
//	Route::post('/users', 'UserController@store');
var routePattern = regexp.MustCompile(
	`Route::(get|post|put|patch|delete|options)\s*\(\s*['"]([^'"]+)['"]\s*,\s*(?:\[([^\]]+)\]|'([^']+)')`,
)

// middlewarePattern matches ->middleware('auth') or ->middleware(['auth', 'throttle:60,1'])
var middlewarePattern = regexp.MustCompile(`->middleware\(\s*(?:\[([^\]]*)\]|'([^']*)')\s*\)`)

// validatePattern matches $request->validate([...]) inline
var validatePattern = regexp.MustCompile(`\$request->validate\(\s*\[([\s\S]*?)\]\s*\)`)

// ruleLinePattern matches 'field' => 'required|string|max:255',
var ruleLinePattern = regexp.MustCompile(`['"](\w+)['"]\s*=>\s*['"]([^'"]+)['"]`)

func (p *LaravelParser) Parse(files []string) ([]models.Endpoint, error) {
	var endpoints []models.Endpoint

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f, err)
		}

		parsed, err := p.parseRouteFile(string(content))
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", f, err)
		}
		endpoints = append(endpoints, parsed...)
	}

	return endpoints, nil
}

func (p *LaravelParser) parseRouteFile(content string) ([]models.Endpoint, error) {
	matches := routePattern.FindAllStringSubmatch(content, -1)
	var endpoints []models.Endpoint

	for _, m := range matches {
		method := strings.ToUpper(m[1])
		uri := m[2]

		controller, action := p.resolveHandler(m[3], m[4])
		middleware := p.extractMiddleware(content, uri)

		ep := models.Endpoint{
			Method:     method,
			URI:        uri,
			Controller: controller,
			Action:     action,
			Middleware: middleware,
		}

		// Try to read controller source and enrich the endpoint
		ep.RawSource = p.readControllerSource(controller, action)
		ep.StaticMeta = p.extractStaticMeta(ep.RawSource)

		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}

// resolveHandler normalizes both [Controller::class, 'method'] and 'Controller@method'
func (p *LaravelParser) resolveHandler(arrayStyle, atStyle string) (controller, action string) {
	if arrayStyle != "" {
		// [App\Http\Controllers\UserController::class, 'index']
		parts := strings.Split(arrayStyle, ",")
		if len(parts) == 2 {
			controller = strings.TrimSpace(parts[0])
			controller = strings.ReplaceAll(controller, "::class", "")
			controller = strings.Trim(controller, `'"`)
			action = strings.Trim(strings.TrimSpace(parts[1]), `'"`)
		}
	} else if atStyle != "" {
		// UserController@index
		parts := strings.SplitN(atStyle, "@", 2)
		controller = parts[0]
		if len(parts) == 2 {
			action = parts[1]
		}
	}
	return
}

func (p *LaravelParser) extractMiddleware(content, uri string) []string {
	// Simple heuristic: find middleware near this route definition
	// A more robust implementation would parse route groups
	matches := middlewarePattern.FindAllStringSubmatch(content, -1)
	var middleware []string
	for _, m := range matches {
		raw := m[1] + m[2]
		for _, mw := range strings.Split(raw, ",") {
			mw = strings.Trim(strings.TrimSpace(mw), `'"`)
			if mw != "" {
				middleware = append(middleware, mw)
			}
		}
	}
	return middleware
}

// readControllerSource finds the controller file and extracts the method's source
func (p *LaravelParser) readControllerSource(controller, action string) string {
	if controller == "" || action == "" || p.projectRoot == "" {
		return ""
	}

	// Convert namespace to path: App\Http\Controllers\UserController → app/Http/Controllers/UserController.php
	relative := strings.ReplaceAll(controller, `\`, string(filepath.Separator))
	// Laravel convention: App\ maps to app/
	relative = strings.Replace(relative, "App"+string(filepath.Separator), "app"+string(filepath.Separator), 1)
	filePath := filepath.Join(p.projectRoot, relative+".php")

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "" // controller not found, AI will work with less context
	}

	return extractMethodSource(string(content), action)
}

// extractMethodSource pulls just the relevant method from a PHP class
func extractMethodSource(classContent, method string) string {
	// Find "public function <method>("
	start := strings.Index(classContent, "public function "+method+"(")
	if start == -1 {
		return classContent // fallback: return whole class
	}

	// Walk forward counting braces to find end of method
	depth := 0
	inMethod := false
	for i := start; i < len(classContent); i++ {
		switch classContent[i] {
		case '{':
			depth++
			inMethod = true
		case '}':
			depth--
			if inMethod && depth == 0 {
				return classContent[start : i+1]
			}
		}
	}

	return classContent[start:]
}

func (p *LaravelParser) extractStaticMeta(source string) models.StaticMeta {
	if source == "" {
		return models.StaticMeta{}
	}

	meta := models.StaticMeta{}

	// Extract inline validation rules
	vm := validatePattern.FindStringSubmatch(source)
	if len(vm) > 1 {
		ruleLines := ruleLinePattern.FindAllStringSubmatch(vm[1], -1)
		for _, rl := range ruleLines {
			name := rl[1]
			rules := rl[2]
			required := strings.Contains(rules, "required")
			ptype := inferTypeFromRules(rules)
			meta.RequestParams = append(meta.RequestParams, models.Param{
				Name:     name,
				Type:     ptype,
				Required: required,
				Rules:    rules,
			})
		}
	}

	return meta
}

func inferTypeFromRules(rules string) string {
	switch {
	case strings.Contains(rules, "integer") || strings.Contains(rules, "numeric"):
		return "integer"
	case strings.Contains(rules, "boolean"):
		return "boolean"
	case strings.Contains(rules, "array"):
		return "array"
	case strings.Contains(rules, "file") || strings.Contains(rules, "image"):
		return "file"
	case strings.Contains(rules, "email"):
		return "string (email)"
	default:
		return "string"
	}
}
