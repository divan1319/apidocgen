package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	apidocgen "github.com/divan1319/apidocgen"
	"github.com/divan1319/apidocgen/internal/ai"
	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/internal/project"
	"github.com/divan1319/apidocgen/internal/server"

	_ "github.com/divan1319/apidocgen/internal/parser/dotnet"
	_ "github.com/divan1319/apidocgen/internal/parser/golang"
	_ "github.com/divan1319/apidocgen/internal/parser/laravel"
	_ "github.com/divan1319/apidocgen/internal/parser/node"
	_ "github.com/divan1319/apidocgen/internal/parser/python"
)

func main() {
	genCmd := flag.NewFlagSet("generate", flag.ExitOnError)

	lang := genCmd.String("lang", "laravel", "Language/framework: "+strings.Join(parser.Names(), ", "))
	routes := genCmd.String("routes", "", "Comma-separated route files (e.g. routes/api.php)")
	root := genCmd.String("root", ".", "Project root directory")
	output := genCmd.String("output", "", "Output HTML file (default: docs/<slug>.html)")
	title := genCmd.String("title", "API Documentation", "Documentation title")
	apiKey := genCmd.String("api-key", "", "API key del proveedor activo (o ANTHROPIC_API_KEY / OPENAI_API_KEY / DEEPSEEK_API_KEY)")
	aiProvider := genCmd.String("ai-provider", "", "Proveedor IA: anthropic, openai o deepseek (default: anthropic o el guardado en el proyecto)")
	aiModel := genCmd.String("ai-model", "", "Modelo de IA (opcional; por defecto uno razonable por proveedor)")
	aiBaseURL := genCmd.String("ai-base-url", "", "Base URL para API estilo OpenAI (opcional; p. ej. proxy compatible con OpenAI)")
	docLang := genCmd.String("doc-lang", "", "Documentation language: en (English) or es (Español). Prompted interactively if not set.")
	cacheFile := genCmd.String("cache", "", "Path to cache file (default: cache/<slug>-cache.json). Use --cache=none to disable.")
	forceRegen := genCmd.Bool("force", false, "Ignore cache and re-document all endpoints.")
	workers := genCmd.Int("workers", 5, "Peticiones concurrentes a la API de IA (default: 5)")
	projectSlug := genCmd.String("project", "", "Project slug to select directly (skip interactive menu)")

	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	servePort := serveCmd.Int("port", 8080, "HTTP server port")
	serveAPIKey := serveCmd.String("api-key", "", "Clave opcional para un solo proveedor (usar con --api-key-for; si no, usa solo variables de entorno)")
	serveAPIKeyFor := serveCmd.String("api-key-for", "anthropic", "Proveedor al que aplica --api-key: anthropic, openai o deepseek")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		genCmd.Parse(os.Args[2:])
		runGenerate(*lang, *routes, *root, *output, *title, *apiKey, *aiProvider, *aiModel, *aiBaseURL, *docLang, *cacheFile, *forceRegen, *workers, *projectSlug)
	case "serve":
		serveCmd.Parse(os.Args[2:])
		runServe(*servePort, *serveAPIKey, *serveAPIKeyFor)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runServe(port int, apiKeyFlag, apiKeyFor string) {
	keys := server.APIKeys{
		Anthropic: strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
		OpenAI:    strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		DeepSeek:  strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")),
	}
	if strings.TrimSpace(apiKeyFlag) != "" {
		prov, err := ai.ParseProvider(apiKeyFor)
		if err != nil {
			fatal("%v", err)
		}
		switch prov {
		case ai.ProviderAnthropic:
			keys.Anthropic = strings.TrimSpace(apiKeyFlag)
		case ai.ProviderOpenAI:
			keys.OpenAI = strings.TrimSpace(apiKeyFlag)
		case ai.ProviderDeepSeek:
			keys.DeepSeek = strings.TrimSpace(apiKeyFlag)
		}
	}
	if keys.Anthropic == "" && keys.OpenAI == "" && keys.DeepSeek == "" {
		fatal("se requiere al menos una clave en el entorno o con --api-key: ANTHROPIC_API_KEY, OPENAI_API_KEY y/o DEEPSEEK_API_KEY (opcional --api-key-for si usas --api-key)")
	}
	if err := server.EnsureDirs(); err != nil {
		fatal("%v", err)
	}
	server.SetWebAssets(apidocgen.WebAssets)
	server.Run(port, keys)
}

func runGenerate(lang, routes, root, output, title, apiKey, aiProviderFlag, aiModelFlag, aiBaseURLFlag, docLang, cacheFile string, forceRegen bool, workers int, projectSlug string) {

	reader := bufio.NewReader(os.Stdin)

	if err := server.EnsureDirs(); err != nil {
		fatal("%v", err)
	}

	var proj *project.Project

	hasExplicitFlags := routes != ""

	if hasExplicitFlags && projectSlug == "" {
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

	if proj.DocLang == "" && docLang != "" {
		proj.DocLang = docLang
	}

	provStr := aiProviderFlag
	if provStr == "" {
		provStr = proj.AIProvider
	}
	prov, err := ai.ParseProvider(provStr)
	if err != nil {
		fatal("%v", err)
	}
	if apiKey == "" {
		apiKey = apiKeyForProviderCLI(prov)
	}
	if apiKey == "" {
		switch prov {
		case ai.ProviderOpenAI:
			fatal("clave OpenAI requerida: --api-key o OPENAI_API_KEY")
		case ai.ProviderDeepSeek:
			fatal("clave DeepSeek requerida: --api-key o DEEPSEEK_API_KEY")
		default:
			fatal("clave Anthropic requerida: --api-key o ANTHROPIC_API_KEY")
		}
	}

	if proj.DocLang == "" {
		fmt.Println("\n→ Idioma de la documentación:")
		fmt.Println("  [1] English")
		fmt.Println("  [2] Español")
		fmt.Print("  Selecciona [1/2] (default: 1): ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		switch choice {
		case "2", "es", "español", "spanish":
			proj.DocLang = "es"
		default:
			proj.DocLang = "en"
		}
		if proj.Slug != "" {
			_ = project.Save(server.ProjectsDir, *proj)
		}
	}

	result, err := server.RunGenerate(server.GenerateRequest{
		Project:    *proj,
		APIKey:     apiKey,
		AIProvider: aiProviderFlag,
		AIModel:    aiModelFlag,
		AIBaseURL:  aiBaseURLFlag,
		ForceRegen: forceRegen,
		Workers:    workers,
		DocLang:    proj.DocLang,
		Output:     output,
		CacheFile:  cacheFile,
	}, os.Stdout)
	if err != nil {
		fatal("%v", err)
	}

	fmt.Printf("✓ ¡Listo! %d endpoints documentados, %d desde cache, %d fallidos.\n",
		result.NewlyDocumented, result.FromCache, result.Failed)
	fmt.Printf("  Abre %s en tu navegador.\n", result.OutputPath)
}

func selectOrCreateProject(reader *bufio.Reader, projectSlug, defaultLang, defaultRoutes, defaultRoot, defaultTitle, defaultDocLang string) *project.Project {
	projects, err := project.LoadAll(server.ProjectsDir)
	if err != nil {
		fatal("loading projects: %v", err)
	}

	if projectSlug != "" {
		for _, p := range projects {
			if p.Slug == projectSlug {
				fmt.Printf("→ Proyecto seleccionado: %s (%s)\n", p.Name, p.Slug)
				return &p
			}
		}
		fatal("proyecto no encontrado: %s", projectSlug)
	}

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

	aiProvInput := prompt(reader, "  Proveedor IA (anthropic / openai / deepseek)", "anthropic")
	if _, err := ai.ParseProvider(aiProvInput); err != nil {
		fatal("%v", err)
	}
	aiModelInput := prompt(reader, "  Modelo IA (opcional, vacío = default)", "")
	aiBaseInput := prompt(reader, "  Base URL API OpenAI-compatible (opcional)", "")

	proj := project.Project{
		Name:       name,
		Slug:       slug,
		Lang:       langInput,
		Routes:     routesInput,
		Root:       rootInput,
		Title:      titleInput,
		DocLang:    docLangInput,
		AIProvider: strings.ToLower(strings.TrimSpace(aiProvInput)),
		AIModel:    strings.TrimSpace(aiModelInput),
		AIBaseURL:  strings.TrimSpace(aiBaseInput),
	}

	if err := project.Save(server.ProjectsDir, proj); err != nil {
		fatal("guardando proyecto: %v", err)
	}
	fmt.Printf("  ✓ Proyecto guardado en %s/%s.json\n\n", server.ProjectsDir, slug)

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

func printUsage() {
	fmt.Printf(`apidocgen - Generador de documentación de APIs con IA

USO:
  apidocgen generate [flags]    Generar documentación
  apidocgen serve [flags]       Iniciar servidor web
  apidocgen help                Mostrar esta ayuda

FLAGS DE generate:
  --project   Slug del proyecto a generar (salta el menú interactivo)
  --lang      Framework/lenguaje: %s (default: laravel)
  --routes    Archivos de rutas separados por coma
  --root      Directorio raíz del proyecto (default: .)
  --output    Archivo HTML de salida (default: docs/<slug>.html)
  --title     Título de la documentación (default: "API Documentation")
  --api-key       API key del proveedor activo (o ANTHROPIC_API_KEY / OPENAI_API_KEY / DEEPSEEK_API_KEY)
  --ai-provider   anthropic, openai o deepseek (sustituye al del proyecto si se indica)
  --ai-model      Modelo concreto (opcional)
  --ai-base-url   Base URL para API compatible con OpenAI (opcional)
  --doc-lang      Idioma de documentación: en (English) o es (Español)
  --cache         Ruta al archivo de cache (default: cache/<slug>-cache.json)
                  Usar --cache=none para deshabilitar cache.
  --force         Ignorar cache y re-documentar todos los endpoints.
  --workers       Peticiones concurrentes a la API de IA (default: 5)

FLAGS DE serve:
  --port          Puerto del servidor HTTP (default: 8080)
  --api-key       Opcional: una clave que no esté en el entorno
  --api-key-for   Proveedor de --api-key (default: anthropic). Al generar, cada proyecto
                  usa la clave según su ai_provider (no según este flag).
                  Claves por defecto: ANTHROPIC_API_KEY, OPENAI_API_KEY, DEEPSEEK_API_KEY

EJEMPLOS:
  apidocgen generate
  apidocgen generate --project mi-api-laravel
  apidocgen serve --port 3000
`, strings.Join(parser.Names(), ", "))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func apiKeyForProviderCLI(p ai.Provider) string {
	switch p {
	case ai.ProviderOpenAI:
		return os.Getenv("OPENAI_API_KEY")
	case ai.ProviderDeepSeek:
		return os.Getenv("DEEPSEEK_API_KEY")
	default:
		return os.Getenv("ANTHROPIC_API_KEY")
	}
}
