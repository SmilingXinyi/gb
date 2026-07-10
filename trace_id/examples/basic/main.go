package main

import (
	"fmt"

	"github.com/SmilingXinyi/gb/trace_id"
)

func main() {
	idString, err := trace_id.NewString()
	if err != nil {
		fmt.Printf("Failed to generate trace ID: %v\n", err)
		return
	}
	fmt.Printf("With dashes: %s\n", idString)

	idHex, err := trace_id.NewHex()
	if err != nil {
		fmt.Printf("Failed to generate trace ID hex: %v\n", err)
		return
	}
	fmt.Printf("Without dashes: %s\n", idHex)

	fmt.Printf("Must without dashes: %s\n", trace_id.MustNewHex())
}
