package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServiceGetDelayedShipments_UsesThreshold(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{
			AutoDelayShipments:          true,
			AutoDelayShipmentsThreshold: ptrInt16(45),
		}, nil).
		Once()

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetDelayedShipments(mock.Anything, mock.MatchedBy(func(req *repositories.GetDelayedShipmentsRequest) bool {
			return req.TenantInfo.OrgID == orgID && req.TenantInfo.BuID == buID
		}), int16(45)).
		Return([]*shipment.Shipment{{ID: pulid.MustNew("shp_")}}, nil).
		Once()

	svc := &service{
		l:           zap.NewNop(),
		repo:        repo,
		controlRepo: controlRepo,
	}

	entities, err := svc.GetDelayedShipments(t.Context(), &repositories.GetDelayedShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	})

	require.NoError(t, err)
	require.Len(t, entities, 1)
}

func TestServiceDelayShipments_SkipsWhenAutoDelayDisabled(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{AutoDelayShipments: false}, nil).
		Once()

	svc := &service{
		l:           zap.NewNop(),
		repo:        mocks.NewMockShipmentRepository(t),
		controlRepo: controlRepo,
		realtime:    mocks.NewMockRealtimeService(t),
	}

	entities, err := svc.DelayShipments(t.Context(), &repositories.DelayShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	}, &services.RequestActor{})

	require.NoError(t, err)
	assert.Empty(t, entities)
}

func TestServiceDelayShipments_PublishesRealtimeInvalidations(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	shipmentID := pulid.MustNew("shp_")

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{
			AutoDelayShipments:          true,
			AutoDelayShipmentsThreshold: ptrInt16(15),
		}, nil).
		Once()

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		DelayShipments(mock.Anything, mock.MatchedBy(func(req *repositories.DelayShipmentsRequest) bool {
			return req.TenantInfo.OrgID == orgID && req.TenantInfo.BuID == buID
		}), int16(15)).
		Return([]*shipment.Shipment{{
			ID:             shipmentID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         shipment.StatusDelayed,
		}}, nil).
		Once()

	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *services.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" &&
				req.Action == "delayed" &&
				req.RecordID == shipmentID &&
				req.ActorUserID == userID
		})).
		Return(nil).
		Once()

	svc := &service{
		l:           zap.NewNop(),
		repo:        repo,
		controlRepo: controlRepo,
		realtime:    realtime,
	}

	entities, err := svc.DelayShipments(t.Context(), &repositories.DelayShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	}, &services.RequestActor{
		PrincipalType: services.PrincipalTypeUser,
		PrincipalID:   userID,
		UserID:        userID,
	})

	require.NoError(t, err)
	require.Len(t, entities, 1)
}

func TestDelayThresholdMinutes_DisablesAutomaticDelayWhenToggleOff(t *testing.T) {
	t.Parallel()

	assert.Equal(t, shipmentstate.DisabledDelayThresholdMinutes, delayThresholdMinutes(nil))
	assert.Equal(t, shipmentstate.DisabledDelayThresholdMinutes, delayThresholdMinutes(&tenant.ShipmentControl{
		AutoDelayShipments:          false,
		AutoDelayShipmentsThreshold: ptrInt16(15),
	}))
	assert.Equal(t, int16(15), delayThresholdMinutes(&tenant.ShipmentControl{
		AutoDelayShipments:          true,
		AutoDelayShipmentsThreshold: ptrInt16(15),
	}))
}

//go:fix inline
func ptrInt16(v int16) *int16 {
	return &v
}
