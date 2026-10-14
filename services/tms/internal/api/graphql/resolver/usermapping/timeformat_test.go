package usermapping

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeFormatRoundTrips(t *testing.T) {
	t.Parallel()

	for _, value := range []domaintypes.TimeFormat{
		domaintypes.TimeFormat12Hour,
		domaintypes.TimeFormat24Hour,
	} {
		back, err := TimeFormatToDomain(TimeFormatFromDomain(value))
		require.NoError(t, err)
		assert.Equal(t, value, back)
	}
}

func TestTimeFormatToDomainRefusesAnUnknownValue(t *testing.T) {
	t.Parallel()

	_, err := TimeFormatToDomain(gqlmodel.TimeFormat("Sundial"))

	require.Error(t, err)
}
