package permit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusValues_AreEveryValidStatus(t *testing.T) {
	t.Parallel()

	values := StatusValues()
	assert.Len(t, values, 4)
	for _, value := range values {
		assert.True(t, value.IsValid(), value)
	}
	assert.False(t, Status("Revoked").IsValid())
}
