package base

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionalPulid(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("tr_")

	parsed, err := OptionalPulid(nil)
	require.NoError(t, err)
	assert.Nil(t, parsed, "an absent optional ID is not an error")

	value := id.String()
	parsed, err = OptionalPulid(&value)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, id, *parsed)

	empty := ""
	parsed, err = OptionalPulid(&empty)
	require.NoError(t, err)
	assert.Nil(t, parsed)

	garbage := "not-an-id"
	_, err = OptionalPulid(&garbage)
	assert.Error(t, err)
}
