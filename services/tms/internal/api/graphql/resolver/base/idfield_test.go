package base

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIDField_NamesTheFieldOfAMalformedID(t *testing.T) {
	t.Parallel()

	_, err := ParseIDField("suggestedAgentId", "not-an-id")

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "suggestedAgentId", validation.Field)
	assert.Equal(t, errortypes.ErrInvalid, validation.Code)
}

func TestParseOptionalIDField_ReadsAnAbsentIDAsNone(t *testing.T) {
	t.Parallel()

	empty := ""
	for _, raw := range []*string{nil, &empty} {
		id, err := ParseOptionalIDField("currentAgentId", raw)
		require.NoError(t, err)
		assert.True(t, id.IsNil())
	}

	want := pulid.MustNew("agdef_")
	raw := want.String()
	id, err := ParseOptionalIDField("currentAgentId", &raw)
	require.NoError(t, err)
	assert.Equal(t, want, id)
}
