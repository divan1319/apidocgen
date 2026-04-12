package server

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/divan1319/apidocgen/internal/ai"
	"github.com/divan1319/apidocgen/internal/cache"
	"github.com/divan1319/apidocgen/internal/generator"
	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/internal/project"
	"github.com/divan1319/apidocgen/pkg/models"
)

const (
	ProjectsDir = "projects"
	CacheDir    = "cache"
	DocsDir     = "docs"
)

type GenerateRequest struct {
	Project    project.Project
	APIKey     string
	ForceRegen bool
	Workers    int
	DocLang    string
	Output     string
	CacheFile  string
}

type GenerateResult struct {
	TotalEndpoints  int    `json:"total_endpoints"`
	FromCache       int    `json:"from_cache"`
	NewlyDocumented int    `json:"newly_documented"`
	Failed          int    `json:"failed"`
	OutputPath      string `json:"output_path"`
}

func EnsureDirs() error {
	for _, d := range []string{ProjectsDir, CacheDir, DocsDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}
	return nil
}

func RunGenerate(req GenerateRequest, log io.Writer) (*GenerateResult, error) {
	if log == nil {
		log = os.Stdout
	}

	proj := req.Project
	docLang := req.DocLang
	if docLang == "" {
		docLang = proj.DocLang
	}
	if docLang == "" {
		docLang = "en"
	}

	output := req.Output
	if output == "" && proj.Slug != "" {
		output = proj.OutputPath(DocsDir)
	} else if output == "" {
		output = "api-docs.html"
	}

	cacheFile := req.CacheFile
	if cacheFile == "" && proj.Slug != "" {
		cacheFile = proj.CachePath(CacheDir)
	} else if cacheFile == "" {
		cacheFile = "apidocgen-cache.json"
	}
	if cacheFile == "none" {
		cacheFile = ""
	}

	if proj.Routes == "" {
		return nil, fmt.Errorf("routes is required")
	}

	p, err := parser.Get(proj.Lang, proj.Root)
	if err != nil {
		return nil, err
	}

	files := splitTrim(proj.Routes, ",")

	fmt.Fprintf(log, "→ Resolviendo archivos...\n")
	allFiles, err := p.ResolveIncludes(files)
	if err != nil {
		return nil, fmt.Errorf("resolving files: %w", err)
	}
	fmt.Fprintf(log, "  Encontrados %d archivo(s)\n", len(allFiles))

	var docCache *cache.Cache
	if cacheFile != "" {
		docCache, err = cache.Load(cacheFile)
		if err != nil {
			return nil, fmt.Errorf("loading cache: %w", err)
		}
		if docCache.Len() > 0 {
			fmt.Fprintf(log, "→ Cache cargado: %d endpoint(s) ya documentados\n", docCache.Len())
		}
	}

	fmt.Fprintf(log, "→ Parseando endpoints...\n")
	sections, err := p.ParseSections(allFiles)
	if err != nil {
		return nil, fmt.Errorf("parsing endpoints: %w", err)
	}

	for i, s := range sections {
		if s.Name == "" {
			sections[i].Name = suggestSectionName(s.FilePath)
		}
	}

	totalEndpoints := 0
	for _, s := range sections {
		totalEndpoints += len(s.Endpoints)
	}
	fmt.Fprintf(log, "  %d sección(es), %d endpoint(s) en total\n", len(sections), totalEndpoints)

	workers := req.Workers
	if workers <= 0 {
		workers = 5
	}

	fmt.Fprintf(log, "→ Generando documentación con Claude (workers: %d)...\n", workers)
	client, err := ai.New(req.APIKey, docLang)
	if err != nil {
		return nil, fmt.Errorf("initializing AI client: %w", err)
	}

	var sectionDocs []models.SectionDoc
	totalHits, totalMisses, totalFailed := 0, 0, 0

	for _, section := range sections {
		fmt.Fprintf(log, "  [%s] — %d endpoints\n", section.Name, len(section.Endpoints))
		sd, hits, misses, failed := documentSection(client, docCache, section, workers, req.ForceRegen, log)
		sectionDocs = append(sectionDocs, sd)
		totalHits += hits
		totalMisses += misses
		totalFailed += failed

		if docCache != nil && misses > 0 {
			if err := docCache.Save(); err != nil {
				fmt.Fprintf(log, "  warning: no se pudo guardar el cache: %v\n", err)
			}
		}
	}

	fmt.Fprintf(log, "→ Escribiendo %s...\n", output)
	gen := generator.New()
	if err := gen.GenerateSections(sectionDocs, proj.Title, output); err != nil {
		return nil, fmt.Errorf("generating HTML: %w", err)
	}

	if proj.Slug != "" {
		RegenerateIndex(gen, log)
	}

	return &GenerateResult{
		TotalEndpoints:  totalEndpoints,
		FromCache:       totalHits,
		NewlyDocumented: totalMisses,
		Failed:          totalFailed,
		OutputPath:      output,
	}, nil
}

func RegenerateIndex(gen *generator.HTMLGenerator, log io.Writer) {
	projects, err := project.LoadAll(ProjectsDir)
	if err != nil {
		fmt.Fprintf(log, "  warning: no se pudo regenerar index.html: %v\n", err)
		return
	}

	var entries []generator.IndexEntry
	for _, p := range projects {
		htmlFile := p.Slug + ".html"
		if _, err := os.Stat(filepath.Join(DocsDir, htmlFile)); err != nil {
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

	indexPath := filepath.Join(DocsDir, "index.html")
	if err := gen.GenerateIndex(entries, indexPath); err != nil {
		fmt.Fprintf(log, "  warning: no se pudo generar index.html: %v\n", err)
		return
	}
	fmt.Fprintf(log, "→ Index actualizado: %s (%d proyecto(s))\n", indexPath, len(entries))
}

func documentSection(
	client *ai.Client,
	docCache *cache.Cache,
	section models.RouteSection,
	workers int,
	forceRegen bool,
	log io.Writer,
) (sd models.SectionDoc, hits, misses, failed int) {
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
			fmt.Fprintf(log, "    %s %s  (cached)\n", ep.Method, ep.URI)
			sd.Docs = append(sd.Docs, *r.doc)
			hits++
			continue
		}
		if r.err != nil {
			fmt.Fprintf(log, "    warning: failed %s %s: %v\n", ep.Method, ep.URI, r.err)
			failed++
			continue
		}
		if r.doc != nil {
			fmt.Fprintf(log, "    %s %s  ✓\n", ep.Method, ep.URI)
			sd.Docs = append(sd.Docs, *r.doc)
			misses++
		}
	}

	fmt.Fprintf(log, "    — %d documentados, %d desde cache, %d fallidos\n",
		misses, hits, failed)
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
