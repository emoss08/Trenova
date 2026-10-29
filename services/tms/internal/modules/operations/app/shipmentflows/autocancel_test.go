package shipmentflows

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServiceGetAutoCancelableShipments_UsesThreshold(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&shipmentcontrol.ShipmentControl{
			AutoCancelShipments:          true,
			AutoCancelShipmentsThreshold: new(int16(21)),
		}, nil).
		Once()

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetAutoCancelableShipments(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetAutoCancelableShipmentsRequest) bool {
			return req.TenantInfo.OrgID == orgID && req.TenantInfo.BuID == buID
		}), int16(21)).
		Return([]*shipment.Shipment{{ID: pulid.MustNew("shp_")}}, nil).
		Once()

	svc := assembled(&rig{
		l:           zap.NewNop(),
		repo:        repo,
		controlRepo: controlRepo,
	})

	entities, err := svc.lifecycle.GetAutoCancelableShipments(
		t.Context(),
		&shipmentports.GetAutoCancelableShipmentsRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		},
	)

	require.NoError(t, err)
	require.Len(t, entities, 1)
}

func TestServiceAutoCancelShipments_SkipsWhenDisabled(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&shipmentcontrol.ShipmentControl{AutoCancelShipments: false}, nil).
		Once()

	svc := assembled(&rig{
		l:           zap.NewNop(),
		repo:        shipmentportsmock.NewMockShipmentRepository(t),
		controlRepo: controlRepo,
		invalidator: newTestInvalidator(realtimeportsmock.NewMockRealtimeService(t)),
	})

	entities, err := svc.lifecycle.AutoCancelShipments(
		t.Context(),
		&shipmentports.AutoCancelShipmentsRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		},
		&permissionports.RequestActor{},
	)

	require.NoError(t, err)
	assert.Empty(t, entities)
}

func TestServiceAutoCancelShipments_PublishesBulkInvalidation(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&shipmentcontrol.ShipmentControl{
			AutoCancelShipments:          true,
			AutoCancelShipmentsThreshold: new(int16(30)),
		}, nil).
		Once()

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		AutoCancelShipments(mock.Anything, mock.MatchedBy(func(req *shipmentports.AutoCancelShipmentsRequest) bool {
			return req.TenantInfo.OrgID == orgID && req.TenantInfo.BuID == buID
		}), int16(30)).
		Return([]*shipment.Shipment{{
			ID:             pulid.MustNew("shp_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         shipment.StatusCanceled,
		}}, nil).
		Once()

	realtime := realtimeportsmock.NewMockRealtimeService(t)
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *realtimeports.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" &&
				req.Action == "bulk_canceled" &&
				req.OrganizationID == orgID &&
				req.BusinessUnitID == buID &&
				req.ActorUserID == userID &&
				req.RecordID.IsNil()
		})).
		Return(nil).
		Once()

	svc := assembled(&rig{
		l:           zap.NewNop(),
		repo:        repo,
		controlRepo: controlRepo,
		invalidator: newTestInvalidator(realtime),
	})

	entities, err := svc.lifecycle.AutoCancelShipments(
		t.Context(),
		&shipmentports.AutoCancelShipmentsRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		},
		&permissionports.RequestActor{
			PrincipalType: permissionports.PrincipalTypeUser,
			PrincipalID:   userID,
			UserID:        userID,
		},
	)

	require.NoError(t, err)
	require.Len(t, entities, 1)
}
