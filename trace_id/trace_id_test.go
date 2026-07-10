package trace_id

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	idString, err := New()
	assert.NoError(t, err)
	assert.Contains(t, idString, "-")

	id, err := uuid.Parse(idString)
	assert.NoError(t, err)
	assert.Equal(t, uuid.Version(7), id.Version())
}

func TestRemoveDashes(t *testing.T) {
	idString, err := New()
	assert.NoError(t, err)

	idHex := RemoveDashes(idString)
	assert.Len(t, idHex, 32)
	assert.NotContains(t, idHex, "-")
	assert.Equal(t, idHex, strings.ToLower(idHex))
	assert.Equal(t, RemoveDashes("019f4b37-35f3-7ca0-bdfa-ad88c25d618d"), "019f4b3735f37ca0bdfaad88c25d618d")
}
