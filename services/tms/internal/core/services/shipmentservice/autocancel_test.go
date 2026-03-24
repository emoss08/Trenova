package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
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

func TestServiceGetAutoCancelableShipments_UsesThreshold(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{
			AutoCancelShipments:          true,
			AutoCancelShipmentsThreshold: ptrInt8(21),
		}, nil).
		Once()

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetAutoCancelableShipments(mock.Anything, mock.MatchedBy(func(req *repositories.GetAutoCancelableShipmentsRequest) bool {
			return req.TenantInfo.OrgID == orgID && req.TenantInfo.BuID == buID
		}), int8(21)).
		Return([]*shipment.Shipment{{ID: pulid.MustNew("shp_")}}, nil).
		Once()

	svc := &service{
		l:           zap.NewNop(),
		repo:        repo,
		controlRepo: controlRepo,
	}

	entities, err := svc.GetAutoCancelableShipments(t.Context(), &repositories.GetAutoCancelableShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	})

	require.NoError(t, err)
	require.Len(t, entities, 1)
}

func TestServiceAutoCancelShipments_SkipsWhenDisabled(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{AutoCancelShipments: false}, nil).
		Once()

	svc := &service{
		l:           zap.NewNop(),
		repo:        mocks.NewMockShipmentRepository(t),
		controlRepo: controlRepo,
		realtime:    mocks.NewMockRealtimeService(t),
	}

	entities, err := svc.AutoCancelShipments(t.Context(), &repositories.AutoCancelShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	}, &services.RequestActor{})

	require.NoError(t, err)
	assert.Empty(t, entities)
}

func TestServiceAutoCancelShipments_PublishesBulkInvalidation(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{
			AutoCancelShipments:          true,
			AutoCancelShipmentsThreshold: ptrInt8(30),
		}, nil).
		Once()

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		AutoCancelShipments(mock.Anything, mock.MatchedBy(func(req *repositories.AutoCancelShipmentsRequest) bool {
			return req.TenantInfo.OrgID == orgID && req.TenantInfo.BuID == buID
		}), int8(30)).
		Return([]*shipment.Shipment{{
			ID:             pulid.MustNew("shp_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         shipment.StatusCanceled,
		}}, nil).
		Once()

	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *services.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" &&
				req.Action == "bulk_canceled" &&
				req.OrganizationID == orgID &&
				req.BusinessUnitID == buID &&
				req.ActorUserID == userID &&
				req.RecordID.IsNil()
		})).
		Return(nil).
		Once()

	svc := &service{
		l:           zap.NewNop(),
		repo:        repo,
		controlRepo: controlRepo,
		realtime:    realtime,
	}

	entities, err := svc.AutoCancelShipments(t.Context(), &repositories.AutoCancelShipmentsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	}, &services.RequestActor{
		PrincipalType: services.PrincipalTypeUser,
		PrincipalID:   userID,
		UserID:        userID,
	})

	require.NoError(t, err)
	require.Len(t, entities, 1)
}

//go:fix inline
func ptrInt8(v int8) *int8 {
	return &v
}
