package agenttoolservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTransitionToInReview_WaitsForAPersonWhileTheItemIsOnHold(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status billingqueue.Status
		want   agent.AutonomyTier
	}{
		{name: "on hold", status: billingqueue.StatusOnHold, want: agent.TierPropose},
		{name: "exception", status: billingqueue.StatusException, want: agent.TierAutoExecute},
		{
			name:   "ready for review",
			status: billingqueue.StatusReadyForReview,
			want:   agent.TierAutoExecute,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			itemID := pulid.MustNew("bqi_")
			params := deskParams(map[string]any{"billingQueueItemId": itemID.String()})
			billing := mocks.NewMockBillingQueueService(t)
			billing.EXPECT().
				GetByID(mock.Anything, mock.MatchedBy(
					func(req *repositories.GetBillingQueueItemByIDRequest) bool {
						return req.ItemID == itemID &&
							req.TenantInfo.OrgID == params.OrganizationID &&
							req.TenantInfo.BuID == params.BusinessUnitID
					},
				)).
				Return(&billingqueue.BillingQueueItem{ID: itemID, Status: tc.status}, nil).
				Once()

			condition := newTransitionToInReviewTool(billing).Policy().Condition
			require.NotNil(t, condition)
			assert.Equal(t, tc.want, condition.Limit(t.Context(), params))
		})
	}
}

func TestTransitionToInReview_AnItemThatCannotBeReadIsAProposal(t *testing.T) {
	t.Parallel()

	billing := mocks.NewMockBillingQueueService(t)
	billing.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(nil, errors.New("item not found")).
		Once()

	condition := newTransitionToInReviewTool(billing).Policy().Condition
	params := deskParams(map[string]any{"billingQueueItemId": pulid.MustNew("bqi_").String()})
	assert.Equal(t, agent.TierPropose, condition.Limit(t.Context(), params))

	missing := deskParams(map[string]any{})
	assert.Equal(t, agent.TierPropose, condition.Limit(t.Context(), missing))
}

func TestTransitionToInReview_AnAutomaticAgentStillProposesOffHold(t *testing.T) {
	t.Parallel()

	itemID := pulid.MustNew("bqi_")
	params := deskParams(map[string]any{"billingQueueItemId": itemID.String()})
	billing := mocks.NewMockBillingQueueService(t)
	billing.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&billingqueue.BillingQueueItem{ID: itemID, Status: billingqueue.StatusOnHold}, nil).
		Once()

	tool := newTransitionToInReviewTool(billing)
	decision := agenttoolpolicy.Decide(t.Context(), agenttoolpolicy.DecideInput{
		Policy: tool.Policy(),
		Params: params,
		Definition: &agentdefinition.Definition{
			AutonomyCeiling: agent.TierAutoExecute,
			ToolTiers:       map[string]agent.AutonomyTier{tool.Name(): agent.TierAutoExecute},
		},
		Unattended: true,
		Taint:      &agent.RunTaint{},
	})

	assert.Equal(t, agent.TierPropose, decision.Tier)
	assert.Contains(t, decision.HeldBy, agenttoolpolicy.HeldByCondition)
}
