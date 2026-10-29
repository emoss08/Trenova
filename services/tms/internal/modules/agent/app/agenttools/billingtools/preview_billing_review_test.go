package billingtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingqueuepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func billingQueueFor(
	t *testing.T,
	item *billingqueue.BillingQueueItem,
	guard *agenttoolstest.WriteGuard,
) *billingqueueportsmock.MockBillingQueueService {
	t.Helper()

	billing := billingqueueportsmock.NewMockBillingQueueService(t)
	billing.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(
			context.Context,
			*billingqueueports.GetBillingQueueItemByIDRequest,
		) (*billingqueue.BillingQueueItem, error) {
			copied := *item

			return &copied, nil
		}).
		Maybe()
	billing.EXPECT().
		UpdateStatus(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *billingqueueports.UpdateBillingQueueStatusRequest,
			actor *permissionports.RequestActor,
		) (*billingqueue.BillingQueueItem, error) {
			if err := guard.Write(); err != nil {
				return nil, err
			}
			if err := billingqueueservice.PlanStatusChange(
				item, req, actor, timeutils.NowUnix(),
			); err != nil {
				return nil, err
			}

			return item, nil
		}).
		Maybe()
	billing.EXPECT().
		AssignBiller(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *billingqueueports.AssignBillerRequest,
			_ *permissionports.RequestActor,
		) (*billingqueue.BillingQueueItem, error) {
			if err := guard.Write(); err != nil {
				return nil, err
			}
			if err := billingqueueservice.PlanAssignBiller(
				item, req.BillerID, timeutils.NowUnix(),
			); err != nil {
				return nil, err
			}

			return item, nil
		}).
		Maybe()

	return billing
}

func TestTransitionToInReview_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	item := &billingqueue.BillingQueueItem{
		ID:      pulid.MustNew("bqi_"),
		Number:  "BQ-1042",
		Status:  billingqueue.StatusException,
		Version: 6,
	}
	before := *item
	guard := &agenttoolstest.WriteGuard{}
	tool := NewTransitionToInReviewTool(billingQueueFor(t, item, guard)).(*TransitionToInReviewTool)
	params := agenttoolstest.ExecuteParams(map[string]any{"billingQueueItemId": item.ID.String()})
	params.Actor.PrincipalType = permissionports.PrincipalTypeUser

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceBillingQueue, change.Resource)
	assert.Equal(t, "BQ-1042", change.Label)
	assert.Equal(t, "Exception", agenttoolstest.FieldByPath(t, change, "status").Before)
	assert.Equal(t, "InReview", agenttoolstest.FieldByPath(t, change, "status").After)
	assert.True(t, agenttoolstest.FieldByPath(t, change, "reviewStartedAt").Volatile)
	biller := agenttoolstest.FieldByPath(t, change, "assignedBillerId")
	require.NotNil(t, biller.AfterRef, "an item with nobody on it goes to the person asking")
	assert.Equal(t, params.Actor.UserID, biller.AfterRef.ID)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, item,
		toolpreview.Only(inReviewFields...), toolpreview.Volatile(inReviewVolatileFields...),
		toolpreview.WithRefs(billingqueuepreview.DecisionRefs))
}

// An unattended run is an agent principal, and moving an item into review is
// a move an agent may make: the preview shows it and the write lands. Both
// used to be refused, so the tool failed on every run it could run on alone.
func TestTransitionToInReview_AnAgentMovesTheItemIntoReview(t *testing.T) {
	t.Parallel()

	billerID := pulid.MustNew("usr_")
	item := &billingqueue.BillingQueueItem{
		ID:               pulid.MustNew("bqi_"),
		Number:           "BQ-2001",
		Status:           billingqueue.StatusException,
		AssignedBillerID: &billerID,
	}
	guard := &agenttoolstest.WriteGuard{}
	tool := NewTransitionToInReviewTool(billingQueueFor(t, item, guard)).(*TransitionToInReviewTool)
	params := agenttoolstest.ExecuteParams(map[string]any{"billingQueueItemId": item.ID.String()})
	params.Actor = agenttoolstest.AgentActorFor(params.OrganizationID, params.BusinessUnitID)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	assert.Empty(t, preview.Warnings)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "InReview", agenttoolstest.FieldByPath(t, change, "status").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, billingqueue.StatusInReview, item.Status)
}
