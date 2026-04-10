package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/divan1319/apidocgen/internal/ai"
	"github.com/divan1319/apidocgen/internal/generator"
	"github.com/divan1319/apidocgen/internal/parser/laravel"
	"github.com/divan1319/apidocgen/pkg/models"
)

func main() {
	genCmd := flag.NewFlagSet("generate", flag.ExitOnError)

	lang := genCmd.String("lang", "laravel", "Language/framework: laravel (more coming soon)")
	routes := genCmd.String("routes", "", "Comma-separated route files (e.g. routes/api.php)")
	root := genCmd.String("root", ".", "Project root directory")
	output := genCmd.String("output", "api-docs.html", "Output HTML file")
	title := genCmd.String("title", "API Documentation", "Documentation title")
	apiKey := genCmd.String("api-key", "", "Anthropic API key (or set ANTHROPIC_API_KEY env var)")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		genCmd.Parse(os.Args[2:])
		runGenerate(*lang, *routes, *root, *output, *title, *apiKey)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runGenerate(lang, routes, root, output, title, apiKey string) {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		fatal("API key required: use --api-key or set ANTHROPIC_API_KEY")
	}
	if routes == "" {
		fatal("--routes is required (e.g. --routes routes/api.php)")
	}

	files := splitTrim(routes, ",")

	switch lang {
	case "laravel":
		runLaravel(files, root, output, title, apiKey)
	default:
		fatal("unsupported language: %s (supported: laravel)", lang)
	}
}

func runLaravel(files []string, root, output, title, apiKey string) {
	p := laravel.New(root)

	// ── Step 1: resolve all included files ───────────────────────────────────
	fmt.Println("→ Resolving includes...")
	allFiles, err := p.ResolveIncludes(files)
	if err != nil {
		fatal("resolving includes: %v", err)
	}
	fmt.Printf("  Found %d file(s)\n", len(allFiles))

	// ── Step 2: prompt user to name each section ──────────────────────────────
	fmt.Println("\n→ Name each section (press Enter to use the suggested name):")
	reader := bufio.NewReader(os.Stdin)
	sectionNames := make(map[string]string, len(allFiles))

	for _, f := range allFiles {
		suggested := suggestSectionName(f)
		fmt.Printf("  %s [%s]: ", f, suggested)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			input = suggested
		}
		sectionNames[f] = input
	}

	// ── Step 3: parse sections ────────────────────────────────────────────────
	fmt.Println("\n→ Parsing routes...")
	sections, err := p.ParseSections(allFiles)
	if err != nil {
		fatal("parsing routes: %v", err)
	}

	// Apply names from CLI prompts
	for i := range sections {
		if name, ok := sectionNames[sections[i].FilePath]; ok {
			sections[i].Name = name
		}
	}

	totalEndpoints := 0
	for _, s := range sections {
		totalEndpoints += len(s.Endpoints)
	}
	fmt.Printf("  Found %d section(s), %d endpoint(s) total\n", len(sections), totalEndpoints)

	// ── Step 4: document with AI ──────────────────────────────────────────────
	fmt.Println("\n→ Generating documentation with Claude...")
	client := ai.New(apiKey)
	var sectionDocs []models.SectionDoc

	for _, section := range sections {
		fmt.Printf("\n  [%s]\n", section.Name)
		sd := models.SectionDoc{
			Name:     section.Name,
			Version:  section.Version,
			FilePath: section.FilePath,
		}

		for i, ep := range section.Endpoints {
			fmt.Printf("    [%d/%d] %s %s\n", i+1, len(section.Endpoints), ep.Method, ep.URI)
			doc, err := client.DocumentEndpoint(ep)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    warning: failed to document %s %s: %v\n", ep.Method, ep.URI, err)
				continue
			}
			sd.Docs = append(sd.Docs, *doc)
		}

		sectionDocs = append(sectionDocs, sd)
	}

	// ── Step 5: generate HTML ─────────────────────────────────────────────────
	fmt.Printf("\n→ Writing %s...\n", output)
	gen := generator.New()
	if err := gen.GenerateSections(sectionDocs, title, output); err != nil {
		fatal("generating HTML: %v", err)
	}

	fmt.Printf("✓ Done! Open %s in your browser.\n", output)
}

// suggestSectionName infers a human-readable name from a file path.
// api/v2/gestionAcademicaRoute.php → "Gestion Academica"
// api/inscripcionRoute.php         → "Inscripcion"
func suggestSectionName(path string) string {
	base := filepath.Base(path)
	// Remove extension
	base = strings.TrimSuffix(base, filepath.Ext(base))
	// Remove common suffixes
	for _, suffix := range []string{"Route", "Routes", "route", "routes"} {
		base = strings.TrimSuffix(base, suffix)
	}
	// Split camelCase into words
	re := regexp.MustCompile(`([A-Z][a-z]+|[a-z]+|[A-Z]+(?:[A-Z][a-z]*)*)`)
	words := re.FindAllString(base, -1)
	for i, w := range words {
		words[i] = strings.Title(strings.ToLower(w))
	}
	name := strings.Join(words, " ")

	// Append version if detected in path
	vre := regexp.MustCompile(`(?i)[/\\](v\d+)[/\\]`)
	if m := vre.FindStringSubmatch(path); m != nil {
		name += " " + strings.ToUpper(m[1])
	}

	return strings.TrimSpace(name)
}

func splitTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func printUsage() {
	fmt.Println(`apidocgen - AI-powered API documentation generator

USAGE:
  apidocgen generate [flags]

FLAGS:
  --lang      Language/framework (default: laravel)
  --routes    Comma-separated route files to parse
  --root      Project root directory (default: .)
  --output    Output HTML file (default: api-docs.html)
  --title     Documentation title (default: "API Documentation")
  --api-key   Anthropic API key (or use ANTHROPIC_API_KEY env var)

EXAMPLES:
  apidocgen generate --routes routes/api.php --root /path/to/laravel-project
  apidocgen generate --routes routes/api.php,routes/web.php --title "My API v2"
  ANTHROPIC_API_KEY=sk-... apidocgen generate --routes routes/api.php`)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
