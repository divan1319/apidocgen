package server

import (
	"fmt"
	"strings"

	"github.com/divan1319/apidocgen/internal/ai"
	"github.com/divan1319/apidocgen/internal/project"
)

// APIKeys agrupa claves por proveedor (variables de entorno o flags de serve).
type APIKeys struct {
	Anthropic string
	OpenAI    string
	DeepSeek  string
}

func apiKeyForProject(keys APIKeys, proj project.Project) (string, error) {
	prov, err := ai.ParseProvider(proj.AIProvider)
	if err != nil {
		return "", err
	}
	switch prov {
	case ai.ProviderAnthropic:
		if strings.TrimSpace(keys.Anthropic) != "" {
			return strings.TrimSpace(keys.Anthropic), nil
		}
		return "", fmt.Errorf("falta ANTHROPIC_API_KEY o usa --api-key al iniciar serve")
	case ai.ProviderOpenAI:
		if strings.TrimSpace(keys.OpenAI) != "" {
			return strings.TrimSpace(keys.OpenAI), nil
		}
		return "", fmt.Errorf("falta OPENAI_API_KEY en el entorno del servidor")
	case ai.ProviderDeepSeek:
		if strings.TrimSpace(keys.DeepSeek) != "" {
			return strings.TrimSpace(keys.DeepSeek), nil
		}
		return "", fmt.Errorf("falta DEEPSEEK_API_KEY en el entorno del servidor")
	default:
		return "", fmt.Errorf("proveedor no soportado")
	}
}
