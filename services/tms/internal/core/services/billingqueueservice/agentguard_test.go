package billingqueueservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func agentActor(tenantInfo pagination.TenantInfo) *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeAgent,
		PrincipalID:    services.AgentPrincipalID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
}

// Moving an item into review, onto hold, into exception or back to
// operations creates no money and is undone by moving it again, so an agent
// may make it. transition_item_to_in_review was built on this and failed on
// every unattended run while the service refused the agent outright.
func TestUpdateStatus_AnAgentMayMoveAnItemWhereNoMoneyIsMade(t *testing.T) {
	t.Parallel()

	reason := billingqueue.ExceptionMissingDocumentation
	cases := []struct {
		name string
		from billingqueue.Status
		req  services.UpdateBillingQueueStatusRequest
	}{
		{
			name: "into review",
			from: billingqueue.StatusOnHold,
			req:  services.UpdateBillingQueueStatusRequest{NewStatus: billingqueue.StatusInReview},
		},
		{
			name: "on hold",
			from: billingqueue.StatusReadyForReview,
			req:  services.UpdateBillingQueueStatusRequest{NewStatus: billingqueue.StatusOnHold},
		},
		{
			name: "into exception",
			from: billingqueue.StatusInReview,
			req: services.UpdateBillingQueueStatusRequest{
				NewStatus:           billingqueue.StatusException,
				ExceptionReasonCode: &reason,
				ExceptionNotes:      "The POD is missing",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, _, item, tenantInfo := statusChangeFixture(t, tc.from)
			if item.AssignedBillerID == nil {
				billerID := pulid.MustNew("usr_")
				item.AssignedBillerID = &billerID
			}
			req := tc.req
			req.ItemID = item.ID
			req.TenantInfo = tenantInfo

			updated, err := svc.UpdateStatus(t.Context(), &req, agentActor(tenantInfo))

			require.NoError(t, err)
			assert.Equal(t, tc.req.NewStatus, updated.Status)
		})
	}
}

// Approving creates an invoice, canceling drops the charge, posting books
// revenue and returning an item to review re-opens what a person closed: each
// stays a person's decision, and the service says so in the same words as
// before.
func TestUpdateStatus_AnAgentStillCannotMakeAMoneyDecision(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	for _, status := range []billingqueue.Status{
		billingqueue.StatusApproved,
		billingqueue.StatusCanceled,
		billingqueue.StatusPosted,
		billingqueue.StatusReadyForReview,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			item, err := svc.UpdateStatus(t.Context(), &services.UpdateBillingQueueStatusRequest{
				ItemID:     pulid.MustNew("bqi_"),
				NewStatus:  status,
				TenantInfo: tenantInfo,
			}, agentActor(tenantInfo))

			require.Nil(t, item)
			var validation *errortypes.Error
			require.ErrorAs(t, err, &validation)
			assert.Equal(t, errortypes.ErrForbidden, validation.Code)
			assert.Equal(
				t,
				"Agent principals cannot transition billing queue items; a human must decide",
				validation.Error(),
			)
		})
	}
}

func TestPlanStatusChange_FollowsTheSameRuleForAnAgent(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	actor := agentActor(tenantInfo)

	held := &billingqueue.BillingQueueItem{Status: billingqueue.StatusReadyForReview}
	require.NoError(t, PlanStatusChange(held, &services.UpdateBillingQueueStatusRequest{
		NewStatus:   billingqueue.StatusOnHold,
		ReviewNotes: "Waiting on the lumper receipt",
	}, actor, 100))
	assert.Equal(t, billingqueue.StatusOnHold, held.Status)
	assert.Equal(t, "Waiting on the lumper receipt", held.ReviewNotes)

	approved := &billingqueue.BillingQueueItem{Status: billingqueue.StatusInReview}
	require.Error(t, PlanStatusChange(approved, &services.UpdateBillingQueueStatusRequest{
		NewStatus: billingqueue.StatusApproved,
	}, actor, 100))
	assert.Equal(t, billingqueue.StatusInReview, approved.Status, "a refused plan changes nothing")
}

// PlanTransition is the change itself, for a preview of a decision only a
// person makes: it is what the approver's write will do.
func TestPlanTransition_PlansAPersonsDecision(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	item := &billingqueue.BillingQueueItem{Status: billingqueue.StatusInReview}
	require.NoError(t, PlanTransition(item, &services.UpdateBillingQueueStatusRequest{
		NewStatus:    billingqueue.StatusCanceled,
		CancelReason: "Duplicate of BQ-10",
	}, &services.RequestActor{UserID: userID}, 100))

	assert.Equal(t, billingqueue.StatusCanceled, item.Status)
	assert.Equal(t, "Duplicate of BQ-10", item.CancelReason)
	require.NotNil(t, item.CanceledByID)
	assert.Equal(t, userID, *item.CanceledByID)
}

func TestCheckStatusFields_NamesWhatEachStatusRequires(t *testing.T) {
	t.Parallel()

	other := billingqueue.ExceptionOther
	cases := []struct {
		name   string
		item   billingqueue.BillingQueueItem
		fields []string
	}{
		{
			name:   "review needs a biller",
			item:   billingqueue.BillingQueueItem{Status: billingqueue.StatusInReview},
			fields: []string{"assignedBillerId"},
		},
		{
			name:   "exception needs a reason and notes",
			item:   billingqueue.BillingQueueItem{Status: billingqueue.StatusException},
			fields: []string{"exceptionReasonCode", "exceptionNotes"},
		},
		{
			name: "sent back for another reason needs notes",
			item: billingqueue.BillingQueueItem{
				Status:              billingqueue.StatusSentBackToOps,
				ExceptionReasonCode: &other,
			},
			fields: []string{"exceptionNotes"},
		},
		{
			name:   "cancel needs who and why",
			item:   billingqueue.BillingQueueItem{Status: billingqueue.StatusCanceled},
			fields: []string{"canceledById", "cancelReason"},
		},
		{
			name: "hold needs nothing",
			item: billingqueue.BillingQueueItem{Status: billingqueue.StatusOnHold},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			multiErr := errortypes.NewMultiError()
			CheckStatusFields(&tc.item, multiErr)

			fields := make([]string, 0, len(multiErr.Errors))
			for _, entry := range multiErr.Errors {
				fields = append(fields, entry.Field)
			}
			assert.ElementsMatch(t, tc.fields, fields)
		})
	}
}

func TestSendBackComment_IsTheOperationsNoteASendBackPosts(t *testing.T) {
	t.Parallel()

	reason := billingqueue.ExceptionIncorrectRates
	item := &billingqueue.BillingQueueItem{
		OrganizationID:      pulid.MustNew("org_"),
		BusinessUnitID:      pulid.MustNew("bu_"),
		ShipmentID:          pulid.MustNew("shp_"),
		ExceptionReasonCode: &reason,
		ExceptionNotes:      "Linehaul is 200 under the agreement",
	}
	userID := pulid.MustNew("usr_")

	comment := SendBackComment(item, userID)

	assert.Equal(t, item.ShipmentID, comment.ShipmentID)
	assert.Equal(t, userID, comment.UserID)
	assert.Equal(
		t,
		"Sent back from billing: IncorrectRates\n\nLinehaul is 200 under the agreement",
		comment.Comment,
	)
}
