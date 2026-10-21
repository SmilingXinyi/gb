package ss

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"time"
)

// SanitizeAttributes returns a JSON-safe copy of attributes.
// A value that encoding/json cannot marshal is replaced, and sibling keys are kept.
func SanitizeAttributes(attributes map[string]any) map[string]any {
	if len(attributes) == 0 {
		return nil
	}
	sanitized := make(map[string]any, len(attributes))
	for key, value := range attributes {
		sanitized[key] = SanitizeValue(value)
	}
	return sanitized
}

// SanitizeValue returns a value that encoding/json can marshal.
// NaN and infinity become strings. Channels, functions, and cyclic values become placeholders.
func SanitizeValue(value any) any {
	return sanitizeValue(value, map[uintptr]struct{}{})
}

// sanitizeValue walks nested maps and slices, remembering addresses to break cycles.
func sanitizeValue(value any, seen map[uintptr]struct{}) any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		json.Number, time.Time, []byte:
		return typed
	case float32:
		if !isFiniteFloat(float64(typed)) {
			return finiteFloatText(float64(typed))
		}
		return typed
	case float64:
		if !isFiniteFloat(typed) {
			return finiteFloatText(typed)
		}
		return typed
	case map[string]any:
		return sanitizeStringMap(typed, seen)
	case []any:
		return sanitizeSlice(typed, seen)
	default:
		if _, err := json.Marshal(typed); err != nil {
			return unencodableText(typed)
		}
		return typed
	}
}

// sanitizeStringMap copies a string map, replacing values that are not JSON-safe.
func sanitizeStringMap(attributes map[string]any, seen map[uintptr]struct{}) any {
	if attributes == nil {
		return nil
	}
	if !enter(reflect.ValueOf(attributes).Pointer(), seen) {
		return "<cycle>"
	}
	defer leave(reflect.ValueOf(attributes).Pointer(), seen)

	sanitized := make(map[string]any, len(attributes))
	for key, value := range attributes {
		sanitized[key] = sanitizeValue(value, seen)
	}
	return sanitized
}

// sanitizeSlice copies a slice, replacing items that are not JSON-safe.
func sanitizeSlice(values []any, seen map[uintptr]struct{}) any {
	if values == nil {
		return nil
	}
	if !enter(reflect.ValueOf(values).Pointer(), seen) {
		return "<cycle>"
	}
	defer leave(reflect.ValueOf(values).Pointer(), seen)

	sanitized := make([]any, len(values))
	for index, item := range values {
		sanitized[index] = sanitizeValue(item, seen)
	}
	return sanitized
}

// enter records pointer in seen. It returns false when pointer is already present.
func enter(pointer uintptr, seen map[uintptr]struct{}) bool {
	if pointer == 0 {
		return true
	}
	if _, exists := seen[pointer]; exists {
		return false
	}
	seen[pointer] = struct{}{}
	return true
}

// leave removes pointer from seen so a later sibling can visit it again.
func leave(pointer uintptr, seen map[uintptr]struct{}) {
	if pointer == 0 {
		return
	}
	delete(seen, pointer)
}

// isFiniteFloat reports whether value is neither NaN nor infinity.
func isFiniteFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// finiteFloatText renders a non-finite float as a JSON string.
func finiteFloatText(value float64) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "+Inf"
	default:
		return "-Inf"
	}
}

// unencodableText describes a value that encoding/json rejected.
func unencodableText(value any) string {
	return fmt.Sprintf("<unmarshalable %T>", value)
}
