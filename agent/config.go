package agent

// ProviderConfig holds settings for a single LLM provider.
type ProviderConfig struct {
	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
	Host    string `json:"host,omitempty"`
}

// Config is the top-level user configuration stored in config.json.
type Config struct {
	Model     string                    `json:"model"`
	Providers map[string]ProviderConfig `json:"providers,omitempty"`
}
