package dispatchconsoleresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignMoveToCarrierRequestFromInput_CarriesTheAutoRateAndMarginOverrides(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_")
	carrierID := pulid.MustNew("car_")
	tenant := resolvercore.TenantInfo(resolvertest.UserAuthContext())
	yes := true

	req, err := assignMoveToCarrierRequestFromInput(&gqlmodel.DispatchAssignMoveToCarrierInput{
		MoveID:              moveID.String(),
		CarrierID:           carrierID.String(),
		RateMethod:          shipmentvocab.CarrierRateMethodFlat,
		BaseRate:            "0",
		AutoRate:            &yes,
		OverrideMarginFloor: &yes,
	}, tenant)
	require.NoError(t, err)

	assert.Equal(t, moveID, req.ShipmentMoveID)
	assert.Equal(t, carrierID, req.CarrierID)
	assert.True(t, req.AutoRate)
	assert.True(t, req.OverrideMarginFloor)
	assert.False(t, req.Replace)
	assert.Equal(t, tenant, req.TenantInfo)
}
