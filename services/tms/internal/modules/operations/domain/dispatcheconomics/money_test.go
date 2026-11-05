package dispatcheconomics_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/stretchr/testify/assert"
)

func TestDollarsRoundsToWholeDollarsWithSign(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "$42", dispatcheconomics.Dollars(4249))
	assert.Equal(t, "$43", dispatcheconomics.Dollars(4250))
	assert.Equal(t, "$0", dispatcheconomics.Dollars(0))
	assert.Equal(t, "-$63", dispatcheconomics.Dollars(-6300))
}
