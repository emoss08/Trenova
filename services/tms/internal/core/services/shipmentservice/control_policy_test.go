package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidatorValidateCreate_RejectsWeightAboveShipmentControlLimit(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	weight := int64(1001)
	entity.Weight = &weight

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{MaxShipmentWeightLimit: 1000}, nil).
		Maybe()

	v := &Validator{
		validator: newValidatorBuilder(
			nil,
			controlRepo,
			NewTestCustomerRepository(t),
			mocks.NewMockCommodityRepository(t),
			mocks.NewMockHazmatSegregationRuleRepository(t),
			mocks.NewMockShipmentRepository(t),
		).Build(),
	}

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "weight")
}

func TestValidatorValidateUpdate_RejectsMoveRemovalWhenDisallowed(t *testing.T) {
	t.Parallel()

	original := validShipmentForValidation()
	original.ID = pulid.MustNew("shp_")
	original.Version = 1
	original.Moves = []*shipment.ShipmentMove{
		validMove(),
		validMove(),
	}
	original.Moves[0].ID = pulid.MustNew("sm_")
	original.Moves[1].ID = pulid.MustNew("sm_")

	entity := cloneShipment(original)
	entity.Moves = []*shipment.ShipmentMove{cloneMoveForControlPolicy(original.Moves[0])}

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{
			AllowMoveRemovals: false,
		}, nil).
		Maybe()

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(cloneShipment(original), nil).
		Once()

	v := &Validator{
		validator: newValidatorBuilder(
			nil,
			controlRepo,
			NewTestCustomerRepository(t),
			mocks.NewMockCommodityRepository(t),
			mocks.NewMockHazmatSegregationRuleRepository(t),
			shipmentRepo,
		).Build(),
	}

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "moves")
}

func cloneMoveForControlPolicy(move *shipment.ShipmentMove) *shipment.ShipmentMove {
	if move == nil {
		return nil
	}

	cloned := *move
	if len(move.Stops) > 0 {
		cloned.Stops = make([]*shipment.Stop, 0, len(move.Stops))
		for _, stop := range move.Stops {
			if stop == nil {
				cloned.Stops = append(cloned.Stops, nil)
				continue
			}

			stopCopy := *stop
			cloned.Stops = append(cloned.Stops, &stopCopy)
		}
	}

	return &cloned
}
