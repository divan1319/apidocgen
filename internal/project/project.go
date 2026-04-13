package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Project struct {
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Lang    string `json:"lang"`
	Routes  string `json:"routes"`
	Root    string `json:"root"`
	Title   string `json:"title"`
	DocLang string `json:"doc_lang"`
	// Proveedor de IA: anthropic (default), openai, deepseek
	AIProvider string `json:"ai_provider,omitempty"`
	AIModel    string `json:"ai_model,omitempty"`
	AIBaseURL  string `json:"ai_base_url,omitempty"`
}

func (p Project) CachePath(cacheDir string) string {
	return filepath.Join(cacheDir, p.Slug+"-cache.json")
}

func (p Project) OutputPath(docsDir string) string {
	return filepath.Join(docsDir, p.Slug+".html")
}

func Load(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading project file %s: %w", path, err)
	}
	var p Project
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing project file %s: %w", path, err)
	}
	return &p, nil
}

func LoadAll(dir string) ([]Project, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading projects directory: %w", err)
	}

	var projects []Project
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		p, err := Load(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		projects = append(projects, *p)
	}
	return projects, nil
}

func Save(dir string, p Project) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating projects directory: %w", err)
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling project: %w", err)
	}

	path := filepath.Join(dir, p.Slug+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing project file: %w", err)
	}
	return nil
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func SlugFromName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
