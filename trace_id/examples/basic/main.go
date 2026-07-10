package main

import (
	"fmt"

	"github.com/SmilingXinyi/gb/trace_id"
)

func main() {
	traceID, err := trace_id.New()
	if err != nil {
		fmt.Printf("Failed to generate trace ID: %v\n", err)
		return
	}
	fmt.Printf("With dashes: %s\n", traceID)
	fmt.Printf("Without dashes: %s\n", trace_id.RemoveDashes(traceID))
}
