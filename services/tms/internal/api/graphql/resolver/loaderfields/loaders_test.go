package loaderfields

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionalMatchTreatsARecordOutOfReachAsAbsent(t *testing.T) {
	t.Parallel()

	summary := &shipmentports.ShipmentSummary{ProNumber: "PRO-1"}

	got, err := OptionalMatch(summary, nil)
	require.NoError(t, err)
	assert.Same(t, summary, got)

	got, err = OptionalMatch[*shipmentports.ShipmentSummary](
		nil, errortypes.NewNotFoundError("Shipment not found within your organization"),
	)
	require.NoError(t, err, "a deleted or out-of-tenant shipment is no match, not a failed inbox")
	assert.Nil(t, got)

	boom := errors.New("database is down")
	_, err = OptionalMatch[*shipmentports.ShipmentSummary](nil, boom)
	assert.ErrorIs(t, err, boom, "a real failure is still a failure")
}
