package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinAlternatives(t *testing.T) {
	t.Parallel()

	assert.Empty(t, JoinAlternatives(nil))
	assert.Equal(t, "a shipment", JoinAlternatives([]string{"a shipment"}))
	assert.Equal(t, "a shipment or an invoice",
		JoinAlternatives([]string{"a shipment", "an invoice"}))
	assert.Equal(t, "a shipment, an invoice or a worker",
		JoinAlternatives([]string{"a shipment", "an invoice", "a worker"}))
}
