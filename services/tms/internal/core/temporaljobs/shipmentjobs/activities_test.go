package shipmentjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestActivitiesBulkDuplicateShipmentsActivity(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	sourceID := pulid.MustNew("shp_")
	copyOne := pulid.MustNew("shp_")
	copyTwo := pulid.MustNew("shp_")

	repo.EXPECT().
		BulkDuplicate(mock.Anything, mock.MatchedBy(func(req *repositories.BulkDuplicateShipmentRequest) bool {
			return req.ShipmentID == sourceID &&
				req.Count == 2 &&
				req.OverrideDates &&
				req.TenantInfo == (pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID})
		})).
		Return([]*shipment.Shipment{
			{ID: copyOne, OrganizationID: orgID, BusinessUnitID: buID},
			{ID: copyTwo, OrganizationID: orgID, BusinessUnitID: buID},
		}, nil).
		Once()

	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Twice()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *services.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" &&
				req.Action == "bulk_created" &&
				req.OrganizationID == orgID &&
				req.BusinessUnitID == buID &&
				req.ActorUserID == userID &&
				req.ActorType == services.PrincipalTypeUser &&
				req.ActorID == userID &&
				req.RecordID.IsNil() &&
				req.Entity == nil
		})).
		Return(nil).
		Once()

	activities := NewActivities(ActivitiesParams{
		Repo:         repo,
		AuditService: audit,
		Realtime:     realtime,
		Logger:       zap.NewNop(),
	})

	result, err := activities.BulkDuplicateShipmentsActivity(t.Context(), &BulkDuplicateShipmentsPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			UserID:         userID,
		},
		ShipmentID:    sourceID,
		Count:         2,
		OverrideDates: true,
		RequestedBy:   userID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.DuplicatedCount)
	assert.Equal(t, []pulid.ID{copyOne, copyTwo}, result.ShipmentIDs)
	assert.Equal(t, sourceID, result.SourceShipmentID)
}

func TestActivitiesAutoDelayShipmentsActivity(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	realtime := mocks.NewMockRealtimeService(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")

	repo.EXPECT().
		AutoDelayShipments(mock.Anything).
		Return([]*shipment.Shipment{{
			ID:             shipmentID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         shipment.StatusDelayed,
		}}, nil).
		Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *services.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" &&
				req.Action == "delayed" &&
				req.RecordID == shipmentID
		})).
		Return(nil).
		Once()

	activities := NewActivities(ActivitiesParams{
		Repo:         repo,
		AuditService: mocks.NewMockAuditService(t),
		Realtime:     realtime,
		Logger:       zap.NewNop(),
	})

	result, err := activities.AutoDelayShipmentsActivity(t.Context())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.DelayedCount)
	assert.Equal(t, []pulid.ID{shipmentID}, result.ShipmentIDs)
}

func TestActivitiesAutoCancelShipmentsActivity(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	realtime := mocks.NewMockRealtimeService(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")

	repo.EXPECT().
		RunAutoCancelShipments(mock.Anything).
		Return([]*shipment.Shipment{{
			ID:             shipmentID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         shipment.StatusCanceled,
		}}, nil).
		Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *services.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" &&
				req.Action == "bulk_canceled" &&
				req.OrganizationID == orgID &&
				req.BusinessUnitID == buID &&
				req.RecordID.IsNil() &&
				req.Entity == nil
		})).
		Return(nil).
		Once()

	activities := NewActivities(ActivitiesParams{
		Repo:         repo,
		AuditService: mocks.NewMockAuditService(t),
		Realtime:     realtime,
		Logger:       zap.NewNop(),
	})

	result, err := activities.AutoCancelShipmentsActivity(t.Context())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.CanceledCount)
	assert.Equal(t, []pulid.ID{shipmentID}, result.ShipmentIDs)
}

func TestActivitiesAutoCancelShipmentsActivity_PublishesPerAffectedTenant(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	realtime := mocks.NewMockRealtimeService(t)

	orgOne := pulid.MustNew("org_")
	buOne := pulid.MustNew("bu_")
	orgTwo := pulid.MustNew("org_")
	buTwo := pulid.MustNew("bu_")

	repo.EXPECT().
		RunAutoCancelShipments(mock.Anything).
		Return([]*shipment.Shipment{
			{ID: pulid.MustNew("shp_"), OrganizationID: orgOne, BusinessUnitID: buOne, Status: shipment.StatusCanceled},
			{ID: pulid.MustNew("shp_"), OrganizationID: orgOne, BusinessUnitID: buOne, Status: shipment.StatusCanceled},
			{ID: pulid.MustNew("shp_"), OrganizationID: orgTwo, BusinessUnitID: buTwo, Status: shipment.StatusCanceled},
		}, nil).
		Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *services.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" &&
				req.Action == "bulk_canceled" &&
				req.OrganizationID == orgOne &&
				req.BusinessUnitID == buOne &&
				req.RecordID.IsNil()
		})).
		Return(nil).
		Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *services.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" &&
				req.Action == "bulk_canceled" &&
				req.OrganizationID == orgTwo &&
				req.BusinessUnitID == buTwo &&
				req.RecordID.IsNil()
		})).
		Return(nil).
		Once()

	activities := NewActivities(ActivitiesParams{
		Repo:         repo,
		AuditService: mocks.NewMockAuditService(t),
		Realtime:     realtime,
		Logger:       zap.NewNop(),
	})

	result, err := activities.AutoCancelShipmentsActivity(t.Context())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 3, result.CanceledCount)
	require.Len(t, result.ShipmentIDs, 3)
}
