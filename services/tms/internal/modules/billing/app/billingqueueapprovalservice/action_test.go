package billingqueueapprovalservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingapprovalrun"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldsOf(t *testing.T, err error) []string {
	t.Helper()
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}

	return fields
}

func TestEachBulkActionAsksForWhatItNeeds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action billingapprovalrun.RunAction
		params billingapprovalrun.RunParams
		fields []string
	}{
		{
			"assign names a biller",
			billingapprovalrun.RunActionAssign,
			billingapprovalrun.RunParams{},
			[]string{"params.billerId"},
		},
		{
			"hold gives a known reason",
			billingapprovalrun.RunActionHold,
			billingapprovalrun.RunParams{HoldReasonCode: "Lunch"},
			[]string{"params.holdReasonCode"},
		},
		{
			"cancel says why",
			billingapprovalrun.RunActionCancel,
			billingapprovalrun.RunParams{Reason: "   "},
			[]string{"params.reason"},
		},
		{
			"settling names the check and option",
			billingapprovalrun.RunActionResolveIssue,
			billingapprovalrun.RunParams{},
			[]string{"params.issueCode", "params.optionKey"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateRunAction(&billingqueueports.StartBillingQueueApprovalRequest{
				Action: tt.action,
				Params: tt.params,
			})
			assert.Equal(t, tt.fields, fieldsOf(t, err))
		})
	}
}

func TestBulkActionsThatNeedNothingMorePass(t *testing.T) {
	t.Parallel()

	for _, action := range []billingapprovalrun.RunAction{
		billingapprovalrun.RunActionApprove,
		billingapprovalrun.RunActionPost,
		billingapprovalrun.RunActionRequestPaperwork,
	} {
		assert.NoError(
			t,
			validateRunAction(&billingqueueports.StartBillingQueueApprovalRequest{Action: action}),
			action,
		)
	}
	assert.NoError(t, validateRunAction(&billingqueueports.StartBillingQueueApprovalRequest{
		Action: billingapprovalrun.RunActionAssign,
		Params: billingapprovalrun.RunParams{BillerID: pulid.MustNew("usr_")},
	}))
}

func TestAnUnknownBulkActionIsRefused(t *testing.T) {
	t.Parallel()

	err := validateRunAction(&billingqueueports.StartBillingQueueApprovalRequest{Action: "Archive"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown bulk action")
}

func TestPostingStartsAtOnceAndEverythingElseWaitsForUndo(t *testing.T) {
	t.Parallel()

	assert.False(t, billingapprovalrun.RunActionPost.Undoable())
	before := commitAt(billingapprovalrun.RunActionPost)
	held := commitAt(billingapprovalrun.RunActionHold)
	assert.InDelta(t, billingapprovalrun.ApprovalUndoWindowSeconds, held-before, 1)
}
