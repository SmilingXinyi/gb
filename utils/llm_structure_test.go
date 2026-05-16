package utils

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildLLMStructure(t *testing.T) {
	input := `{
		"name": "string",
		"age": "integer",
		"status": "string enum(active, inactive)",
		"tags": ["string"]
	}`

	schemaStr, err := BuildLLMStructure(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, schemaStr)

	var schema map[string]interface{}
	err = json.Unmarshal([]byte(schemaStr), &schema)
	assert.NoError(t, err)

	assert.Equal(t, "object", schema["type"])
	properties := schema["properties"].(map[string]interface{})
	assert.Contains(t, properties, "name")
	assert.Contains(t, properties, "age")
	assert.Contains(t, properties, "status")
	assert.Contains(t, properties, "tags")

	status := properties["status"].(map[string]interface{})
	assert.Equal(t, "string", status["type"])
	assert.ElementsMatch(t, []interface{}{"active", "inactive"}, status["enum"])

	tags := properties["tags"].(map[string]interface{})
	assert.Equal(t, "array", tags["type"])
	items := tags["items"].(map[string]interface{})
	assert.Equal(t, "string", items["type"])
}
