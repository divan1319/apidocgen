package ai

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/divan1319/apidocgen/pkg/models"
)

//go:embed prompts.json
var promptsData []byte

type prompts struct {
	EN string `json:"en"`
	ES string `json:"es"`
}

func loadPrompt(lang string) (string, error) {
	var p prompts
	if err := json.Unmarshal(promptsData, &p); err != nil {
		return "", fmt.Errorf("loading prompts.json: %w", err)
	}
	switch strings.ToLower(lang) {
	case "es", "español", "spanish":
		return p.ES, nil
	default:
		return p.EN, nil
	}
}

func buildPrompt(ep models.Endpoint) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Method: %s\nURI: %s\n", ep.Method, ep.URI))

	if ep.Controller != "" {
		sb.WriteString(fmt.Sprintf("Controller: %s@%s\n", ep.Controller, ep.Action))
	}
	if len(ep.Middleware) > 0 {
		sb.WriteString(fmt.Sprintf("Middleware: %s\n", strings.Join(ep.Middleware, ", ")))
	}

	if len(ep.StaticMeta.RequestParams) > 0 {
		sb.WriteString("\nStatically extracted parameters:\n")
		for _, p := range ep.StaticMeta.RequestParams {
			sb.WriteString(fmt.Sprintf("  - %s (%s, required=%v): %s\n", p.Name, p.Type, p.Required, p.Rules))
		}
	}

	if ep.RawSource != "" {
		lang := ep.Language
		if lang == "" {
			lang = "php"
		}
		sb.WriteString(fmt.Sprintf("\nController source code:\n```%s\n", lang))
		sb.WriteString(ep.RawSource)
		sb.WriteString("\n```\n")
	}

	sb.WriteString("\nGenerate complete API documentation for this endpoint.")
	return sb.String()
}
