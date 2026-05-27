package openai

// ChatCompletionRequest is the payload for POST /chat/completions.
type ChatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []ChatMessage   `json:"messages"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// ChatMessage is a single message in a chat completion request.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseFormat configures structured output for compatible providers.
type ResponseFormat struct {
	Type       string     `json:"type"`
	JSONSchema JSONSchema `json:"json_schema"`
}

// JSONSchema wraps a JSON Schema document for strict structured outputs.
type JSONSchema struct {
	Name   string                 `json:"name"`
	Strict bool                   `json:"strict"`
	Schema map[string]interface{} `json:"schema"`
}

// ChatCompletionResponse is the successful response from POST /chat/completions.
type ChatCompletionResponse struct {
	Choices []ChatCompletionChoice `json:"choices"`
}

// ChatCompletionChoice contains one completion candidate.
type ChatCompletionChoice struct {
	Message ChatMessage `json:"message"`
}

// ErrorResponse represents an API error body.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError describes a provider error payload.
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}
