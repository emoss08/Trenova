package floatutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 12.5, floatutils.Parse(" 12.5 "))
	assert.Equal(t, 0.0, floatutils.Parse(""))
	assert.Equal(t, 0.0, floatutils.Parse("bad"))
}
