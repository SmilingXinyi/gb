package trace_id

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ID is a 128-bit distributed trace identifier.
// Values created by this package use UUID version 7.
type ID uuid.UUID

// String returns the canonical UUID string representation with dashes.
func (id ID) String() string {
	return uuid.UUID(id).String()
}

// Hex returns a Seq and W3C compatible 32-character lowercase hexadecimal string without dashes.
func (id ID) Hex() string {
	return strings.ReplaceAll(id.String(), "-", "")
}

// UUID returns the underlying google/uuid value.
func (id ID) UUID() uuid.UUID {
	return uuid.UUID(id)
}

// Version returns the UUID version nibble.
func (id ID) Version() uuid.Version {
	return uuid.UUID(id).Version()
}

// IsZero reports whether the ID is the zero value.
func (id ID) IsZero() bool {
	return uuid.UUID(id) == uuid.Nil
}

// Parse parses a trace ID from a canonical UUID string or a 32-character hexadecimal string.
func Parse(raw string) (ID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ID{}, fmt.Errorf("parse trace id: empty input")
	}

	if len(trimmed) == 32 && isHexString(trimmed) {
		decoded, decodeErr := hex.DecodeString(trimmed)
		if decodeErr != nil {
			return ID{}, fmt.Errorf("parse trace id: %w", decodeErr)
		}
		var parsed uuid.UUID
		copy(parsed[:], decoded)
		return ID(parsed), nil
	}

	parsed, parseErr := uuid.Parse(trimmed)
	if parseErr != nil {
		return ID{}, fmt.Errorf("parse trace id: %w", parseErr)
	}
	return ID(parsed), nil
}

// IsValid reports whether raw is a valid trace ID in UUID or 32-hex format.
func IsValid(raw string) bool {
	_, err := Parse(raw)
	return err == nil
}

// isHexString reports whether value contains only hexadecimal characters.
func isHexString(value string) bool {
	for _, character := range value {
		switch {
		case character >= '0' && character <= '9':
		case character >= 'a' && character <= 'f':
		case character >= 'A' && character <= 'F':
		default:
			return false
		}
	}
	return true
}
