package writers

import (
	"encoding/json"
	"testing"
)

func TestMapZerologLevelToCLEF(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "trace", expected: "Verbose"},
		{input: "debug", expected: "Debug"},
		{input: "info", expected: "Information"},
		{input: "warn", expected: "Warning"},
		{input: "error", expected: "Error"},
		{input: "fatal", expected: "Fatal"},
		{input: "panic", expected: "Fatal"},
		{input: "unknown", expected: "Information"},
	}

	for _, testCase := range tests {
		t.Run(testCase.input, func(t *testing.T) {
			got := MapZerologLevelToCLEF(testCase.input)
			if got != testCase.expected {
				t.Fatalf("MapZerologLevelToCLEF(%q) = %q, want %q", testCase.input, got, testCase.expected)
			}
		})
	}
}

func TestConvertZerologJSONToCLEF(t *testing.T) {
	payload := []byte(`{"level":"info","time":"2026-06-24T12:00:00Z","message":"hello","module":"auth","caller":"log/log.go:10"}`)

	clefPayload, err := ConvertZerologJSONToCLEF(payload, "demo-app")
	if err != nil {
		t.Fatalf("ConvertZerologJSONToCLEF() error = %v", err)
	}

	var clefEvent map[string]any
	if err := json.Unmarshal(clefPayload, &clefEvent); err != nil {
		t.Fatalf("decode clef payload: %v", err)
	}

	if clefEvent["@t"] != "2026-06-24T12:00:00Z" {
		t.Fatalf("@t = %v, want timestamp", clefEvent["@t"])
	}
	if clefEvent["@l"] != "Information" {
		t.Fatalf("@l = %v, want Information", clefEvent["@l"])
	}
	if clefEvent["@mt"] != "hello" {
		t.Fatalf("@mt = %v, want hello", clefEvent["@mt"])
	}
	if clefEvent["Application"] != "demo-app" {
		t.Fatalf("Application = %v, want demo-app", clefEvent["Application"])
	}
	if clefEvent["module"] != "auth" {
		t.Fatalf("module = %v, want auth", clefEvent["module"])
	}
	if clefEvent["caller"] != "log/log.go:10" {
		t.Fatalf("caller = %v, want log/log.go:10", clefEvent["caller"])
	}
}

func TestConvertZerologJSONToCLEFErrorField(t *testing.T) {
	payload := []byte(`{"level":"error","time":"2026-06-24T12:00:00Z","message":"failed","error":"boom"}`)

	clefPayload, err := ConvertZerologJSONToCLEF(payload, "")
	if err != nil {
		t.Fatalf("ConvertZerologJSONToCLEF() error = %v", err)
	}

	var clefEvent map[string]any
	if err := json.Unmarshal(clefPayload, &clefEvent); err != nil {
		t.Fatalf("decode clef payload: %v", err)
	}

	if clefEvent["@x"] != "boom" {
		t.Fatalf("@x = %v, want boom", clefEvent["@x"])
	}
}
