package carrierassignmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAssignmentPaysAPerMileRateOnThePaidDistance(t *testing.T) {
	t.Parallel()

	routed, paid := 400.0, 500.0
	move := &shipment.ShipmentMove{ID: pulid.MustNew("sm_"), Distance: &routed, PayDistance: &paid}
	req := &carrierassignmentports.AssignMoveToCarrierRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		ShipmentMoveID: move.ID,
		CarrierID:      pulid.MustNew("car_"),
		RateMethod:     shipmentvocab.CarrierRateMethodPerMile,
		BaseRate:       decimal.RequireFromString("2.00"),
	}

	entity := (&Service{}).buildAssignment(req, move)

	assert.Equal(t, "1000", entity.BaseAmount.String())
}

func TestValidatePerMileDistanceReadsThePaidDistance(t *testing.T) {
	t.Parallel()

	paid := 500.0
	paidOnly := &shipment.ShipmentMove{PayDistance: &paid}
	require.NoError(
		t,
		validatePerMileDistance(shipmentvocab.CarrierRateMethodPerMile, paidOnly.PaidDistance()),
	)
	require.Error(
		t,
		validatePerMileDistance(
			shipmentvocab.CarrierRateMethodPerMile,
			(&shipment.ShipmentMove{}).PaidDistance(),
		),
	)
}
