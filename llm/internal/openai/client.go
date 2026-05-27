package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client calls an OpenAI-compatible chat completions endpoint.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// Config holds settings for the OpenAI-compatible HTTP client.
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

// NewClient creates a client for chat completions.
func NewClient(config Config) (*Client, error) {
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model is required")
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	return &Client{
		apiKey:  config.APIKey,
		baseURL: strings.TrimRight(config.BaseURL, "/"),
		model:   config.Model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// ChatCompletion sends a chat completion request and returns the assistant text.
func (client *Client) ChatCompletion(
	ctx context.Context,
	request ChatCompletionRequest,
	traceID string,
) (string, error) {
	request.Model = client.model

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	endpoint := client.baseURL + "/chat/completions"
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("create chat request: %w", err)
	}

	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+client.apiKey)
	if traceID != "" {
		httpRequest.Header.Set("X-Request-ID", traceID)
	}

	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return "", fmt.Errorf("chat request failed: %w", err)
	}
	defer httpResponse.Body.Close()

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return "", fmt.Errorf("read chat response: %w", err)
	}

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return "", parseErrorResponse(httpResponse.StatusCode, responseBody)
	}

	var completion ChatCompletionResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}

	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("chat response has no choices")
	}

	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("chat response content is empty")
	}

	return content, nil
}

func parseErrorResponse(statusCode int, responseBody []byte) error {
	var apiError ErrorResponse
	if err := json.Unmarshal(responseBody, &apiError); err == nil && apiError.Error.Message != "" {
		return fmt.Errorf("chat API error (status %d): %s", statusCode, apiError.Error.Message)
	}
	return fmt.Errorf("chat API error (status %d): %s", statusCode, strings.TrimSpace(string(responseBody)))
}
