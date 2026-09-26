package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingqueueservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func billingQueueFor(
	t *testing.T,
	item *billingqueue.BillingQueueItem,
	guard *writeGuard,
) *mocks.MockBillingQueueService {
	t.Helper()

	billing := mocks.NewMockBillingQueueService(t)
	billing.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(
			context.Context,
			*repositories.GetBillingQueueItemByIDRequest,
		) (*billingqueue.BillingQueueItem, error) {
			copied := *item

			return &copied, nil
		}).
		Maybe()
	billing.EXPECT().
		UpdateStatus(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *serviceports.UpdateBillingQueueStatusRequest,
			actor *serviceports.RequestActor,
		) (*billingqueue.BillingQueueItem, error) {
			if err := guard.write(); err != nil {
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
	guard := &writeGuard{}
	tool := newTransitionToInReviewTool(billingQueueFor(t, item, guard)).(*transitionToInReviewTool)
	params := executeParams(map[string]any{"billingQueueItemId": item.ID.String()})
	params.Actor.PrincipalType = serviceports.PrincipalTypeUser

	preview := previewWithoutWrites(t, guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceBillingQueue, change.Resource)
	assert.Equal(t, "BQ-1042", change.Label)
	assert.Equal(t, "Exception", fieldByPath(t, change, "status").Before)
	assert.Equal(t, "InReview", fieldByPath(t, change, "status").After)
	assert.True(t, fieldByPath(t, change, "reviewStartedAt").Volatile)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, item,
		toolpreview.Only(inReviewFields...), toolpreview.Volatile(inReviewVolatileFields...))
}

func TestTransitionToInReview_PreviewWarnsWhenAnAgentWouldRunIt(t *testing.T) {
	t.Parallel()

	item := &billingqueue.BillingQueueItem{ID: pulid.MustNew("bqi_"), Status: billingqueue.StatusOnHold}
	guard := &writeGuard{}
	tool := newTransitionToInReviewTool(billingQueueFor(t, item, guard)).(*transitionToInReviewTool)
	params := executeParams(map[string]any{"billingQueueItemId": item.ID.String()})
	params.Actor = agentActorFor(params.OrganizationID, params.BusinessUnitID)

	preview := previewWithoutWrites(t, guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}
