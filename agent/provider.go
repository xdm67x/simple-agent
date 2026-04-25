package agent

import (
	"context"
	"strings"
)

// Provider defines the interface for LLM providers.
type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (Message, error)
	ListModels(ctx context.Context) ([]string, error)
}

// NewProvider creates a provider based on the model prefix in cfg.Model.
// Supported prefixes:
//   - "openrouter/<model>"  -> OpenRouter provider
//   - "opencode/<model>"    -> OpenCode provider
//   - "ollama/<model>"      -> Ollama provider
//   - "<model>"             -> Ollama provider (default)
func NewProvider(cfg Config) (Provider, string, error) {
	model := cfg.Model
	switch {
	case strings.HasPrefix(model, "openrouter/"):
		actual := strings.TrimPrefix(model, "openrouter/")
		pcfg := cfg.Providers["openrouter"]
		p, err := newOpenRouterProvider(actual, pcfg)
		return p, actual, err
	case strings.HasPrefix(model, "opencode/"):
		actual := strings.TrimPrefix(model, "opencode/")
		pcfg := cfg.Providers["opencode"]
		p, err := newOpenCodeProvider(actual, pcfg)
		return p, actual, err
	case strings.HasPrefix(model, "ollama/"):
		actual := strings.TrimPrefix(model, "ollama/")
		pcfg := cfg.Providers["ollama"]
		p, err := newOllamaProvider(actual, pcfg)
		return p, actual, err
	default:
		pcfg := cfg.Providers["ollama"]
		p, err := newOllamaProvider(model, pcfg)
		return p, model, err
	}
}
