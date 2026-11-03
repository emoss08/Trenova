package shipmentjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentinvalidation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/shipmentjobs/shipmentcontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
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

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	realtime := realtimeportsmock.NewMockRealtimeService(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	sourceID := pulid.MustNew("shp_")
	copyOne := pulid.MustNew("shp_")
	copyTwo := pulid.MustNew("shp_")

	repo.EXPECT().
		BulkDuplicate(mock.Anything, mock.MatchedBy(func(req *shipmentports.BulkDuplicateShipmentRequest) bool {
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
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *realtimeports.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" &&
				req.Action == "bulk_created" &&
				req.OrganizationID == orgID &&
				req.BusinessUnitID == buID &&
				req.ActorUserID == userID &&
				req.ActorType == permissionports.PrincipalTypeUser &&
				req.ActorID == userID &&
				req.RecordID.IsNil() &&
				req.Entity == nil
		})).
		Return(nil).
		Once()

	agentEvents := &recordingAgentEvents{}
	activities := NewActivities(ActivitiesParams{
		Repo:         repo,
		AuditService: audit,
		Invalidator:  shipmentinvalidation.NewWithDependencies(realtime, nil, zap.NewNop()),
		AgentEvents:  agentEvents,
		Logger:       zap.NewNop(),
	})

	result, err := activities.BulkDuplicateShipmentsActivity(
		t.Context(),
		&shipmentcontract.BulkDuplicateShipmentsPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: orgID,
				BusinessUnitID: buID,
				UserID:         userID,
			},
			ShipmentID:    sourceID,
			Count:         2,
			OverrideDates: true,
			RequestedBy:   userID,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.DuplicatedCount)
	assert.Equal(t, []pulid.ID{copyOne, copyTwo}, result.ShipmentIDs)
	assert.Equal(t, sourceID, result.SourceShipmentID)
	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	assert.Equal(t, []agenteventports.AgentEvent{
		{Kind: agenttrigger.EventShipmentCreated, SubjectID: copyOne, TenantInfo: tenant},
		{Kind: agenttrigger.EventShipmentCreated, SubjectID: copyTwo, TenantInfo: tenant},
	}, agentEvents.events)
}

type recordingAgentEvents struct {
	events []agenteventports.AgentEvent
}

func (r *recordingAgentEvents) Publish(_ context.Context, event *agenteventports.AgentEvent) {
	r.events = append(r.events, *event)
}

func tenantWork(orgID, buID pulid.ID) *ShipmentTenantWorkPayload {
	return &ShipmentTenantWorkPayload{
		TenantWorkItem: temporaljobs.TenantWorkItem{OrganizationID: orgID, BusinessUnitID: buID},
	}
}

func TestActivitiesAutoDelayTenantShipmentsActivity_InvalidatesEachDelayedShipment(t *testing.T) {
	t.Parallel()

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	realtime := realtimeportsmock.NewMockRealtimeService(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")
	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}

	repo.EXPECT().
		RunAutoDelayShipmentsForTenant(mock.Anything, tenant, temporaljobs.DefaultTenantRecordLimit).
		Return([]*shipment.Shipment{{
			ID:             shipmentID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         shipmentvocab.StatusDelayed,
		}}, nil).
		Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *realtimeports.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" &&
				req.Action == "delayed" &&
				req.OrganizationID == orgID &&
				req.BusinessUnitID == buID &&
				req.RecordID == shipmentID
		})).
		Return(nil).
		Once()

	activities := NewActivities(ActivitiesParams{
		Repo:         repo,
		AuditService: auditportsmock.NewMockAuditService(t),
		Invalidator:  shipmentinvalidation.NewWithDependencies(realtime, nil, zap.NewNop()),
		Logger:       zap.NewNop(),
	})

	result, err := activities.AutoDelayTenantShipmentsActivity(t.Context(), tenantWork(orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.DelayedCount)
	assert.Equal(t, []pulid.ID{shipmentID}, result.ShipmentIDs)
	assert.Equal(t, 1, result.RecordsProcessed)
}

func TestActivitiesAutoCancelTenantShipmentsActivity_InvalidatesTheTenantOnce(t *testing.T) {
	t.Parallel()

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	realtime := realtimeportsmock.NewMockRealtimeService(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	first := pulid.MustNew("shp_")
	second := pulid.MustNew("shp_")

	repo.EXPECT().
		RunAutoCancelShipmentsForTenant(mock.Anything, tenant, temporaljobs.DefaultTenantRecordLimit).
		Return([]*shipment.Shipment{
			{ID: first, OrganizationID: orgID, BusinessUnitID: buID, Status: shipmentvocab.StatusCanceled},
			{ID: second, OrganizationID: orgID, BusinessUnitID: buID, Status: shipmentvocab.StatusCanceled},
		}, nil).
		Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *realtimeports.PublishResourceInvalidationRequest) bool {
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
		AuditService: auditportsmock.NewMockAuditService(t),
		Invalidator:  shipmentinvalidation.NewWithDependencies(realtime, nil, zap.NewNop()),
		Logger:       zap.NewNop(),
	})

	result, err := activities.AutoCancelTenantShipmentsActivity(t.Context(), tenantWork(orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.CanceledCount)
	assert.Equal(t, []pulid.ID{first, second}, result.ShipmentIDs)
	assert.Equal(t, 2, result.RecordsProcessed)
}

func TestActivitiesAutoCancelTenantShipmentsActivity_NothingCanceledPublishesNothing(t *testing.T) {
	t.Parallel()

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	realtime := realtimeportsmock.NewMockRealtimeService(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	repo.EXPECT().
		RunAutoCancelShipmentsForTenant(
			mock.Anything,
			pagination.TenantInfo{OrgID: orgID, BuID: buID},
			temporaljobs.DefaultTenantRecordLimit,
		).
		Return([]*shipment.Shipment{}, nil).
		Once()

	activities := NewActivities(ActivitiesParams{
		Repo:         repo,
		AuditService: auditportsmock.NewMockAuditService(t),
		Invalidator:  shipmentinvalidation.NewWithDependencies(realtime, nil, zap.NewNop()),
		Logger:       zap.NewNop(),
	})

	result, err := activities.AutoCancelTenantShipmentsActivity(t.Context(), tenantWork(orgID, buID))

	require.NoError(t, err)
	assert.Equal(t, 0, result.CanceledCount)
	assert.Empty(t, result.ShipmentIDs)
}
