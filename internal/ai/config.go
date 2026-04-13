package ai

import (
	"fmt"
	"strings"
)

// Provider identifica el backend de chat usado para documentar.
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
	ProviderDeepSeek  Provider = "deepseek"
)

// Config agrupa credenciales y opciones del proveedor de IA.
type Config struct {
	Provider Provider
	APIKey   string
	Model    string
	BaseURL  string
	DocLang  string
}

// ParseProvider normaliza el nombre del proveedor desde flags o JSON.
func ParseProvider(s string) (Provider, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ProviderAnthropic, nil
	}
	switch s {
	case "anthropic", "claude":
		return ProviderAnthropic, nil
	case "openai":
		return ProviderOpenAI, nil
	case "deepseek":
		return ProviderDeepSeek, nil
	default:
		return "", fmt.Errorf("proveedor IA desconocido: %q (use anthropic, openai o deepseek)", s)
	}
}

// DefaultModel por proveedor cuando el usuario no indica uno.
func DefaultModel(p Provider) string {
	switch p {
	case ProviderAnthropic:
		return "claude-sonnet-4-6"
	case ProviderOpenAI:
		return "gpt-4o-mini"
	case ProviderDeepSeek:
		return "deepseek-chat"
	default:
		return ""
	}
}

// DefaultBaseURL para APIs estilo OpenAI (chat completions).
func DefaultBaseURL(p Provider) string {
	switch p {
	case ProviderOpenAI:
		return "https://api.openai.com/v1"
	case ProviderDeepSeek:
		return "https://api.deepseek.com"
	default:
		return ""
	}
}

// ResolvedModel devuelve el modelo efectivo a enviar a la API.
func (c Config) ResolvedModel() string {
	if strings.TrimSpace(c.Model) != "" {
		return strings.TrimSpace(c.Model)
	}
	return DefaultModel(c.Provider)
}

// ResolvedBaseURL devuelve la base URL para chat completions (sin path final).
func (c Config) ResolvedBaseURL() string {
	if strings.TrimSpace(c.BaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	}
	return DefaultBaseURL(c.Provider)
}

// CacheScope distingue entradas de caché al cambiar proveedor o modelo.
func (c Config) CacheScope() string {
	return string(c.Provider) + ":" + c.ResolvedModel()
}
