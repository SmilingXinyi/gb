package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	_, err := NewClient(LLMConfig{
		BaseURL: "https://example.com/v1",
		Model:   "gpt-4o-mini",
	})
	assert.Error(t, err)
}

func TestClientChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/v1/chat/completions", request.URL.Path)
		assert.NotEmpty(t, request.Header.Get("Authorization"))
		assert.NotEmpty(t, request.Header.Get("X-Request-ID"))

		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
		assert.Equal(t, "gpt-4o-mini", payload["model"])

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"choices": [{
				"message": {
					"role": "assistant",
					"content": "Hello from the model"
				}
			}]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL + "/v1",
		Model:   "gpt-4o-mini",
	})
	require.NoError(t, err)

	reply, err := client.Chat(context.Background(), []Message{
		UserMessage("Say hello"),
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello from the model", reply)
}

func TestClientExtract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))

		responseFormat, ok := payload["response_format"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "json_schema", responseFormat["type"])

		jsonSchema, ok := responseFormat["json_schema"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "extracted_data", jsonSchema["name"])
		assert.Equal(t, true, jsonSchema["strict"])

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"choices": [{
				"message": {
					"role": "assistant",
					"content": "{\"name\":\"Ada\",\"age\":36}"
				}
			}]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL + "/v1",
		Model:   "gpt-4o-mini",
	})
	require.NoError(t, err)

	structureDescription := `{
		"name": "string",
		"age": "integer"
	}`

	result, err := client.Extract(
		context.Background(),
		"Extract person data",
		structureDescription,
	)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.Equal(t, "Ada", parsed["name"])
	assert.EqualValues(t, 36, parsed["age"])
}

func TestClientExtractRejectsInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"choices": [{
				"message": {
					"role": "assistant",
					"content": "{\"name\":\"Ada\"}"
				}
			}]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(LLMConfig{
		APIKey:  "test-key",
		BaseURL: server.URL + "/v1",
		Model:   "gpt-4o-mini",
	})
	require.NoError(t, err)

	_, err = client.Extract(
		context.Background(),
		"Extract person data",
		`{"name": "string", "age": "integer"}`,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validate extracted JSON")
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
