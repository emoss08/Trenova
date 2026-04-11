package manualjournal

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestRequestSyncTotalsAndBalance(t *testing.T) {
	t.Parallel()

	entity := &Request{
		Lines: []*Line{
			{DebitAmount: 1500},
			{CreditAmount: 1200},
			{CreditAmount: 300},
		},
	}

	entity.SyncTotals()

	assert.Equal(t, int64(1500), entity.TotalDebit)
	assert.Equal(t, int64(1500), entity.TotalCredit)
	assert.True(t, entity.IsBalanced())
	assert.Equal(t, 1, entity.Lines[0].LineNumber)
	assert.Equal(t, 3, entity.Lines[2].LineNumber)
}

func TestStatusCanCancel(t *testing.T) {
	t.Parallel()

	assert.True(t, StatusDraft.CanCancel())
	assert.True(t, StatusPendingApproval.CanCancel())
	assert.True(t, StatusApproved.CanCancel())
	assert.False(t, StatusRejected.CanCancel())
	assert.False(t, StatusPosted.CanCancel())
}

func TestStatusHelpers(t *testing.T) {
	t.Parallel()

	assert.True(t, StatusDraft.IsEditable())
	assert.True(t, StatusDraft.CanSubmit())
	assert.False(t, StatusDraft.CanApprove())
	assert.False(t, StatusDraft.CanReject())
	assert.True(t, StatusPendingApproval.CanApprove())
	assert.True(t, StatusPendingApproval.CanReject())
	assert.False(t, StatusApproved.CanSubmit())
}

func TestRequestValidateCollectsErrors(t *testing.T) {
	t.Parallel()

	entity := &Request{
		Lines: []*Line{{Description: "invalid", DebitAmount: 10, CreditAmount: 10}},
	}
	multiErr := errortypes.NewMultiError()

	entity.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "cannot be blank")
	assert.Contains(t, multiErr.Error(), "Exactly one of debit or credit amount must be greater than zero")
}

func TestLineValidateAcceptsSingleSidedAmount(t *testing.T) {
	t.Parallel()

	line := &Line{GLAccountID: pulid.MustNew("gla_"), Description: "Debit", DebitAmount: 25}
	multiErr := errortypes.NewMultiError()

	line.Validate(multiErr)

	require.False(t, multiErr.HasErrors())
}

func TestBeforeAppendModelSetsIDsAndTimestamps(t *testing.T) {
	t.Parallel()

	request := &Request{}
	require.NoError(t, request.BeforeAppendModel(t.Context(), &bun.InsertQuery{}))
	assert.True(t, request.ID.IsNotNil())
	assert.NotZero(t, request.CreatedAt)

	line := &Line{}
	require.NoError(t, line.BeforeAppendModel(t.Context(), &bun.InsertQuery{}))
	assert.True(t, line.ID.IsNotNil())
	assert.NotZero(t, line.CreatedAt)

	require.NoError(t, request.BeforeAppendModel(t.Context(), &bun.UpdateQuery{}))
	require.NoError(t, line.BeforeAppendModel(t.Context(), &bun.UpdateQuery{}))
	assert.NotZero(t, request.UpdatedAt)
	assert.NotZero(t, line.UpdatedAt)
}
