package factoringtools

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLedger struct {
	factoringports.LedgerService

	guard     agenttoolstest.WriteGuard
	sub       *factoring.Submission
	item      *factoring.SubmissionItem
	funding   *factoringports.RecordFundingRequest
	movement  *factoringports.ReserveMovementRequest
	recorded  string
	emptyPlan bool
}

func newFakeLedger() *fakeLedger {
	item := &factoring.SubmissionItem{
		ID:                   pulid.MustNew("facsit_"),
		InvoiceID:            pulid.MustNew("inv_"),
		InvoiceNumber:        "INV-7",
		FaceMinor:            100_000,
		ExpectedAdvanceMinor: 90_000,
		ExpectedFeeMinor:     3_000,
		Status:               factoring.SubmissionItemStatusPending,
	}
	return &fakeLedger{
		sub: &factoring.Submission{
			ID:           pulid.MustNew("facsub_"),
			Number:       "FS2610000009",
			Status:       factoring.SubmissionStatusSent,
			CurrencyCode: "USD",
			Version:      3,
			Items:        []*factoring.SubmissionItem{item},
		},
		item: item,
	}
}

func (f *fakeLedger) planned(kind factoring.ReserveEntryKind) *factoringports.PlannedMovement {
	if f.emptyPlan {
		return &factoringports.PlannedMovement{
			ItemID:        f.item.ID,
			InvoiceNumber: f.item.InvoiceNumber,
			Movement:      factoring.Movement{Kind: kind},
		}
	}
	return &factoringports.PlannedMovement{
		ItemID:        f.item.ID,
		InvoiceID:     f.item.InvoiceID,
		InvoiceNumber: f.item.InvoiceNumber,
		Movement:      factoring.Movement{Kind: kind, CashMinor: 88_000},
		Lines: []factoringports.PlannedLine{
			{Role: factoring.AccountRoleCash, AccountID: pulid.MustNew("gla_"), Debit: 88_000},
			{Role: factoring.AccountRoleFeeExpense, AccountID: pulid.MustNew("gla_"), Debit: 2_500},
			{
				Role:      factoring.AccountRoleDueFromFactor,
				AccountID: pulid.MustNew("gla_"),
				Debit:     9_500,
			},
			{Role: factoring.AccountRoleAR, AccountID: pulid.MustNew("gla_"), Credit: 100_000},
		},
		RequestedDate:    1_800_000_000,
		AccountingDate:   1_800_000_000,
		FiscalPeriodID:   pulid.MustNew("fp_"),
		FiscalPeriodName: "October 2026",
		EntryStatus:      "Posted",
	}
}

func (f *fakeLedger) PlanFunding(
	_ context.Context,
	req *factoringports.RecordFundingRequest,
	_ *permissionports.RequestActor,
) (*factoringports.FundingPlan, error) {
	if req.SubmissionID != f.sub.ID {
		return nil, errortypes.NewNotFoundError("Factoring schedule not found")
	}
	after := *f.sub
	after.Status = factoring.SubmissionStatusFunded
	return &factoringports.FundingPlan{
		Before:       f.sub,
		After:        &after,
		Movements:    []*factoringports.PlannedMovement{f.planned(factoring.ReserveEntryFunding)},
		FundedCount:  1,
		AdvanceMinor: 88_000,
		FeeMinor:     2_500,
		ReserveMinor: 9_500,
	}, nil
}

func (f *fakeLedger) RecordFunding(
	_ context.Context,
	req *factoringports.RecordFundingRequest,
	_ *permissionports.RequestActor,
) (*factoringports.FundingResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.funding = req
	return &factoringports.FundingResult{Submission: f.sub}, nil
}

func (f *fakeLedger) movementPlan(
	kind factoring.ReserveEntryKind,
) *factoringports.ReserveMovementPlan {
	after := *f.item
	switch kind {
	case factoring.ReserveEntryRelease:
		after.Status = factoring.SubmissionItemStatusSettled
	case factoring.ReserveEntryChargeback:
		after.Status = factoring.SubmissionItemStatusChargedBack
	case factoring.ReserveEntryFunding, factoring.ReserveEntryChargebackReversal:
		after.Status = factoring.SubmissionItemStatusFunded
	}
	return &factoringports.ReserveMovementPlan{
		Before:   f.sub,
		After:    f.sub,
		Item:     &after,
		Movement: f.planned(kind),
	}
}

func (f *fakeLedger) PlanRelease(
	_ context.Context,
	_ *factoringports.ReserveMovementRequest,
	_ *permissionports.RequestActor,
) (*factoringports.ReserveMovementPlan, error) {
	return f.movementPlan(factoring.ReserveEntryRelease), nil
}

func (f *fakeLedger) ReleaseReserve(
	_ context.Context,
	req *factoringports.ReserveMovementRequest,
	_ *permissionports.RequestActor,
) (*factoringports.ReserveMovementResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.movement, f.recorded = req, "release"
	return &factoringports.ReserveMovementResult{Submission: f.sub}, nil
}

func (f *fakeLedger) PlanChargeback(
	_ context.Context,
	_ *factoringports.ReserveMovementRequest,
	_ *permissionports.RequestActor,
) (*factoringports.ReserveMovementPlan, error) {
	return f.movementPlan(factoring.ReserveEntryChargeback), nil
}

func (f *fakeLedger) RecordChargeback(
	_ context.Context,
	req *factoringports.ReserveMovementRequest,
	_ *permissionports.RequestActor,
) (*factoringports.ReserveMovementResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.movement, f.recorded = req, "chargeback"
	return &factoringports.ReserveMovementResult{Submission: f.sub}, nil
}

func fundingParams(f *fakeLedger, items ...any) agenttoolports.ToolExecuteParams {
	return toolParams(map[string]any{
		paramSubmissionID: f.sub.ID.String(),
		paramVersion:      float64(f.sub.Version),
		paramFundedOn:     "2026-10-14",
		paramItems:        items,
		paramMemo:         "Funding report 4411",
	})
}

func TestLedgerToolPolicies_AreMoneyAndProposeOnly(t *testing.T) {
	t.Parallel()

	ledger := newFakeLedger()
	for _, tool := range []agenttoolports.AgentTool{
		NewRecordFactoringFundingTool(ledger, nil),
		NewReleaseFactoringReserveTool(ledger, nil),
		NewRecordFactoringChargebackTool(ledger, nil),
		NewReverseFactoringChargebackTool(ledger, nil),
	} {
		policy := tool.Policy()
		assert.Equal(t, tool.Name(), policy.Name)
		assert.Equal(t, permission.ResourceFactoringReserve, policy.Resource, tool.Name())
		assert.Equal(t, permission.OpCreate, policy.Operation, tool.Name())
		assert.Equal(t, agent.TierPropose, policy.DefaultTier, tool.Name())
		assert.Equal(t, agent.TierPropose, policy.MaxTier, tool.Name())
		assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress, tool.Name())
	}
}

func TestRecordFactoringFunding_PreviewsTheJournalAndRunsOnlyWhenApproved(t *testing.T) {
	t.Parallel()

	ledger := newFakeLedger()
	tool := NewRecordFactoringFundingTool(ledger, nil)
	params := fundingParams(ledger, map[string]any{
		paramItemID:  ledger.item.ID.String(),
		paramOutcome: string(factoring.FundingOutcomeFunded),
		paramAdvance: "880.00",
		paramFee:     "25",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&ledger.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, permission.ResourceFactoringSubmission, preview.Changes[0].Resource)
	assert.Equal(t, permission.ResourceJournalEntry, preview.Changes[1].Resource)
	assert.Contains(t, preview.Summary, "FS2610000009")
	assert.Contains(t, preview.Summary, "1 invoice funded")

	err := tool.Execute(t.Context(), params)
	require.ErrorIs(t, err, common.ErrNeedsAPersonsApproval)
	assert.Nil(t, ledger.funding)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.Approved(params)))
	require.NotNil(t, ledger.funding)
	require.Len(t, ledger.funding.Items, 1)
	item := ledger.funding.Items[0]
	require.NotNil(t, item.AdvanceMinor)
	assert.Equal(t, int64(88_000), *item.AdvanceMinor)
	require.NotNil(t, item.FeeMinor)
	assert.Equal(t, int64(2_500), *item.FeeMinor)
	assert.Equal(t, time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC).Unix(), ledger.funding.FundedOn)
	assert.Equal(t, "Funding report 4411", ledger.funding.Memo)
	assert.Equal(t, int64(3), ledger.funding.Version)
}

func TestRecordFactoringFunding_RefusesBadItems(t *testing.T) {
	t.Parallel()

	ledger := newFakeLedger()
	tool := NewRecordFactoringFundingTool(ledger, nil)
	validator := tool.(agenttoolports.ToolValidator)

	cases := map[string][]any{
		"no items": {},
		"unknown outcome": {
			map[string]any{paramItemID: ledger.item.ID.String(), paramOutcome: "Paid"},
		},
		"negative advance": {
			map[string]any{
				paramItemID:  ledger.item.ID.String(),
				paramOutcome: "Funded",
				paramAdvance: "-1",
			},
		},
		"not an object": {"funded"},
	}
	for name, items := range cases {
		err := validator.Validate(t.Context(), fundingParams(ledger, items...))
		require.Error(t, err, name)
	}
}

func TestRecordFactoringChargeback_NeedsAReason(t *testing.T) {
	t.Parallel()

	ledger := newFakeLedger()
	tool := NewRecordFactoringChargebackTool(ledger, nil)
	params := toolParams(map[string]any{
		paramSubmissionID: ledger.sub.ID.String(),
		paramItemID:       ledger.item.ID.String(),
		paramVersion:      float64(ledger.sub.Version),
		paramOccurred:     "2026-10-20",
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))

	params.Params[paramReason] = "Customer filed for bankruptcy"
	params.Params[paramCash] = "930.00"
	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "chargeback of invoice INV-7")

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.Approved(params)))
	assert.Equal(t, "chargeback", ledger.recorded)
	require.NotNil(t, ledger.movement.CashMinor)
	assert.Equal(t, int64(93_000), *ledger.movement.CashMinor)
	assert.Equal(t, "Customer filed for bankruptcy", ledger.movement.Reason)
}

func TestReleaseFactoringReserve_SaysWhenNothingIsBooked(t *testing.T) {
	t.Parallel()

	ledger := newFakeLedger()
	ledger.emptyPlan = true
	tool := NewReleaseFactoringReserveTool(ledger, nil)
	params := toolParams(map[string]any{
		paramSubmissionID: ledger.sub.ID.String(),
		paramItemID:       ledger.item.ID.String(),
		paramVersion:      float64(ledger.sub.Version),
		paramOccurred:     "2026-10-20",
	})
	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, preview.Changes, 1)
	assert.Contains(t, preview.Summary, "Nothing is booked")

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.Approved(params)))
	assert.Equal(t, "release", ledger.recorded)
	assert.Nil(t, ledger.movement.CashMinor)
}
