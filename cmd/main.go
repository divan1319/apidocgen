package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/divan1319/apidocgen/internal/ai"
	"github.com/divan1319/apidocgen/internal/cache"
	"github.com/divan1319/apidocgen/internal/generator"
	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/internal/project"
	"github.com/divan1319/apidocgen/pkg/models"

	_ "github.com/divan1319/apidocgen/internal/parser/dotnet"
	_ "github.com/divan1319/apidocgen/internal/parser/laravel"
)

const (
	projectsDir = "projects"
	cacheDir    = "cache"
	docsDir     = "docs"
)

func main() {
	genCmd := flag.NewFlagSet("generate", flag.ExitOnError)

	lang       := genCmd.String("lang", "laravel", "Language/framework: "+strings.Join(parser.Names(), ", "))
	routes     := genCmd.String("routes", "", "Comma-separated route files (e.g. routes/api.php)")
	root       := genCmd.String("root", ".", "Project root directory")
	output     := genCmd.String("output", "", "Output HTML file (default: docs/<slug>.html)")
	title      := genCmd.String("title", "API Documentation", "Documentation title")
	apiKey     := genCmd.String("api-key", "", "Anthropic API key (or set ANTHROPIC_API_KEY env var)")
	docLang    := genCmd.String("doc-lang", "", "Documentation language: en (English) or es (Español). Prompted interactively if not set.")
	cacheFile  := genCmd.String("cache", "", "Path to cache file (default: cache/<slug>-cache.json). Use --cache=none to disable.")
	forceRegen := genCmd.Bool("force", false, "Ignore cache and re-document all endpoints with Claude.")
	workers    := genCmd.Int("workers", 5, "Concurrent requests to Claude API (default: 5)")
	projectSlug := genCmd.String("project", "", "Project slug to select directly (skip interactive menu)")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		genCmd.Parse(os.Args[2:])
		runGenerate(*lang, *routes, *root, *output, *title, *apiKey, *docLang, *cacheFile, *forceRegen, *workers, *projectSlug)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func ensureDirs() {
	for _, d := range []string{projectsDir, cacheDir, docsDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			fatal("creating directory %s: %v", d, err)
		}
	}
}

func runGenerate(lang, routes, root, output, title, apiKey, docLang, cacheFile string, forceRegen bool, workers int, projectSlug string) {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		fatal("API key required: use --api-key or set ANTHROPIC_API_KEY")
	}

	reader := bufio.NewReader(os.Stdin)

	ensureDirs()

	// ── Resolve project ─────────────────────────────────────────────────────
	var proj *project.Project

	hasExplicitFlags := routes != ""

	if hasExplicitFlags && projectSlug == "" {
		// Legacy / non-interactive mode: flags provided directly without --project
		proj = &project.Project{
			Lang:    lang,
			Routes:  routes,
			Root:    root,
			Title:   title,
			DocLang: docLang,
		}
	} else {
		proj = selectOrCreateProject(reader, projectSlug, lang, routes, root, title, docLang)
	}

	// Apply project values
	lang = proj.Lang
	routes = proj.Routes
	root = proj.Root
	title = proj.Title
	if proj.DocLang != "" {
		docLang = proj.DocLang
	}

	// Resolve output and cache paths from slug if available
	if output == "" && proj.Slug != "" {
		output = proj.OutputPath(docsDir)
	} else if output == "" {
		output = "api-docs.html"
	}

	if cacheFile == "" && proj.Slug != "" {
		cacheFile = proj.CachePath(cacheDir)
	} else if cacheFile == "" {
		cacheFile = "apidocgen-cache.json"
	}
	if cacheFile == "none" {
		cacheFile = ""
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
	fmt.Println("→ Resolviendo archivos...")
	allFiles, err := p.ResolveIncludes(files)
	if err != nil {
		fatal("resolving files: %v", err)
	}
	fmt.Printf("  Encontrados %d archivo(s)\n", len(allFiles))

	// ── Step 2: ask for documentation language if not set via flag ────────────
	if docLang == "" {
		fmt.Println("\n→ Idioma de la documentación:")
		fmt.Println("  [1] English")
		fmt.Println("  [2] Español")
		fmt.Print("  Selecciona [1/2] (default: 1): ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		switch choice {
		case "2", "es", "español", "spanish":
			docLang = "es"
		default:
			docLang = "en"
		}
		if proj.Slug != "" {
			proj.DocLang = docLang
			_ = project.Save(projectsDir, *proj)
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
			fmt.Printf("\n→ Cache cargado: %d endpoint(s) ya documentados (%s)\n", docCache.Len(), cacheFile)
		} else {
			fmt.Printf("\n→ Cache: iniciando desde cero (%s)\n", cacheFile)
		}
		if forceRegen {
			fmt.Println("  --force: ignorando cache, todos los endpoints serán re-documentados.")
		}
	}

	// ── Step 4: parse sections ───────────────────────────────────────────────
	fmt.Println("\n→ Parseando endpoints...")
	sections, err := p.ParseSections(allFiles)
	if err != nil {
		fatal("parsing endpoints: %v", err)
	}

	// ── Step 5: prompt user to name each section ─────────────────────────────
	fmt.Println("\n→ Nombra cada sección (presiona Enter para usar el nombre sugerido):")
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
	fmt.Printf("  Encontradas %d sección(es), %d endpoint(s) en total\n", len(sections), totalEndpoints)

	// ── Step 6: document with AI (worker pool + cache) ───────────────────────
	fmt.Printf("\n→ Generando documentación con Claude (workers: %d)...\n", workers)
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
				fmt.Fprintf(os.Stderr, "  warning: no se pudo guardar el cache: %v\n", err)
			}
		}
	}

	if docCache != nil {
		fmt.Printf("\n  Resumen cache: %d desde cache, %d nuevos documentados\n", totalHits, totalMisses)
	}

	// ── Step 7: generate HTML ────────────────────────────────────────────────
	fmt.Printf("\n→ Escribiendo %s...\n", output)
	gen := generator.New()
	if err := gen.GenerateSections(sectionDocs, title, output); err != nil {
		fatal("generating HTML: %v", err)
	}

	// ── Step 8: regenerate index.html ────────────────────────────────────────
	if proj.Slug != "" {
		regenerateIndex(gen)
	}

	fmt.Printf("✓ ¡Listo! Abre %s en tu navegador.\n", output)
}

func selectOrCreateProject(reader *bufio.Reader, projectSlug, defaultLang, defaultRoutes, defaultRoot, defaultTitle, defaultDocLang string) *project.Project {
	projects, err := project.LoadAll(projectsDir)
	if err != nil {
		fatal("loading projects: %v", err)
	}

	// Direct selection via --project flag
	if projectSlug != "" {
		for _, p := range projects {
			if p.Slug == projectSlug {
				fmt.Printf("→ Proyecto seleccionado: %s (%s)\n", p.Name, p.Slug)
				return &p
			}
		}
		fatal("proyecto no encontrado: %s", projectSlug)
	}

	// Interactive menu
	fmt.Println("\n→ Selecciona un proyecto de documentación:")
	if len(projects) > 0 {
		for i, p := range projects {
			fmt.Printf("  [%d] %s (%s) — %s\n", i+1, p.Name, p.Slug, p.Lang)
		}
		fmt.Printf("  [%d] Crear nuevo proyecto\n", len(projects)+1)
		fmt.Printf("  Selecciona [1-%d]: ", len(projects)+1)
	} else {
		fmt.Println("  No hay proyectos configurados. Vamos a crear uno nuevo.")
		fmt.Println()
	}

	if len(projects) > 0 {
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		idx, err := strconv.Atoi(choice)
		if err == nil && idx >= 1 && idx <= len(projects) {
			p := projects[idx-1]
			fmt.Printf("  → Seleccionado: %s\n", p.Name)
			return &p
		}
	}

	// Create new project
	return createNewProject(reader, defaultLang, defaultRoutes, defaultRoot, defaultTitle, defaultDocLang)
}

func createNewProject(reader *bufio.Reader, defaultLang, defaultRoutes, defaultRoot, defaultTitle, defaultDocLang string) *project.Project {
	fmt.Println("\n→ Crear nuevo proyecto de documentación:")

	name := prompt(reader, "  Nombre del proyecto", "")
	if name == "" {
		fatal("el nombre del proyecto es obligatorio")
	}

	slug := project.SlugFromName(name)
	fmt.Printf("  Slug generado: %s\n", slug)

	langInput := prompt(reader, fmt.Sprintf("  Framework/lenguaje (%s)", strings.Join(parser.Names(), ", ")), defaultLang)
	if langInput == "" {
		langInput = defaultLang
	}

	routesInput := prompt(reader, "  Archivos de rutas (separados por coma)", defaultRoutes)
	if routesInput == "" {
		routesInput = defaultRoutes
	}

	rootInput := prompt(reader, "  Directorio raíz del proyecto", defaultRoot)
	if rootInput == "" {
		rootInput = defaultRoot
	}

	titleInput := prompt(reader, "  Título de la documentación", defaultTitle)
	if titleInput == "" {
		titleInput = defaultTitle
	}

	docLangInput := prompt(reader, "  Idioma de documentación (en/es)", defaultDocLang)
	if docLangInput == "" {
		docLangInput = defaultDocLang
	}

	proj := project.Project{
		Name:    name,
		Slug:    slug,
		Lang:    langInput,
		Routes:  routesInput,
		Root:    rootInput,
		Title:   titleInput,
		DocLang: docLangInput,
	}

	if err := project.Save(projectsDir, proj); err != nil {
		fatal("guardando proyecto: %v", err)
	}
	fmt.Printf("  ✓ Proyecto guardado en %s/%s.json\n\n", projectsDir, slug)

	return &proj
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func regenerateIndex(gen *generator.HTMLGenerator) {
	projects, err := project.LoadAll(projectsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  warning: no se pudo regenerar index.html: %v\n", err)
		return
	}

	var entries []generator.IndexEntry
	for _, p := range projects {
		htmlFile := p.Slug + ".html"
		if _, err := os.Stat(filepath.Join(docsDir, htmlFile)); err != nil {
			continue
		}
		entries = append(entries, generator.IndexEntry{
			Name:     p.Name,
			Slug:     p.Slug,
			Title:    p.Title,
			Lang:     p.Lang,
			HtmlFile: htmlFile,
		})
	}

	indexPath := filepath.Join(docsDir, "index.html")
	if err := gen.GenerateIndex(entries, indexPath); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: no se pudo generar index.html: %v\n", err)
		return
	}
	fmt.Printf("→ Index actualizado: %s (%d proyecto(s))\n", indexPath, len(entries))
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

	fmt.Printf("    — %d documentados, %d desde cache, %d fallidos\n",
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
	fmt.Printf(`apidocgen - Generador de documentación de APIs con IA

USO:
  apidocgen generate [flags]

FLAGS:
  --project   Slug del proyecto a generar (salta el menú interactivo)
  --lang      Framework/lenguaje: %s (default: laravel)
  --routes    Archivos de rutas separados por coma
  --root      Directorio raíz del proyecto (default: .)
  --output    Archivo HTML de salida (default: docs/<slug>.html)
  --title     Título de la documentación (default: "API Documentation")
  --api-key   Anthropic API key (o usar ANTHROPIC_API_KEY env var)
  --doc-lang  Idioma de documentación: en (English) o es (Español)
  --cache     Ruta al archivo de cache (default: cache/<slug>-cache.json)
              Usar --cache=none para deshabilitar cache.
  --force     Ignorar cache y re-documentar todos los endpoints con Claude.
  --workers   Peticiones concurrentes a Claude API (default: 5)

ESTRUCTURA:
  projects/   Configuración de cada proyecto (.json)
  cache/      Archivos de cache por proyecto
  docs/       Documentación HTML generada + index.html

EJEMPLOS:
  # Menú interactivo (seleccionar o crear proyecto)
  apidocgen generate

  # Seleccionar proyecto directamente
  apidocgen generate --project mi-api-laravel

  # Modo legacy (sin proyecto guardado)
  apidocgen generate --lang laravel --routes routes/api.php --root /path/to/project

  # .NET
  apidocgen generate --lang dotnet --routes Controllers/ --root /path/to/dotnet-project
`, strings.Join(parser.Names(), ", "))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
