package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/divan1319/apidocgen/internal/parser"
	"github.com/divan1319/apidocgen/internal/project"
)

type handlers struct {
	keys APIKeys
}

func (h *handlers) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := project.LoadAll(ProjectsDir)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []project.Project{}
	}

	type projectWithStatus struct {
		project.Project
		HasDocs bool `json:"has_docs"`
	}

	var result []projectWithStatus
	for _, p := range projects {
		result = append(result, projectWithStatus{
			Project: p,
			HasDocs: fileExists(p.OutputPath(DocsDir)),
		})
	}

	jsonOK(w, result)
}

func (h *handlers) getProject(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	path := filepath.Join(ProjectsDir, slug+".json")

	p, err := project.Load(path)
	if err != nil {
		jsonError(w, http.StatusNotFound, "project not found")
		return
	}

	jsonOK(w, p)
}

func (h *handlers) createProject(w http.ResponseWriter, r *http.Request) {
	var p project.Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if p.Name == "" {
		jsonError(w, http.StatusBadRequest, "name is required")
		return
	}

	if p.Slug == "" {
		p.Slug = project.SlugFromName(p.Name)
	}

	existing := filepath.Join(ProjectsDir, p.Slug+".json")
	if fileExists(existing) {
		jsonError(w, http.StatusConflict, fmt.Sprintf("project '%s' already exists", p.Slug))
		return
	}

	if err := project.Save(ProjectsDir, p); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, p)
}

func (h *handlers) updateProject(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	path := filepath.Join(ProjectsDir, slug+".json")

	if !fileExists(path) {
		jsonError(w, http.StatusNotFound, "project not found")
		return
	}

	var p project.Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	p.Slug = slug
	if err := project.Save(ProjectsDir, p); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonOK(w, p)
}

func (h *handlers) deleteProject(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	projectPath := filepath.Join(ProjectsDir, slug+".json")
	if !fileExists(projectPath) {
		jsonError(w, http.StatusNotFound, "project not found")
		return
	}

	os.Remove(projectPath)
	os.Remove(filepath.Join(CacheDir, slug+"-cache.json"))
	os.Remove(filepath.Join(DocsDir, slug+".html"))

	jsonOK(w, map[string]string{"status": "deleted"})
}

func (h *handlers) generateDocs(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	path := filepath.Join(ProjectsDir, slug+".json")

	p, err := project.Load(path)
	if err != nil {
		jsonError(w, http.StatusNotFound, "project not found")
		return
	}

	var opts struct {
		ForceRegen bool `json:"force_regen"`
		Workers    int  `json:"workers"`
	}
	json.NewDecoder(r.Body).Decode(&opts)
	if opts.Workers <= 0 {
		opts.Workers = 5
	}

	apiKey, err := apiKeyForProject(h.keys, *p)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	var logBuf bytes.Buffer
	result, err := RunGenerate(GenerateRequest{
		Project:    *p,
		APIKey:     apiKey,
		ForceRegen: opts.ForceRegen,
		Workers:    opts.Workers,
		DocLang:    p.DocLang,
	}, &logBuf)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("generation failed: %v\n\nLog:\n%s", err, logBuf.String()))
		return
	}

	jsonOK(w, map[string]any{
		"result": result,
		"log":    logBuf.String(),
	})
}

func (h *handlers) checkDocs(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	htmlPath := filepath.Join(DocsDir, slug+".html")

	exists := fileExists(htmlPath)
	jsonOK(w, map[string]any{
		"slug":   slug,
		"exists": exists,
		"url":    "/docs/" + slug + ".html",
	})
}

func (h *handlers) getSettings(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]any{
		"parsers":   parser.Names(),
		"doc_langs": []string{"en", "es"},
		"ai_providers": []map[string]string{
			{"id": "anthropic", "label": "Anthropic (Claude)"},
			{"id": "openai", "label": "OpenAI"},
			{"id": "deepseek", "label": "DeepSeek"},
		},
	})
}

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
