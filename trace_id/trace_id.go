package trace_id

import (
	"strings"

	"github.com/google/uuid"
)

// New generates a new UUID v7 trace id as a 32-character hexadecimal string without dashes.
func New() (string, error) {
	generated, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(generated.String(), "-", ""), nil
}
