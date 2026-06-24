package writers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	clefTimestampField = "@t"
	clefLevelField     = "@l"
	clefMessageField   = "@mt"
	clefExceptionField = "@x"
)

var (
	zerologLevelField   = "level"
	zerologTimeField    = "time"
	zerologMessageField = "message"
	zerologErrorField   = "error"
)

// ConvertZerologJSONToCLEF transforms a zerolog JSON log line into a Seq CLEF event.
func ConvertZerologJSONToCLEF(payload []byte, application string) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return nil, fmt.Errorf("decode zerolog json: %w", err)
	}

	clefEvent := make(map[string]any, len(fields)+4)

	timestamp, err := extractTimestamp(fields)
	if err != nil {
		return nil, err
	}
	clefEvent[clefTimestampField] = timestamp

	if levelValue, ok := fields[zerologLevelField]; ok {
		var level string
		if err := json.Unmarshal(levelValue, &level); err != nil {
			return nil, fmt.Errorf("decode level: %w", err)
		}
		clefEvent[clefLevelField] = MapZerologLevelToCLEF(level)
		delete(fields, zerologLevelField)
	}

	if messageValue, ok := fields[zerologMessageField]; ok {
		var message string
		if err := json.Unmarshal(messageValue, &message); err != nil {
			return nil, fmt.Errorf("decode message: %w", err)
		}
		clefEvent[clefMessageField] = message
		delete(fields, zerologMessageField)
	}

	if errorValue, ok := fields[zerologErrorField]; ok {
		var exception string
		if err := json.Unmarshal(errorValue, &exception); err != nil {
			return nil, fmt.Errorf("decode error: %w", err)
		}
		clefEvent[clefExceptionField] = exception
		delete(fields, zerologErrorField)
	}

	delete(fields, zerologTimeField)

	if application != "" {
		clefEvent["Application"] = application
	}

	for fieldName, fieldValue := range fields {
		decodedValue, err := decodeJSONValue(fieldValue)
		if err != nil {
			return nil, fmt.Errorf("decode field %q: %w", fieldName, err)
		}
		clefEvent[fieldName] = decodedValue
	}

	return json.Marshal(clefEvent)
}

// MapZerologLevelToCLEF maps zerolog level names to Seq CLEF level names.
func MapZerologLevelToCLEF(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace":
		return "Verbose"
	case "debug":
		return "Debug"
	case "info":
		return "Information"
	case "warn":
		return "Warning"
	case "error":
		return "Error"
	case "fatal", "panic":
		return "Fatal"
	default:
		return "Information"
	}
}

// extractTimestamp reads the zerolog time field or falls back to the current UTC time.
func extractTimestamp(fields map[string]json.RawMessage) (string, error) {
	timeValue, ok := fields[zerologTimeField]
	if !ok {
		return time.Now().UTC().Format(time.RFC3339Nano), nil
	}

	var timestamp string
	if err := json.Unmarshal(timeValue, &timestamp); err != nil {
		return "", fmt.Errorf("decode timestamp: %w", err)
	}
	if timestamp == "" {
		return time.Now().UTC().Format(time.RFC3339Nano), nil
	}
	return timestamp, nil
}

// decodeJSONValue converts a raw JSON fragment into a Go value for CLEF encoding.
func decodeJSONValue(rawValue json.RawMessage) (any, error) {
	var decodedValue any
	if err := json.Unmarshal(rawValue, &decodedValue); err != nil {
		return nil, err
	}
	return decodedValue, nil
}
