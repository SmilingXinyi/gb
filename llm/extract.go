package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SmilingXinyi/gb/jv"
	"github.com/SmilingXinyi/gb/llm/internal/openai"
	"github.com/SmilingXinyi/gb/trace_id"
	"github.com/SmilingXinyi/gb/utils"
)

const extractSchemaName = "extracted_data"

// Extract asks the model to return JSON that matches a simplified structure description.
// The description uses the same format as utils.BuildLLMStructure, and the response is
// validated with jv before it is returned.
func (client *Client) Extract(ctx context.Context, prompt string, structureDescription string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	if structureDescription == "" {
		return "", fmt.Errorf("structure description is required")
	}

	schemaContent, err := utils.BuildLLMStructure(structureDescription)
	if err != nil {
		return "", fmt.Errorf("build JSON schema: %w", err)
	}

	var schemaDocument map[string]interface{}
	if err := json.Unmarshal([]byte(schemaContent), &schemaDocument); err != nil {
		return "", fmt.Errorf("parse JSON schema: %w", err)
	}

	traceIdentifier, err := trace_id.NewString()
	if err != nil {
		return "", fmt.Errorf("generate trace id: %w", err)
	}

	request := openai.ChatCompletionRequest{
		Messages: []openai.ChatMessage{
			{
				Role:    string(RoleSystem),
				Content: "You extract structured data. Reply with JSON only, matching the provided schema.",
			},
			{
				Role:    string(RoleUser),
				Content: prompt,
			},
		},
		ResponseFormat: &openai.ResponseFormat{
			Type: "json_schema",
			JSONSchema: openai.JSONSchema{
				Name:   extractSchemaName,
				Strict: true,
				Schema: schemaDocument,
			},
		},
	}

	responseContent, err := client.openaiClient.ChatCompletion(ctx, request, traceIdentifier)
	if err != nil {
		return "", err
	}

	if err := jv.ValidateWithSchema(responseContent, schemaContent); err != nil {
		return "", fmt.Errorf("validate extracted JSON: %w", err)
	}

	return responseContent, nil
}
