package llm

import (
	"context"
	"fmt"

	"github.com/SmilingXinyi/gb/llm/internal/openai"
	"github.com/SmilingXinyi/gb/trace_id"
)

// Client provides chat and structured extraction against an OpenAI-compatible API.
type Client struct {
	config       LLMConfig
	openaiClient *openai.Client
}

// NewClient creates a client from the given configuration.
func NewClient(config LLMConfig) (*Client, error) {
	openaiClient, err := openai.NewClient(openai.Config{
		APIKey:  config.APIKey,
		BaseURL: config.BaseURL,
		Model:   config.Model,
		Timeout: config.Timeout,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		config:       config,
		openaiClient: openaiClient,
	}, nil
}

// Chat sends messages to the model and returns the assistant reply text.
func (client *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("at least one message is required")
	}

	traceIdentifier, err := trace_id.NewString()
	if err != nil {
		return "", fmt.Errorf("generate trace id: %w", err)
	}

	request := openai.ChatCompletionRequest{
		Messages: toOpenAIMessages(messages),
	}

	return client.openaiClient.ChatCompletion(ctx, request, traceIdentifier)
}

func toOpenAIMessages(messages []Message) []openai.ChatMessage {
	openaiMessages := make([]openai.ChatMessage, len(messages))
	for index, message := range messages {
		openaiMessages[index] = openai.ChatMessage{
			Role:    string(message.Role),
			Content: message.Content,
		}
	}
	return openaiMessages
}
