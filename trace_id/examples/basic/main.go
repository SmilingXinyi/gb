package main

import (
	"fmt"

	"github.com/SmilingXinyi/gb/trace_id"
)

func main() {
	// Generate a new trace ID as a typed value
	traceID, err := trace_id.New()
	if err != nil {
		fmt.Printf("Failed to generate trace ID: %v\n", err)
		return
	}
	fmt.Printf("Generated Trace ID: %s\n", traceID.String())
	fmt.Printf("Generated Trace ID Hex: %s\n", traceID.Hex())

	// Generate a new trace ID as a canonical UUID string
	traceIDString, err := trace_id.NewString()
	if err != nil {
		fmt.Printf("Failed to generate trace ID string: %v\n", err)
		return
	}
	fmt.Printf("Generated Trace ID String: %s\n", traceIDString)

	// Generate a Seq and W3C compatible 32-character hex trace ID
	traceIDHex, err := trace_id.NewHex()
	if err != nil {
		fmt.Printf("Failed to generate trace ID hex: %v\n", err)
		return
	}
	fmt.Printf("Generated Trace ID Hex String: %s\n", traceIDHex)

	// Generate using Must functions (panics on error)
	mustTraceID := trace_id.MustNew()
	fmt.Printf("Must Generated Trace ID: %s\n", mustTraceID.String())

	mustTraceIDString := trace_id.MustNewString()
	fmt.Printf("Must Generated Trace ID String: %s\n", mustTraceIDString)

	mustTraceIDHex := trace_id.MustNewHex()
	fmt.Printf("Must Generated Trace ID Hex: %s\n", mustTraceIDHex)

	parsedTraceID, err := trace_id.Parse(mustTraceIDHex)
	if err != nil {
		fmt.Printf("Failed to parse trace ID: %v\n", err)
		return
	}
	fmt.Printf("Parsed Trace ID: %s\n", parsedTraceID.String())
}
