package llm

import (
	"os"
	"time"
)

const (
	defaultBaseURL = "https://api.openai.com/v1"
	defaultModel   = "gpt-4o-mini"
	defaultTimeout = 120 * time.Second
)

// Config defines connection settings for an OpenAI-compatible chat API.
type Config struct {
	// APIKey is the bearer token used for authentication.
	APIKey string
	// BaseURL is the API root, for example https://api.openai.com/v1.
	BaseURL string
	// Model is the model name sent in chat completion requests.
	Model string
	// Timeout limits how long each HTTP request may take.
	Timeout time.Duration
}

// DefaultConfig returns configuration populated from environment variables when present.
// Supported variables: LLM_API_KEY, LLM_BASE_URL, LLM_MODEL.
func DefaultConfig() Config {
	config := Config{
		BaseURL: defaultBaseURL,
		Model:   defaultModel,
		Timeout: defaultTimeout,
	}

	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}
	if baseURL := os.Getenv("LLM_BASE_URL"); baseURL != "" {
		config.BaseURL = baseURL
	}
	if model := os.Getenv("LLM_MODEL"); model != "" {
		config.Model = model
	}

	return config
}

// LLMConfig is an alias for Config kept for backward compatibility.
type LLMConfig = Config
