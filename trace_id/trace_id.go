package trace_id

import (
	"strings"

	"github.com/google/uuid"
)

// New generates a new UUID v7 and returns it as a string with dashes.
func New() (string, error) {
	generated, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return generated.String(), nil
}

// RemoveDashes removes dashes from a UUID string and returns a 32-character hexadecimal string.
func RemoveDashes(id string) string {
	return strings.ReplaceAll(id, "-", "")
}
