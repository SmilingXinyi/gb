package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSessionDefaults(t *testing.T) {
	session := NewSession("test-key", "https://example.com/v1", "gpt-4o-mini")
	assert.NotNil(t, session)
	assert.NotNil(t, session.model)
	assert.Len(t, session.messages, 0)
	assert.Len(t, session.opts, 1)
}

func TestSessionFluentAPI(t *testing.T) {
	session := NewSession("test-key", "https://example.com/v1", "gpt-4o-mini").
		WithSystem("You are helpful.").
		AddUser("Hello").
		AddAssistant("Hi").
		AddHistory([]Message{
			{Role: RoleUser, Content: "Earlier question"},
		}).
		WithTemperature(0.5).
		WithMaxOutputTokens(128)

	assert.Equal(t, "You are helpful.", session.system)
	assert.Len(t, session.messages, 3)
	assert.Len(t, session.opts, 3)
}

func TestSessionBuildOptsIncludesSystemAndMessages(t *testing.T) {
	session := NewSession("test-key", "https://example.com/v1", "gpt-4o-mini").
		WithSystem("system prompt").
		AddUser("user prompt")

	opts := session.buildOpts()
	assert.GreaterOrEqual(t, len(opts), 2)
}

func TestDefaultConfigReadsEnvironment(t *testing.T) {
	t.Setenv("LLM_API_KEY", "env-key")
	t.Setenv("LLM_BASE_URL", "https://custom.example/v1")
	t.Setenv("LLM_MODEL", "custom-model")

	config := DefaultConfig()
	assert.Equal(t, "env-key", config.APIKey)
	assert.Equal(t, "https://custom.example/v1", config.BaseURL)
	assert.Equal(t, "custom-model", config.Model)
}

func TestNewSessionFromConfig(t *testing.T) {
	session := NewSessionFromConfig(Config{
		APIKey:  "key",
		BaseURL: "https://example.com/v1",
		Model:   "gpt-4o-mini",
		Timeout: defaultTimeout,
	})
	assert.NotNil(t, session)
}
