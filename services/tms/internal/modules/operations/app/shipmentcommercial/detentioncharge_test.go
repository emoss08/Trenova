package shipmentcommercial

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestEnsureGeneratedDetentionCharge_ReusesTheOneRow(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	accessorial := &accessorialcharge.AccessorialCharge{
		ID:     pulid.MustNew("acc_"),
		Method: accessorialcharge.MethodPerUnit,
		Amount: decimal.NewFromInt(75),
	}

	ensureGeneratedDetentionCharge(entity, accessorial, 2)
	require.Len(t, entity.AdditionalCharges, 1)
	require.True(t, entity.AdditionalCharges[0].IsDetention)
	require.Equal(t, int16(2), entity.AdditionalCharges[0].Unit)

	entity.AdditionalCharges[0].ID = pulid.MustNew("ac_")
	persistedID := entity.AdditionalCharges[0].ID

	ensureGeneratedDetentionCharge(entity, accessorial, 4)
	require.Len(t, entity.AdditionalCharges, 1)
	require.Equal(t, persistedID, entity.AdditionalCharges[0].ID)
	require.Equal(t, int16(4), entity.AdditionalCharges[0].Unit)

	removeGeneratedDetentionCharges(entity, accessorial.ID)
	require.Empty(t, entity.AdditionalCharges)
}
