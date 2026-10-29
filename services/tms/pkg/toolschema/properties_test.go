package toolschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameShape(t *testing.T) {
	t.Parallel()

	assert.Equal(t, NameShape("shipmentId"), NameShape("shipment_id"))
	assert.Equal(t, "ab.c", JoinPath("ab", "c"))
	assert.Equal(t, "c", JoinPath("", "c"))
}
