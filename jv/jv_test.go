package jv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	// Valid JSON
	jsonStr := `{"name": "John Doe", "age": 30}`
	err := Validate(jsonStr, "user.json")
	assert.NoError(t, err)

	// Invalid JSON (missing required field)
	jsonStr = `{"name": "John Doe"}`
	err = Validate(jsonStr, "user.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing properties")

	// Invalid JSON (wrong type)
	jsonStr = `{"name": "John Doe", "age": "thirty"}`
	err = Validate(jsonStr, "user.json")
	assert.Error(t, err)
}

func TestValidateWithSchema(t *testing.T) {
	schemaContent := `{
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"items": { "type": "string" }
			}
		}
	}`

	// Valid JSON
	jsonStr := `{"tags": ["go", "json"]}`
	err := ValidateWithSchema(jsonStr, schemaContent)
	assert.NoError(t, err)

	// Invalid JSON
	jsonStr = `{"tags": [1, 2]}`
	err = ValidateWithSchema(jsonStr, schemaContent)
	assert.Error(t, err)
}

func TestListSchemas(t *testing.T) {
	schemas, err := ListSchemas()
	assert.NoError(t, err)
	assert.Contains(t, schemas, "user.json")
}

func TestGetSchema(t *testing.T) {
	content, err := GetSchema("user.json")
	assert.NoError(t, err)
	assert.Contains(t, content, "\"type\": \"object\"")
}
