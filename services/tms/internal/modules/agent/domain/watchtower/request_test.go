package watchtower

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/shared/pulid"
)

// A request is kept on the item with who asked and what, and puts an open
// item in progress; a resolved item takes none.
func TestRecordRequestKeepsWhoAskedAndStartsTheItem(t *testing.T) {
	t.Parallel()

	by := pulid.MustNew("usr_")
	item := &Item{Status: StatusOpen}
	require.NoError(t, item.RecordRequest(by, " Dana Reyes ", " Asked Lee for the POD in Dash ", 500))

	require.NotNil(t, item.LastRequest)
	assert.Equal(t, int64(500), item.LastRequest.At)
	assert.Equal(t, by, item.LastRequest.ByID)
	assert.Equal(t, "Dana Reyes", item.LastRequest.ByName)
	assert.Equal(t, "Asked Lee for the POD in Dash", item.LastRequest.Summary)
	assert.Equal(t, StatusInProgress, item.Status)
	require.NotNil(t, item.AcknowledgedAt)
	assert.Equal(t, int64(500), *item.AcknowledgedAt)

	resolvedAt := int64(400)
	resolved := &Item{Status: StatusResolved, ResolvedAt: &resolvedAt}
	require.Error(t, resolved.RecordRequest(by, "Dana Reyes", "Asked again", 600))
	assert.Nil(t, resolved.LastRequest)
}
