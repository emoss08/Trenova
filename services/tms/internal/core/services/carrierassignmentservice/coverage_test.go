package carrierassignmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmentservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestApplyCoverageChangeStampsCoverageExplicitly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initial    shipment.MoveCoverageType
		assignment *shipment.CarrierAssignment
		want       shipment.MoveCoverageType
	}{
		{
			name:       "assigning a carrier marks the move carrier covered",
			initial:    shipment.MoveCoverageTypeUnassigned,
			assignment: &shipment.CarrierAssignment{Status: shipment.CarrierAssignmentStatusPending},
			want:       shipment.MoveCoverageTypeCarrier,
		},
		{
			name:    "canceling a carrier returns the move to uncovered, not driver covered",
			initial: shipment.MoveCoverageTypeCarrier,
			want:    shipment.MoveCoverageTypeUnassigned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			moveID := pulid.MustNew("sm_")
			shipmentID := pulid.MustNew("shp_")
			tenantInfo := pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			}

			original := coverageShipment(shipmentID, moveID, tenantInfo)
			original.Moves[0].CoverageType = tt.initial

			var persisted *shipment.Shipment
			shipmentRepo := mocks.NewMockShipmentRepository(t)
			shipmentRepo.EXPECT().
				Update(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
				RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
					persisted = entity
					return entity, nil
				}).
				Once()

			controlRepo := mocks.NewMockShipmentControlRepository(t)
			controlRepo.EXPECT().
				Get(mock.Anything, repositories.GetShipmentControlRequest{TenantInfo: tenantInfo}).
				Return(&tenant.ShipmentControl{}, nil).
				Once()

			svc := &Service{
				l:                 zap.NewNop(),
				shipmentRepo:      shipmentRepo,
				controlRepo:       controlRepo,
				shipmentValidator: shipmentservice.NewTestValidator(t),
				coordinator:       shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
			}

			require.NoError(
				t,
				svc.applyCoverageChange(t.Context(), tenantInfo, original, moveID, tt.assignment),
			)
			require.NotNil(t, persisted)
			assert.Equal(t, tt.want, persisted.Moves[0].CoverageType)
			assert.Equal(t, tt.initial, original.Moves[0].CoverageType)
		})
	}
}

func coverageShipment(
	shipmentID, moveID pulid.ID,
	tenantInfo pagination.TenantInfo,
) *shipment.Shipment {
	return &shipment.Shipment{
		ID:                shipmentID,
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		ServiceTypeID:     pulid.MustNew("svc_"),
		CustomerID:        pulid.MustNew("cust_"),
		BOL:               "BOL-1",
		FormulaTemplateID: pulid.MustNew("ft_"),
		Status:            shipment.StatusNew,
		Moves: []*shipment.ShipmentMove{
			{
				ID:             moveID,
				ShipmentID:     shipmentID,
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
				Status:         shipment.MoveStatusNew,
				Loaded:         true,
				Stops: []*shipment.Stop{
					{
						ID:                   pulid.MustNew("stp_"),
						ShipmentMoveID:       moveID,
						OrganizationID:       tenantInfo.OrgID,
						BusinessUnitID:       tenantInfo.BuID,
						Sequence:             0,
						Status:               shipment.StopStatusNew,
						Type:                 shipment.StopTypePickup,
						LocationID:           pulid.MustNew("loc_"),
						ScheduleType:         shipment.StopScheduleTypeOpen,
						ScheduledWindowStart: 1,
						ScheduledWindowEnd:   new(int64(2)),
					},
					{
						ID:                   pulid.MustNew("stp_"),
						ShipmentMoveID:       moveID,
						OrganizationID:       tenantInfo.OrgID,
						BusinessUnitID:       tenantInfo.BuID,
						Sequence:             1,
						Status:               shipment.StopStatusNew,
						Type:                 shipment.StopTypeDelivery,
						LocationID:           pulid.MustNew("loc_"),
						ScheduleType:         shipment.StopScheduleTypeOpen,
						ScheduledWindowStart: 3,
						ScheduledWindowEnd:   new(int64(4)),
					},
				},
			},
		},
	}
}
