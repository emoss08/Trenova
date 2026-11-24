package carrierresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/carriercoverports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoverEndingRequestBoundsTheWindowAndTheCount(t *testing.T) {
	t.Parallel()

	req, err := coverEndingRequest(1000, nil, nil, true)
	require.NoError(t, err)
	assert.Equal(t, int64(1000+30*86400), req.Until)
	assert.Equal(t, carriercoverports.DefaultCarriers, req.Limit)
	assert.True(t, req.Loads)

	blind, err := coverEndingRequest(1000, nil, nil, false)
	require.NoError(t, err)
	assert.False(t, blind.Loads, "loads are not counted for a reader who may not read shipments")

	for _, days := range []int{0, 91} {
		_, err = coverEndingRequest(1000, &days, nil, true)
		require.Error(t, err, days)
	}
	for _, limit := range []int{0, 101} {
		_, err = coverEndingRequest(1000, nil, &limit, true)
		require.Error(t, err, limit)
	}
}

func TestCoverEndingsWithholdTheLoadCountUnlessEveryLoadIsReadable(t *testing.T) {
	t.Parallel()

	whole := &carriercoverports.Cover{
		CarrierID: pulid.MustNew("car_"),
		Name:      "Ridgeline Freight",
		Policy: &carrier.CarrierInsurancePolicy{
			PolicyType: carrier.InsurancePolicyTypeAutoLiability, ExpirationDate: 5000,
		},
		LoadsRead:     true,
		OpenLoads:     4,
		ReadableLoads: 4,
	}
	partial := &carriercoverports.Cover{
		CarrierID: pulid.MustNew("car_"),
		Policy: &carrier.CarrierInsurancePolicy{
			PolicyType: carrier.InsurancePolicyTypeCargoLiability, ExpirationDate: 6000,
		},
		LoadsRead:     true,
		OpenLoads:     2,
		ReadableLoads: 1,
	}

	models := coverEndingsToModel([]*carriercoverports.Cover{whole, partial}, true)
	require.Len(t, models, 2)
	require.NotNil(t, models[0].OpenLoads)
	assert.Equal(t, 4, *models[0].OpenLoads)
	assert.Equal(t, 5000, models[0].EndsAt)
	assert.Nil(t, models[1].OpenLoads)

	blind := coverEndingsToModel([]*carriercoverports.Cover{whole}, false)
	assert.Nil(t, blind[0].OpenLoads, "a reader who may not read shipments sees no load count")
}
