package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/divan1319/apidocgen/internal/ai"
	"github.com/divan1319/apidocgen/internal/generator"
	"github.com/divan1319/apidocgen/internal/parser/laravel"
	"github.com/divan1319/apidocgen/pkg/models"
)

func main() {
	// Subcommands
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
	// Resolve API key
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		fatal("API key required: use --api-key or set ANTHROPIC_API_KEY")
	}
	if routes == "" {
		fatal("--routes is required (e.g. --routes routes/api.php)")
	}

	files := strings.Split(routes, ",")
	for i := range files {
		files[i] = strings.TrimSpace(files[i])
	}

	// 1. Parse
	fmt.Printf("→ Parsing %s routes...\n", lang)
	var endpoints []models.Endpoint
	var err error

	switch lang {
	case "laravel":
		p := laravel.New(root)
		endpoints, err = p.Parse(files)
	default:
		fatal("unsupported language: %s (supported: laravel)", lang)
	}

	if err != nil {
		fatal("parsing failed: %v", err)
	}
	fmt.Printf("  Found %d endpoints\n", len(endpoints))

	// 2. Document with AI
	fmt.Println("→ Generating documentation with Claude...")
	client := ai.New(apiKey)
	var docs []models.EndpointDoc

	for i, ep := range endpoints {
		fmt.Printf("  [%d/%d] %s %s\n", i+1, len(endpoints), ep.Method, ep.URI)
		doc, err := client.DocumentEndpoint(ep)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  warning: failed to document %s %s: %v\n", ep.Method, ep.URI, err)
			continue
		}
		docs = append(docs, *doc)
	}

	// 3. Generate HTML
	fmt.Printf("→ Writing %s...\n", output)
	gen := generator.New()
	if err := gen.Generate(docs, title, output); err != nil {
		fatal("generating HTML: %v", err)
	}

	fmt.Printf("✓ Done! Open %s in your browser.\n", output)
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
