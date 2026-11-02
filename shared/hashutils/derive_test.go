package hashutils

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeriveKeySeparatesEveryLabelAndSecret(t *testing.T) {
	t.Parallel()

	first := DeriveKey("server-secret", "purpose one")

	assert.Len(t, first, 32)
	assert.Equal(t, first, DeriveKey("server-secret", "purpose one"))
	assert.NotEqual(t, first, DeriveKey("server-secret", "purpose two"))
	assert.NotEqual(t, first, DeriveKey("another-secret", "purpose one"))
}

func TestWritePartsSeparatesEveryPart(t *testing.T) {
	t.Parallel()

	var joined, shifted bytes.Buffer
	WriteParts(&joined, "ab", "c")
	WriteParts(&shifted, "a", "bc")

	assert.Equal(t, []byte("ab\x00c"), joined.Bytes())
	assert.NotEqual(t, joined.Bytes(), shifted.Bytes())
}
