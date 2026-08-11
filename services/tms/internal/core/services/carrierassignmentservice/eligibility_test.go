package carrierassignmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = int64(86400)

func qualifiedCarrier(now int64) *carrier.Carrier {
	return &carrier.Carrier{
		Status:           carrier.StatusActive,
		ComplianceStatus: carrier.ComplianceStatusQualified,
		InsurancePolicies: []*carrier.CarrierInsurancePolicy{
			{
				PolicyType:     carrier.InsurancePolicyTypeAutoLiability,
				PolicyNumber:   "AUTO-1",
				ExpirationDate: now + 365*day,
			},
			{
				PolicyType:     carrier.InsurancePolicyTypeCargoLiability,
				PolicyNumber:   "CARGO-1",
				ExpirationDate: now + 365*day,
			},
		},
	}
}

func TestEnforceEligibility(t *testing.T) {
	now := timeutils.NowUnix()

	t.Run("blocked carrier returns error regardless of override", func(t *testing.T) {
		entity := qualifiedCarrier(now)
		entity.Status = carrier.StatusInactive
		require.Error(t, enforceEligibility(entity, true))
	})

	t.Run("warnings require override", func(t *testing.T) {
		entity := qualifiedCarrier(now)
		entity.InsurancePolicies[0].ExpirationDate = now + 10*day
		require.Error(t, enforceEligibility(entity, false))
		require.NoError(t, enforceEligibility(entity, true))
	})
}

func TestCarrierCoverageDrivesMoveStatusDerivation(t *testing.T) {
	coordinator := shipmentstate.NewCoordinator()

	buildShipment := func(coverage *shipment.CarrierAssignment) *shipment.Shipment {
		return &shipment.Shipment{
			Status: shipment.StatusNew,
			Moves: []*shipment.ShipmentMove{
				{
					Status:            shipment.MoveStatusNew,
					CoverageType:      shipment.MoveCoverageTypeDriver,
					CarrierAssignment: coverage,
					Stops: []*shipment.Stop{
						{
							Type:                 shipment.StopTypePickup,
							Status:               shipment.StopStatusNew,
							Sequence:             0,
							ScheduledWindowStart: 100,
						},
						{
							Type:                 shipment.StopTypeDelivery,
							Status:               shipment.StopStatusNew,
							Sequence:             1,
							ScheduledWindowStart: 200,
						},
					},
				},
			},
		}
	}

	original := buildShipment(nil)
	updated := buildShipment(&shipment.CarrierAssignment{
		Status: shipment.CarrierAssignmentStatusPending,
	})
	updated.Moves[0].CoverageType = shipment.MoveCoverageTypeCarrier

	multiErr := coordinator.PrepareForUpdate(original, updated)
	require.Nil(t, multiErr)
	assert.Equal(t, shipment.MoveStatusAssigned, updated.Moves[0].Status)

	reverted := buildShipment(nil)
	reverted.Moves[0].Status = shipment.MoveStatusNew
	multiErr = coordinator.PrepareForUpdate(updated, reverted)
	require.Nil(t, multiErr)
	assert.Equal(t, shipment.MoveStatusNew, reverted.Moves[0].Status)

	canceledCoverage := buildShipment(&shipment.CarrierAssignment{
		Status: shipment.CarrierAssignmentStatusCanceled,
	})
	multiErr = coordinator.PrepareForUpdate(original, canceledCoverage)
	require.Nil(t, multiErr)
	assert.Equal(t, shipment.MoveStatusNew, canceledCoverage.Moves[0].Status)
}
