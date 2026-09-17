package billingqueueservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/agenteventstest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func statusChangeFixture(
	t *testing.T,
	from billingqueue.Status,
) (*service, *agenteventstest.Recorder, *billingqueue.BillingQueueItem, pagination.TenantInfo) {
	t.Helper()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	item := &billingqueue.BillingQueueItem{
		ID:               pulid.MustNew("bqi_"),
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		ShipmentID:       pulid.MustNew("shp_"),
		BillToCustomerID: pulid.MustNew("cust_"),
		Status:           from,
	}
	if from == billingqueue.StatusInReview {
		billerID := pulid.MustNew("usr_")
		item.AssignedBillerID = &billerID
	}

	repo := mocks.NewMockBillingQueueRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *repositories.GetBillingQueueItemByIDRequest) (*billingqueue.BillingQueueItem, error) {
			clone := *item
			return &clone, nil
		}).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*billingqueue.BillingQueueItem")).
		RunAndReturn(func(_ context.Context, entity *billingqueue.BillingQueueItem) (*billingqueue.BillingQueueItem, error) {
			return entity, nil
		}).
		Once()
	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Maybe()

	recorder := &agenteventstest.Recorder{}
	svc := &service{
		l:            zap.NewNop(),
		db:           passthroughDB{},
		repo:         repo,
		auditService: audit,
		realtime:     realtime,
		validator:    testValidator(),
		agentEvents:  recorder,
	}

	return svc, recorder, item, tenantInfo
}

func userActor(tenantInfo pagination.TenantInfo) *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    tenantInfo.UserID,
		UserID:         tenantInfo.UserID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
}

func TestUpdateStatus_PublishesExceptionEvent(t *testing.T) {
	t.Parallel()

	svc, recorder, item, tenantInfo := statusChangeFixture(t, billingqueue.StatusInReview)

	reason := billingqueue.ExceptionMissingDocumentation
	_, err := svc.UpdateStatus(t.Context(), &services.UpdateBillingQueueStatusRequest{
		ItemID:              item.ID,
		NewStatus:           billingqueue.StatusException,
		ExceptionReasonCode: &reason,
		ExceptionNotes:      "Rate confirmation missing",
		TenantInfo:          tenantInfo,
	}, userActor(tenantInfo))

	require.NoError(t, err)
	events := recorder.Published()
	require.Len(t, events, 1)
	require.Equal(t, agent.EventBillingQueueItemException, events[0].Kind)
	require.Equal(t, item.ID, events[0].SubjectID)
	require.Equal(t, tenantInfo.OrgID, events[0].TenantInfo.OrgID)
	require.Equal(t, tenantInfo.BuID, events[0].TenantInfo.BuID)
}

func TestUpdateStatus_PublishesOnHoldEvent(t *testing.T) {
	t.Parallel()

	svc, recorder, item, tenantInfo := statusChangeFixture(t, billingqueue.StatusReadyForReview)

	_, err := svc.UpdateStatus(t.Context(), &services.UpdateBillingQueueStatusRequest{
		ItemID:     item.ID,
		NewStatus:  billingqueue.StatusOnHold,
		TenantInfo: tenantInfo,
	}, userActor(tenantInfo))

	require.NoError(t, err)
	events := recorder.Published()
	require.Len(t, events, 1)
	require.Equal(t, agent.EventBillingQueueItemOnHold, events[0].Kind)
}

func TestUpdateStatus_DoesNotPublishForOtherTransitions(t *testing.T) {
	t.Parallel()

	svc, recorder, item, tenantInfo := statusChangeFixture(t, billingqueue.StatusOnHold)

	_, err := svc.UpdateStatus(t.Context(), &services.UpdateBillingQueueStatusRequest{
		ItemID:     item.ID,
		NewStatus:  billingqueue.StatusReadyForReview,
		TenantInfo: tenantInfo,
	}, userActor(tenantInfo))

	require.NoError(t, err)
	require.Empty(t, recorder.Published())
}
