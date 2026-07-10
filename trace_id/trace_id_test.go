package trace_id

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	idHex, err := New()
	assert.NoError(t, err)
	assert.Len(t, idHex, 32)
	assert.NotContains(t, idHex, "-")
	assert.Equal(t, idHex, strings.ToLower(idHex))
}
