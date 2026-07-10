package sseq

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/SmilingXinyi/gb/trace_id"
)

// newTraceID generates a Seq-compatible 32-character hexadecimal trace id.
func newTraceID() (string, error) {
	return trace_id.NewHex()
}

// newSpanID generates a Seq-compatible 16-character hexadecimal span id.
func newSpanID() (string, error) {
	var randomBytes [8]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return "", fmt.Errorf("generate span id: %w", err)
	}
	return hex.EncodeToString(randomBytes[:]), nil
}
