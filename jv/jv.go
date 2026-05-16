package jv

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schemas/*
var schemasFS embed.FS

// Validator is a JSON Schema validator that supports both embedded and external schemas.
type Validator struct {
	compiler *jsonschema.Compiler
}

// NewValidator creates a new Validator instance.
func NewValidator() *Validator {
	return &Validator{
		compiler: jsonschema.NewCompiler(),
	}
}

// Validate checks if a JSON string matches a specified schema file in the embedded filesystem.
// jsonString: The JSON string to validate.
// schemaName: The name of the schema file (relative to the schemas directory, e.g., "user.json").
func (validator *Validator) Validate(jsonString string, schemaName string) error {
	schemaPath := "schemas/" + schemaName
	schemaData, err := schemasFS.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file %s: %w", schemaName, err)
	}

	if err := validator.compiler.AddResource(schemaName, strings.NewReader(string(schemaData))); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := validator.compiler.Compile(schemaName)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	var jsonValue interface{}
	if err := json.Unmarshal([]byte(jsonString), &jsonValue); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if err := schema.Validate(jsonValue); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// Validate is a convenience function that uses a default validator to check a JSON string against an embedded schema.
func Validate(jsonString string, schemaName string) error {
	return NewValidator().Validate(jsonString, schemaName)
}

// ValidateWithSchema checks if a JSON string matches the provided schema content.
func ValidateWithSchema(jsonString string, schemaContent string) error {
	compiler := jsonschema.NewCompiler()
	schemaName := "schema.json"

	if err := compiler.AddResource(schemaName, strings.NewReader(schemaContent)); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := compiler.Compile(schemaName)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	var jsonValue interface{}
	if err := json.Unmarshal([]byte(jsonString), &jsonValue); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if err := schema.Validate(jsonValue); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// ListSchemas returns a list of all available schema files in the embedded filesystem.
func ListSchemas() ([]string, error) {
	entries, err := schemasFS.ReadDir("schemas")
	if err != nil {
		return nil, fmt.Errorf("failed to read schemas directory: %w", err)
	}

	var schemaFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			schemaFiles = append(schemaFiles, entry.Name())
		}
	}

	return schemaFiles, nil
}

// GetSchema returns the content of a specified schema file from the embedded filesystem.
func GetSchema(schemaName string) (string, error) {
	schemaPath := "schemas/" + schemaName
	data, err := schemasFS.ReadFile(schemaPath)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file %s: %w", schemaName, err)
	}
	return string(data), nil
}
