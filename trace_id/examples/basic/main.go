package main

import (
	"fmt"

	"github.com/SmilingXinyi/gb/trace_id"
)

func main() {
	// Generate a new trace ID as a UUID object
	id, err := trace_id.New()
	if err != nil {
		fmt.Printf("Failed to generate trace ID: %v\n", err)
		return
	}
	fmt.Printf("Generated Trace ID: %s\n", id.String())

	// Generate a new trace ID as a string
	idStr, err := trace_id.NewString()
	if err != nil {
		fmt.Printf("Failed to generate trace ID string: %v\n", err)
		return
	}
	fmt.Printf("Generated Trace ID String: %s\n", idStr)

	// Generate using Must functions (panics on error)
	mustId := trace_id.MustNew()
	fmt.Printf("Must Generated Trace ID: %s\n", mustId.String())

	mustIdStr := trace_id.MustNewString()
	fmt.Printf("Must Generated Trace ID String: %s\n", mustIdStr)
}
