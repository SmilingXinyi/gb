// Package llm provides goai-based large language model helpers.
// It supports fluent session configuration, structured output via GenerateObject,
// and streaming via StreamText. Under the hood, goai/provider/compat connects to
// any OpenAI-compatible API without depending on go-openai.
package llm

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/SmilingXinyi/gb/log"
	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/compat"
)

var moduleLogger = log.Module("llm")

const (
	defaultBaseURL = "https://api.openai.com/v1"
	defaultModel   = "gpt-4o-mini"
	defaultTimeout = 120 * time.Second
)

// Config defines connection settings for an OpenAI-compatible chat API.
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

// DefaultConfig returns configuration from LLM_API_KEY, LLM_BASE_URL, and LLM_MODEL when set.
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

// Role is a strongly typed chat message role.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is used to inject conversation history via AddHistory.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Session manages state for a single model interaction using a fluent API.
type Session struct {
	model    provider.LanguageModel
	system   string
	messages []provider.Message
	opts     []goai.Option
	err      error
}

// NewSession creates a Session. Proxy is disabled for intranet use; default temperature is 0.01.
func NewSession(token, baseURL, model string) *Session {
	return newSession(token, baseURL, model, defaultTimeout)
}

// NewSessionFromConfig creates a Session from Config.
func NewSessionFromConfig(config Config) *Session {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return newSession(config.APIKey, config.BaseURL, config.Model, timeout)
}

func newSession(token, baseURL, model string, timeout time.Duration) *Session {
	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	languageModel := compat.Chat(
		model,
		compat.WithAPIKey(token),
		compat.WithBaseURL(baseURL),
		compat.WithHTTPClient(httpClient),
	)
	return &Session{
		model:    languageModel,
		messages: make([]provider.Message, 0),
		opts:     []goai.Option{goai.WithTemperature(0.01)},
	}
}

// WithSystem sets or replaces the system instruction.
func (session *Session) WithSystem(content string) *Session {
	if content != "" {
		session.system = content
	}
	return session
}

// AddUser appends a user message.
func (session *Session) AddUser(content string) *Session {
	session.messages = append(session.messages, goai.UserMessage(content))
	return session
}

// AddAssistant appends an assistant message from conversation history.
func (session *Session) AddAssistant(content string) *Session {
	session.messages = append(session.messages, goai.AssistantMessage(content))
	return session
}

// AddHistory batch-injects external history messages.
func (session *Session) AddHistory(histories []Message) *Session {
	for _, history := range histories {
		switch history.Role {
		case RoleUser:
			session.messages = append(session.messages, goai.UserMessage(history.Content))
		case RoleAssistant:
			session.messages = append(session.messages, goai.AssistantMessage(history.Content))
		case RoleSystem:
		}
	}
	return session
}

// WithTemperature adjusts sampling randomness.
func (session *Session) WithTemperature(temperature float64) *Session {
	session.opts = append(session.opts, goai.WithTemperature(temperature))
	return session
}

// WithMaxOutputTokens limits the maximum number of output tokens.
func (session *Session) WithMaxOutputTokens(tokenLimit int) *Session {
	session.opts = append(session.opts, goai.WithMaxOutputTokens(tokenLimit))
	return session
}

func (session *Session) buildOpts() []goai.Option {
	opts := make([]goai.Option, 0, len(session.opts)+2)
	if session.system != "" {
		opts = append(opts, goai.WithSystem(session.system))
	}
	if len(session.messages) > 0 {
		opts = append(opts, goai.WithMessages(session.messages...))
	}
	return append(opts, session.opts...)
}

// Execute runs text generation and returns plain text.
func (session *Session) Execute(ctx context.Context) (string, error) {
	if session.err != nil {
		return "", session.err
	}
	moduleLogger.Debug().
		Str("model", session.model.ModelID()).
		Int("messages", len(session.messages)).
		Msg("sending chat completion request")

	result, err := goai.GenerateText(ctx, session.model, session.buildOpts()...)
	if err != nil {
		moduleLogger.Error().Err(err).Msg("llm execute failed")
		return "", fmt.Errorf("llm execute failed: %w", err)
	}
	moduleLogger.Debug().Str("content", result.Text).Msg("chat completion response received")
	return result.Text, nil
}

// ExecuteTo runs structured output into type T via goai.GenerateObject.
func ExecuteTo[T any](ctx context.Context, session *Session) (*T, error) {
	if session.err != nil {
		return nil, session.err
	}
	moduleLogger.Debug().
		Str("model", session.model.ModelID()).
		Int("messages", len(session.messages)).
		Msg("sending structured output request")

	result, err := goai.GenerateObject[T](ctx, session.model, session.buildOpts()...)
	if err != nil {
		moduleLogger.Error().Err(err).Msg("llm structured output failed")
		return nil, fmt.Errorf("llm structured output failed: %w", err)
	}
	moduleLogger.Debug().Msg("structured output completed")
	return &result.Object, nil
}

// Stream returns a channel of streamed text chunks. Consume it fully or cancel ctx.
func (session *Session) Stream(ctx context.Context) (<-chan string, error) {
	if session.err != nil {
		return nil, session.err
	}
	textStream, err := goai.StreamText(ctx, session.model, session.buildOpts()...)
	if err != nil {
		return nil, fmt.Errorf("llm stream request failed: %w", err)
	}
	return textStream.TextStream(), nil
}
