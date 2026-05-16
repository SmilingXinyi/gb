package main

import (
	"fmt"
	"log"

	"github.com/SmilingXinyi/gb/jv"
)

func main() {
	// 1. Validate using embedded schema
	validJSON := `{"name": "Alice", "age": 25, "email": "alice@example.com"}`
	if err := jv.Validate(validJSON, "user.json"); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("Alice is valid!")
	}

	invalidJSON := `{"name": "Bob", "age": -5}`
	if err := jv.Validate(invalidJSON, "user.json"); err != nil {
		fmt.Printf("Bob validation failed (expected): %v\n", err)
	}

	// 2. Validate using custom schema content
	customSchema := `{
		"type": "object",
		"properties": {
			"score": { "type": "number" }
		}
	}`
	scoreJSON := `{"score": 95.5}`
	if err := jv.ValidateWithSchema(scoreJSON, customSchema); err != nil {
		log.Fatalf("Custom validation failed: %v", err)
	}
	fmt.Println("Score is valid!")

	// 3. List available schemas
	schemas, err := jv.ListSchemas()
	if err != nil {
		log.Fatalf("Failed to list schemas: %v", err)
	}
	fmt.Printf("Available schemas: %v\n", schemas)
}
