package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRunKeeper struct {
	guard     *writeGuard
	built     *invoicerun.InvoiceRun
	change    *serviceports.InvoiceRunChangePreview
	commit    *serviceports.InvoiceRunCommitPlan
	statement *serviceports.StatementBillPlan
	previewed *serviceports.PreviewInvoiceRunRequest
	adjusted  *serviceports.AdjustInvoiceRunMembershipRequest
	committed *serviceports.CommitInvoiceRunRequest
	canceled  *serviceports.CancelInvoiceRunRequest
	billed    *serviceports.BillStatementNowRequest
}

func (f *fakeRunKeeper) Preview(
	_ context.Context,
	req *serviceports.PreviewInvoiceRunRequest,
	_ *serviceports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.previewed = req

	return f.built, nil
}

func (f *fakeRunKeeper) PreviewBuild(
	context.Context,
	*serviceports.PreviewInvoiceRunRequest,
	*serviceports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	return f.built, nil
}

func (f *fakeRunKeeper) AdjustMembership(
	_ context.Context,
	req *serviceports.AdjustInvoiceRunMembershipRequest,
	_ *serviceports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.adjusted = req

	return f.change.After, nil
}

func (f *fakeRunKeeper) PreviewMembership(
	context.Context,
	*serviceports.AdjustInvoiceRunMembershipRequest,
) (*serviceports.InvoiceRunChangePreview, error) {
	return f.change, nil
}

func (f *fakeRunKeeper) Commit(
	_ context.Context,
	req *serviceports.CommitInvoiceRunRequest,
	_ *serviceports.RequestActor,
) (*serviceports.CommitInvoiceRunResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.committed = req

	return &serviceports.CommitInvoiceRunResult{}, nil
}

func (f *fakeRunKeeper) PreviewCommit(
	context.Context,
	*serviceports.CommitInvoiceRunRequest,
) (*serviceports.InvoiceRunCommitPlan, error) {
	return f.commit, nil
}

func (f *fakeRunKeeper) Cancel(
	_ context.Context,
	req *serviceports.CancelInvoiceRunRequest,
	_ *serviceports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.canceled = req

	return f.change.After, nil
}

func (f *fakeRunKeeper) PreviewCancel(
	context.Context,
	*serviceports.CancelInvoiceRunRequest,
	*serviceports.RequestActor,
) (*serviceports.InvoiceRunChangePreview, error) {
	return f.change, nil
}

func (f *fakeRunKeeper) BillStatementNow(
	_ context.Context,
	req *serviceports.BillStatementNowRequest,
	_ *serviceports.RequestActor,
) (*serviceports.CommitInvoiceRunResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.billed = req

	return &serviceports.CommitInvoiceRunResult{}, nil
}

func (f *fakeRunKeeper) PreviewBillStatement(
	context.Context,
	*serviceports.BillStatementNowRequest,
	*serviceports.RequestActor,
) (*serviceports.StatementBillPlan, error) {
	return f.statement, nil
}

func proposedRun() *invoicerun.InvoiceRun {
	customerID := pulid.MustNew("cus_")
	group := &invoicerun.InvoiceRunGroup{
		ID:           pulid.MustNew("invrg_"),
		CustomerID:   customerID,
		GroupLabel:   "Acme Foods",
		CurrencyCode: "USD",
		ItemCount:    3,
		TotalAmount:  decimal.RequireFromString("1850.00"),
		Items: []*invoicerun.InvoiceRunGroupItem{
			{ID: pulid.MustNew("invrgi_"), Amount: decimal.RequireFromString("850.00")},
			{ID: pulid.MustNew("invrgi_"), Amount: decimal.RequireFromString("600.00")},
			{ID: pulid.MustNew("invrgi_"), Amount: decimal.RequireFromString("400.00")},
		},
	}
	return &invoicerun.InvoiceRun{
		ID:           pulid.MustNew("invrun_"),
		Number:       "RUN-12",
		Status:       invoicerun.StatusReady,
		PeriodStart:  time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC).Unix(),
		PeriodEnd:    time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Unix(),
		CurrencyCode: "USD",
		GroupCount:   1,
		ItemCount:    3,
		TotalAmount:  decimal.RequireFromString("1850.00"),
		Groups:       []*invoicerun.InvoiceRunGroup{group},
		Version:      5,
	}
}

func TestBuildInvoiceRun_PreviewsTheProposedInvoicesAndBuildsIt(t *testing.T) {
	t.Parallel()

	runs := &fakeRunKeeper{built: proposedRun(), guard: &writeGuard{}}
	tool := newBuildInvoiceRunTool(runs)
	customerID := runs.built.Groups[0].CustomerID
	params := agentParamsFor(map[string]any{
		paramCustomerIDs: []any{customerID.String()},
		paramPeriodStart: "2026-09-01",
		paramPeriodEnd:   "2026-09-30",
		paramInvoiceDate: "2026-10-01",
	})

	preview := previewWithoutWrites(t, runs.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would build an invoice run proposing 1 invoice for 3 shipments")
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, permission.ResourceInvoiceRun, previewChange(t, preview, 0).Resource)
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 1).Operation)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, runs.previewed)
	assert.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC).Unix(), runs.previewed.PeriodStart)
	assert.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Unix(), runs.previewed.PeriodEnd)
	assert.Equal(t, invoicerun.SourceManual, runs.previewed.Source)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceInvoiceRun, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
}

func TestBuildInvoiceRun_RefusesAPeriodThatEndsBeforeItStarts(t *testing.T) {
	t.Parallel()

	tool := newBuildInvoiceRunTool(&fakeRunKeeper{built: proposedRun()})
	params := executeParams(map[string]any{
		paramCustomerIDs: []any{pulid.MustNew("cus_").String()},
		paramPeriodStart: "2026-09-30",
		paramPeriodEnd:   "2026-09-01",
	})
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
}

func TestAdjustInvoiceRunMembership_ShowsTheNewTotals(t *testing.T) {
	t.Parallel()

	before := proposedRun()
	after := proposedRun()
	after.ID = before.ID
	after.Groups[0].ID = before.Groups[0].ID
	after.Groups[0].ItemCount = 2
	after.Groups[0].TotalAmount = decimal.RequireFromString("1450.00")
	after.TotalAmount = decimal.RequireFromString("1450.00")
	after.ExcludedCount = 1
	runs := &fakeRunKeeper{
		change: &serviceports.InvoiceRunChangePreview{Before: before, After: after},
		guard:  &writeGuard{},
	}
	tool := newAdjustInvoiceRunMembershipTool(runs)
	itemID := before.Groups[0].Items[2].ID
	params := executeParams(map[string]any{
		paramInvoiceRunID: before.ID.String(),
		paramExclude: []any{map[string]any{
			paramItemID: itemID.String(),
			paramReason: "POD missing",
		}},
	})

	preview := previewWithoutWrites(t, runs.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "1850.00 to 1450.00")
	require.NotEmpty(t, preview.Changes)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, runs.adjusted)
	require.Len(t, runs.adjusted.Exclude, 1)
	assert.Equal(t, itemID, runs.adjusted.Exclude[0].ItemID)
	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceInvoiceRun, target.Resource)
}

func TestAdjustInvoiceRunMembership_NeedsAnEdit(t *testing.T) {
	t.Parallel()

	tool := newAdjustInvoiceRunMembershipTool(&fakeRunKeeper{})
	params := executeParams(map[string]any{paramInvoiceRunID: pulid.MustNew("invrun_").String()})
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
}

func TestCommitInvoiceRun_OnlyAPersonCommitsWhatWouldBill(t *testing.T) {
	t.Parallel()

	run := proposedRun()
	runs := &fakeRunKeeper{
		guard: &writeGuard{},
		commit: &serviceports.InvoiceRunCommitPlan{
			Run: run,
			Groups: []serviceports.InvoiceRunGroupPlan{
				{
					GroupID: run.Groups[0].ID, GroupLabel: "Acme Foods", ShipmentCount: 3,
					Total: decimal.RequireFromString("1850.00"), CurrencyCode: "USD",
					Outcome: serviceports.InvoiceRunGroupBills,
				},
				{
					GroupID: pulid.MustNew("invrg_"), GroupLabel: "Beta Farms", ShipmentCount: 1,
					Total: decimal.RequireFromString("90.00"), CurrencyCode: "USD",
					Outcome: serviceports.InvoiceRunGroupSkips,
					Reason:  "Below this customer's 100.00 minimum — held for the next period",
				},
			},
		},
	}
	tool := newCommitInvoiceRunTool(runs)
	params := executeParams(map[string]any{paramInvoiceRunID: run.ID.String()})

	preview := previewWithoutWrites(t, runs.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would commit Invoice run RUN-12, making 1 invoice for 1850.00")
	assert.Contains(t, preview.Summary, "Beta Farms is skipped")

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, runs.committed)

	policy := tool.Policy()
	assert.Equal(t, permission.OpApprove, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
}

func TestCancelInvoiceRun_CancelsWithAReason(t *testing.T) {
	t.Parallel()

	before := proposedRun()
	after := *before
	after.Status = invoicerun.StatusCanceled
	after.FailureReason = "Built for the wrong period"
	runs := &fakeRunKeeper{
		change: &serviceports.InvoiceRunChangePreview{Before: before, After: &after},
		guard:  &writeGuard{},
	}
	tool := newCancelInvoiceRunTool(runs)
	params := executeParams(map[string]any{
		paramInvoiceRunID: before.ID.String(),
		paramReason:       "Built for the wrong period",
	})

	preview := previewWithoutWrites(t, runs.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Canceled", fieldByPath(t, previewChange(t, preview, 0), "status").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, runs.canceled)
	assert.Equal(t, "Built for the wrong period", runs.canceled.Reason)
	assert.Equal(t, permission.OpCancel, tool.Policy().Operation)
}

func TestBillStatementNow_OnlyAPersonBillsEarly(t *testing.T) {
	t.Parallel()

	run := proposedRun()
	runs := &fakeRunKeeper{
		guard: &writeGuard{},
		statement: &serviceports.StatementBillPlan{
			Statement: &serviceports.OpenStatement{
				CustomerID:   run.Groups[0].CustomerID,
				CustomerName: "Acme Foods",
				CurrencyCode: "USD",
			},
			Run: run,
			Groups: []serviceports.InvoiceRunGroupPlan{{
				GroupLabel: "Acme Foods", ShipmentCount: 3, CurrencyCode: "USD",
				Total:   decimal.RequireFromString("1850.00"),
				Outcome: serviceports.InvoiceRunGroupBills,
			}},
		},
	}
	tool := newBillStatementNowTool(runs)
	params := executeParams(map[string]any{
		paramCustomerID: run.Groups[0].CustomerID.String(),
		paramReason:     "Customer asked for an early invoice to close their quarter",
	})

	preview := previewWithoutWrites(t, runs.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would bill Acme Foods's open statement now")

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, runs.billed)
	assert.Equal(t, permission.OpApprove, tool.Policy().Operation)
}
