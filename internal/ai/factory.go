package ai

import (
	"fmt"
	"strings"
)

// New construye un Documenter según Config.Provider.
func New(cfg Config) (Documenter, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("API key vacía")
	}
	switch cfg.Provider {
	case ProviderAnthropic:
		return newAnthropicClient(cfg)
	case ProviderOpenAI, ProviderDeepSeek:
		return newOpenAICompatClient(cfg)
	default:
		return nil, fmt.Errorf("proveedor no soportado: %s", cfg.Provider)
	}
}
