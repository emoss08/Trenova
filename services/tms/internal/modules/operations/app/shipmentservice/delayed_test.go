package shipmentservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/agentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
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
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{
			AutoDelayShipments:          true,
			AutoDelayShipmentsThreshold: new(int16(45)),
		}, nil).
		Once()

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetDelayedShipments(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetDelayedShipmentsRequest) bool {
			return req.TenantInfo.OrgID == orgID && req.TenantInfo.BuID == buID
		}), int16(45)).
		Return([]*shipment.Shipment{{ID: pulid.MustNew("shp_")}}, nil).
		Once()

	svc := &service{
		l:           zap.NewNop(),
		repo:        repo,
		controlRepo: controlRepo,
	}

	entities, err := svc.GetDelayedShipments(t.Context(), &shipmentports.GetDelayedShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	})

	require.NoError(t, err)
	require.Len(t, entities, 1)
}

func TestServiceDelayShipments_SkipsWhenAutoDelayDisabled(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{AutoDelayShipments: false}, nil).
		Once()

	svc := &service{
		l:           zap.NewNop(),
		repo:        shipmentportsmock.NewMockShipmentRepository(t),
		controlRepo: controlRepo,
		invalidator: newTestInvalidator(agentportsmock.NewMockRealtimeService(t)),
	}

	entities, err := svc.DelayShipments(t.Context(), &shipmentports.DelayShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	}, &permissionports.RequestActor{})

	require.NoError(t, err)
	assert.Empty(t, entities)
}

func TestServiceDelayShipments_PublishesRealtimeInvalidations(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	shipmentID := pulid.MustNew("shp_")

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{
			AutoDelayShipments:          true,
			AutoDelayShipmentsThreshold: new(int16(15)),
		}, nil).
		Once()

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		DelayShipments(mock.Anything, mock.MatchedBy(func(req *shipmentports.DelayShipmentsRequest) bool {
			return req.TenantInfo.OrgID == orgID && req.TenantInfo.BuID == buID
		}), int16(15)).
		Return([]*shipment.Shipment{{
			ID:             shipmentID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         shipment.StatusDelayed,
		}}, nil).
		Once()

	realtime := agentportsmock.NewMockRealtimeService(t)
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *agentports.PublishResourceInvalidationRequest) bool {
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
		invalidator: newTestInvalidator(realtime),
	}

	entities, err := svc.DelayShipments(t.Context(), &shipmentports.DelayShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	}, &permissionports.RequestActor{
		PrincipalType: permissionports.PrincipalTypeUser,
		PrincipalID:   userID,
		UserID:        userID,
	})

	require.NoError(t, err)
	require.Len(t, entities, 1)
}

func TestDelayThresholdMinutes_DisablesAutomaticDelayWhenToggleOff(t *testing.T) {
	t.Parallel()

	assert.Equal(t, shipmentstate.DisabledDelayThresholdMinutes, delayThresholdMinutes(nil))
	assert.Equal(
		t,
		shipmentstate.DisabledDelayThresholdMinutes,
		delayThresholdMinutes(&tenant.ShipmentControl{
			AutoDelayShipments:          false,
			AutoDelayShipmentsThreshold: new(int16(15)),
		}),
	)
	assert.Equal(t, int16(15), delayThresholdMinutes(&tenant.ShipmentControl{
		AutoDelayShipments:          true,
		AutoDelayShipmentsThreshold: new(int16(15)),
	}))
}

//go:fix inline
//go:fix inline
func ptrInt16(v int16) *int16 {
	return new(v)
}
