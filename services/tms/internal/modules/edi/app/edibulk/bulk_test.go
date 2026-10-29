package edibulk

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestValidateBulkEDIActionIDs(t *testing.T) {
	t.Parallel()

	require.Error(t, ValidateIDs("messageIds", nil))
	require.NoError(t, ValidateIDs("messageIds", []pulid.ID{pulid.MustNew("edimsg_")}))

	tooMany := make([]pulid.ID, MaxItems+1)
	for index := range tooMany {
		tooMany[index] = pulid.MustNew("edimsg_")
	}
	require.Error(t, ValidateIDs("messageIds", tooMany))
}
