package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/divan1319/apidocgen/internal/ai"
	"github.com/divan1319/apidocgen/internal/cache"
	"github.com/divan1319/apidocgen/internal/generator"
	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/pkg/models"

	_ "github.com/divan1319/apidocgen/internal/parser/dotnet"
	_ "github.com/divan1319/apidocgen/internal/parser/laravel"
)

func main() {
	genCmd := flag.NewFlagSet("generate", flag.ExitOnError)

	lang       := genCmd.String("lang",    "laravel",              "Language/framework: "+strings.Join(parser.Names(), ", "))
	routes     := genCmd.String("routes",  "",                     "Comma-separated route files (e.g. routes/api.php)")
	root       := genCmd.String("root",    ".",                    "Project root directory")
	output     := genCmd.String("output",  "api-docs.html",       "Output HTML file")
	title      := genCmd.String("title",   "API Documentation",   "Documentation title")
	apiKey     := genCmd.String("api-key", "",                     "Anthropic API key (or set ANTHROPIC_API_KEY env var)")
	docLang    := genCmd.String("doc-lang","",                     "Documentation language: en (English) or es (Español). Prompted interactively if not set.")
	cacheFile  := genCmd.String("cache",   "apidocgen-cache.json", "Path to cache file. Use --cache=\"\" to disable.")
	forceRegen := genCmd.Bool("force",     false,                  "Ignore cache and re-document all endpoints with Claude.")
	workers    := genCmd.Int("workers",    5,                      "Concurrent requests to Claude API (default: 5)")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		genCmd.Parse(os.Args[2:])
		runGenerate(*lang, *routes, *root, *output, *title, *apiKey, *docLang, *cacheFile, *forceRegen, *workers)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runGenerate(lang, routes, root, output, title, apiKey, docLang, cacheFile string, forceRegen bool, workers int) {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		fatal("API key required: use --api-key or set ANTHROPIC_API_KEY")
	}
	if routes == "" {
		fatal("--routes is required (e.g. --routes routes/api.php)")
	}

	p, err := parser.Get(lang, root)
	if err != nil {
		fatal("%v", err)
	}

	files := splitTrim(routes, ",")

	// ── Step 1: resolve all included files ───────────────────────────────────
	fmt.Println("→ Resolving files...")
	allFiles, err := p.ResolveIncludes(files)
	if err != nil {
		fatal("resolving files: %v", err)
	}
	fmt.Printf("  Found %d file(s)\n", len(allFiles))

	reader := bufio.NewReader(os.Stdin)

	// ── Step 2: ask for documentation language if not set via flag ────────────
	if docLang == "" {
		fmt.Println("\n→ Documentation language / Idioma de la documentación:")
		fmt.Println("  [1] English")
		fmt.Println("  [2] Español")
		fmt.Print("  Select / Selecciona [1/2] (default: 1): ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		switch choice {
		case "2", "es", "español", "spanish":
			docLang = "es"
		default:
			docLang = "en"
		}
	}

	// ── Step 3: load cache ────────────────────────────────────────────────────
	var docCache *cache.Cache
	if cacheFile != "" {
		docCache, err = cache.Load(cacheFile)
		if err != nil {
			fatal("loading cache: %v", err)
		}
		if docCache.Len() > 0 {
			fmt.Printf("\n→ Cache loaded: %d endpoint(s) already documented (%s)\n", docCache.Len(), cacheFile)
		} else {
			fmt.Printf("\n→ Cache: starting fresh (%s)\n", cacheFile)
		}
		if forceRegen {
			fmt.Println("  --force: ignoring cache, all endpoints will be re-documented.")
		}
	}

	// ── Step 4: parse sections ───────────────────────────────────────────────
	fmt.Println("\n→ Parsing endpoints...")
	sections, err := p.ParseSections(allFiles)
	if err != nil {
		fatal("parsing endpoints: %v", err)
	}

	// ── Step 5: prompt user to name each section ─────────────────────────────
	fmt.Println("\n→ Name each section (press Enter to use the suggested name):")
	for i, s := range sections {
		suggested := suggestSectionName(s.FilePath)
		fmt.Printf("  %s [%s]: ", s.FilePath, suggested)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			input = suggested
		}
		sections[i].Name = input
	}

	totalEndpoints := 0
	for _, s := range sections {
		totalEndpoints += len(s.Endpoints)
	}
	fmt.Printf("  Found %d section(s), %d endpoint(s) total\n", len(sections), totalEndpoints)

	// ── Step 6: document with AI (worker pool + cache) ───────────────────────
	fmt.Printf("\n→ Generating documentation with Claude (workers: %d)...\n", workers)
	client, err := ai.New(apiKey, docLang)
	if err != nil {
		fatal("initializing AI client: %v", err)
	}

	var sectionDocs []models.SectionDoc
	totalHits, totalMisses := 0, 0

	for _, section := range sections {
		fmt.Printf("\n  [%s] — %d endpoints\n", section.Name, len(section.Endpoints))
		sd, hits, misses := documentSection(client, docCache, section, workers, forceRegen)
		sectionDocs = append(sectionDocs, sd)
		totalHits += hits
		totalMisses += misses

		if docCache != nil && misses > 0 {
			if err := docCache.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "  warning: could not save cache: %v\n", err)
			}
		}
	}

	if docCache != nil {
		fmt.Printf("\n  Cache summary: %d from cache, %d newly documented\n", totalHits, totalMisses)
	}

	// ── Step 7: generate HTML ────────────────────────────────────────────────
	fmt.Printf("\n→ Writing %s...\n", output)
	gen := generator.New()
	if err := gen.GenerateSections(sectionDocs, title, output); err != nil {
		fatal("generating HTML: %v", err)
	}

	fmt.Printf("✓ Done! Open %s in your browser.\n", output)
}

func documentSection(
	client *ai.Client,
	docCache *cache.Cache,
	section models.RouteSection,
	workers int,
	forceRegen bool,
) (sd models.SectionDoc, hits, misses int) {
	sd = models.SectionDoc{
		Name:     section.Name,
		Version:  section.Version,
		FilePath: section.FilePath,
	}

	type result struct {
		doc    *models.EndpointDoc
		err    error
		cached bool
	}

	results := make([]result, len(section.Endpoints))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var cacheMu sync.Mutex

	for i, ep := range section.Endpoints {
		if docCache != nil && !forceRegen {
			if cached, ok := docCache.Get(ep); ok {
				results[i] = result{doc: cached, cached: true}
				continue
			}
		}

		wg.Add(1)
		go func(i int, ep models.Endpoint) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			doc, err := client.DocumentEndpoint(ep)
			results[i] = result{doc: doc, err: err}

			if err == nil && doc != nil && docCache != nil {
				cacheMu.Lock()
				docCache.Set(ep, *doc)
				cacheMu.Unlock()
			}
		}(i, ep)
	}

	wg.Wait()

	for i, r := range results {
		ep := section.Endpoints[i]
		if r.cached {
			fmt.Printf("    %s %s  (cached)\n", ep.Method, ep.URI)
			sd.Docs = append(sd.Docs, *r.doc)
			hits++
			continue
		}
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "    warning: failed %s %s: %v\n", ep.Method, ep.URI, r.err)
			continue
		}
		if r.doc != nil {
			fmt.Printf("    %s %s  ✓\n", ep.Method, ep.URI)
			sd.Docs = append(sd.Docs, *r.doc)
			misses++
		}
	}

	fmt.Printf("    — %d documented, %d from cache, %d failed\n",
		misses, hits, len(section.Endpoints)-hits-misses)
	return
}

func suggestSectionName(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	for _, suffix := range []string{"Route", "Routes", "route", "routes"} {
		base = strings.TrimSuffix(base, suffix)
	}
	re := regexp.MustCompile(`([A-Z][a-z]+|[a-z]+|[A-Z]+(?:[A-Z][a-z]*)*)`)
	words := re.FindAllString(base, -1)
	for i, w := range words {
		words[i] = strings.Title(strings.ToLower(w))
	}
	name := strings.Join(words, " ")

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
	fmt.Printf(`apidocgen - AI-powered API documentation generator

USAGE:
  apidocgen generate [flags]

FLAGS:
  --lang      Language/framework: %s (default: laravel)
  --routes    Comma-separated route files or directories to parse
  --root      Project root directory (default: .)
  --output    Output HTML file (default: api-docs.html)
  --title     Documentation title (default: "API Documentation")
  --api-key   Anthropic API key (or use ANTHROPIC_API_KEY env var)
  --doc-lang  Documentation language: en (English) or es (Español)
  --cache     Path to cache file (default: apidocgen-cache.json)
              Use --cache="" to disable caching entirely.
  --force     Ignore cache and re-document all endpoints with Claude.
  --workers   Concurrent Claude API requests (default: 5)

EXAMPLES:
  # Laravel
  apidocgen generate --routes routes/api.php --root /path/to/laravel-project
  apidocgen generate --routes routes/api.php --doc-lang es --workers 10

  # .NET / ASP.NET Core
  apidocgen generate --lang dotnet --routes Controllers/ --root /path/to/dotnet-project
  apidocgen generate --lang dotnet --routes Program.cs,Controllers/ --doc-lang es
  ANTHROPIC_API_KEY=sk-... apidocgen generate --lang dotnet --routes Controllers/
`, strings.Join(parser.Names(), ", "))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
