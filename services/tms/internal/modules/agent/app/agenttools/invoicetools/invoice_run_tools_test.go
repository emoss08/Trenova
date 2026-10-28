package invoicetools

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRunKeeper struct {
	guard     *agenttoolstest.WriteGuard
	built     *invoicerun.InvoiceRun
	change    *invoicerunports.InvoiceRunChangePreview
	commit    *invoicerunports.InvoiceRunCommitPlan
	statement *invoicerunports.StatementBillPlan
	previewed *invoicerunports.PreviewInvoiceRunRequest
	adjusted  *invoicerunports.AdjustInvoiceRunMembershipRequest
	committed *invoicerunports.CommitInvoiceRunRequest
	canceled  *invoicerunports.CancelInvoiceRunRequest
	billed    *invoicerunports.BillStatementNowRequest
}

func (f *fakeRunKeeper) Preview(
	_ context.Context,
	req *invoicerunports.PreviewInvoiceRunRequest,
	_ *permissionports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.previewed = req

	return f.built, nil
}

func (f *fakeRunKeeper) PreviewBuild(
	context.Context,
	*invoicerunports.PreviewInvoiceRunRequest,
	*permissionports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	return f.built, nil
}

func (f *fakeRunKeeper) AdjustMembership(
	_ context.Context,
	req *invoicerunports.AdjustInvoiceRunMembershipRequest,
	_ *permissionports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.adjusted = req

	return f.change.After, nil
}

func (f *fakeRunKeeper) PreviewMembership(
	context.Context,
	*invoicerunports.AdjustInvoiceRunMembershipRequest,
) (*invoicerunports.InvoiceRunChangePreview, error) {
	return f.change, nil
}

func (f *fakeRunKeeper) Commit(
	_ context.Context,
	req *invoicerunports.CommitInvoiceRunRequest,
	_ *permissionports.RequestActor,
) (*invoicerunports.CommitInvoiceRunResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.committed = req

	return &invoicerunports.CommitInvoiceRunResult{}, nil
}

func (f *fakeRunKeeper) PreviewCommit(
	context.Context,
	*invoicerunports.CommitInvoiceRunRequest,
) (*invoicerunports.InvoiceRunCommitPlan, error) {
	return f.commit, nil
}

func (f *fakeRunKeeper) Cancel(
	_ context.Context,
	req *invoicerunports.CancelInvoiceRunRequest,
	_ *permissionports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.canceled = req

	return f.change.After, nil
}

func (f *fakeRunKeeper) PreviewCancel(
	context.Context,
	*invoicerunports.CancelInvoiceRunRequest,
	*permissionports.RequestActor,
) (*invoicerunports.InvoiceRunChangePreview, error) {
	return f.change, nil
}

func (f *fakeRunKeeper) BillStatementNow(
	_ context.Context,
	req *invoicerunports.BillStatementNowRequest,
	_ *permissionports.RequestActor,
) (*invoicerunports.CommitInvoiceRunResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.billed = req

	return &invoicerunports.CommitInvoiceRunResult{}, nil
}

func (f *fakeRunKeeper) PreviewBillStatement(
	context.Context,
	*invoicerunports.BillStatementNowRequest,
	*permissionports.RequestActor,
) (*invoicerunports.StatementBillPlan, error) {
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

	runs := &fakeRunKeeper{built: proposedRun(), guard: &agenttoolstest.WriteGuard{}}
	tool := NewBuildInvoiceRunTool(runs)
	customerID := runs.built.Groups[0].CustomerID
	params := agenttoolstest.AgentParamsFor(map[string]any{
		toolparams.ParamCustomerIDs: []any{customerID.String()},
		toolparams.ParamPeriodStart: "2026-09-01",
		toolparams.ParamPeriodEnd:   "2026-09-30",
		toolparams.ParamInvoiceDate: "2026-10-01",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		runs.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(
		t,
		preview.Summary,
		"Would build an invoice run proposing 1 invoice for 3 shipments",
	)
	require.Len(t, preview.Changes, 2)
	assert.Equal(
		t,
		permission.ResourceInvoiceRun,
		agenttoolstest.PreviewChange(t, preview, 0).Resource,
	)
	assert.Equal(
		t,
		agent.PreviewOperationCreate,
		agenttoolstest.PreviewChange(t, preview, 1).Operation,
	)

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

	tool := NewBuildInvoiceRunTool(&fakeRunKeeper{built: proposedRun()})
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamCustomerIDs: []any{pulid.MustNew("cus_").String()},
		toolparams.ParamPeriodStart: "2026-09-30",
		toolparams.ParamPeriodEnd:   "2026-09-01",
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
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
		change: &invoicerunports.InvoiceRunChangePreview{Before: before, After: after},
		guard:  &agenttoolstest.WriteGuard{},
	}
	tool := NewAdjustInvoiceRunMembershipTool(runs)
	itemID := before.Groups[0].Items[2].ID
	params := agenttoolstest.ExecuteParams(map[string]any{
		ParamInvoiceRunID: before.ID.String(),
		paramExclude: []any{map[string]any{
			toolparams.ParamItemID: itemID.String(),
			toolparams.ParamReason: "POD missing",
		}},
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		runs.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "1850.00 to 1450.00")
	require.NotEmpty(t, preview.Changes)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, runs.adjusted)
	require.Len(t, runs.adjusted.Exclude, 1)
	assert.Equal(t, itemID, runs.adjusted.Exclude[0].ItemID)
	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceInvoiceRun, target.Resource)
}

func TestAdjustInvoiceRunMembership_NeedsAnEdit(t *testing.T) {
	t.Parallel()

	tool := NewAdjustInvoiceRunMembershipTool(&fakeRunKeeper{})
	params := agenttoolstest.ExecuteParams(
		map[string]any{ParamInvoiceRunID: pulid.MustNew("invrun_").String()},
	)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
}

func TestCommitInvoiceRun_OnlyAPersonCommitsWhatWouldBill(t *testing.T) {
	t.Parallel()

	run := proposedRun()
	runs := &fakeRunKeeper{
		guard: &agenttoolstest.WriteGuard{},
		commit: &invoicerunports.InvoiceRunCommitPlan{
			Run: run,
			Groups: []invoicerunports.InvoiceRunGroupPlan{
				{
					GroupID: run.Groups[0].ID, GroupLabel: "Acme Foods", ShipmentCount: 3,
					Total: decimal.RequireFromString("1850.00"), CurrencyCode: "USD",
					Outcome: invoicerunports.InvoiceRunGroupBills,
				},
				{
					GroupID: pulid.MustNew("invrg_"), GroupLabel: "Beta Farms", ShipmentCount: 1,
					Total: decimal.RequireFromString("90.00"), CurrencyCode: "USD",
					Outcome: invoicerunports.InvoiceRunGroupSkips,
					Reason:  "Below this customer's 100.00 minimum — held for the next period",
				},
			},
		},
	}
	tool := NewCommitInvoiceRunTool(runs)
	params := agenttoolstest.ExecuteParams(map[string]any{ParamInvoiceRunID: run.ID.String()})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		runs.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(
		t,
		preview.Summary,
		"Would commit Invoice run RUN-12, making 1 invoice for 1850.00",
	)
	assert.Contains(t, preview.Summary, "Beta Farms is skipped")

	require.ErrorIs(t, tool.Execute(t.Context(), params), toolspec.ErrNeedsAPersonsApproval)
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
		change: &invoicerunports.InvoiceRunChangePreview{Before: before, After: &after},
		guard:  &agenttoolstest.WriteGuard{},
	}
	tool := NewCancelInvoiceRunTool(runs)
	params := agenttoolstest.ExecuteParams(map[string]any{
		ParamInvoiceRunID:      before.ID.String(),
		toolparams.ParamReason: "Built for the wrong period",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		runs.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		"Canceled",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "status").After,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, runs.canceled)
	assert.Equal(t, "Built for the wrong period", runs.canceled.Reason)
	assert.Equal(t, permission.OpCancel, tool.Policy().Operation)
}

func TestBillStatementNow_OnlyAPersonBillsEarly(t *testing.T) {
	t.Parallel()

	run := proposedRun()
	runs := &fakeRunKeeper{
		guard: &agenttoolstest.WriteGuard{},
		statement: &invoicerunports.StatementBillPlan{
			Statement: &invoicerunports.OpenStatement{
				CustomerID:   run.Groups[0].CustomerID,
				CustomerName: "Acme Foods",
				CurrencyCode: "USD",
			},
			Run: run,
			Groups: []invoicerunports.InvoiceRunGroupPlan{{
				GroupLabel: "Acme Foods", ShipmentCount: 3, CurrencyCode: "USD",
				Total:   decimal.RequireFromString("1850.00"),
				Outcome: invoicerunports.InvoiceRunGroupBills,
			}},
		},
	}
	tool := NewBillStatementNowTool(runs)
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamCustomerID: run.Groups[0].CustomerID.String(),
		toolparams.ParamReason:     "Customer asked for an early invoice to close their quarter",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		runs.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would bill Acme Foods's open statement now")

	require.ErrorIs(t, tool.Execute(t.Context(), params), toolspec.ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, runs.billed)
	assert.Equal(t, permission.OpApprove, tool.Policy().Operation)
}
