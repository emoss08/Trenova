package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDisputer struct {
	preview   *serviceports.InvoiceDisputePreview
	refusal   error
	guard     *writeGuard
	opened    *serviceports.OpenInvoiceDisputeRequest
	resolved  *serviceports.ResolveInvoiceDisputeRequest
	withdrawn *serviceports.WithdrawInvoiceDisputeRequest
}

func (f *fakeDisputer) plan() (*serviceports.InvoiceDisputePreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.preview, nil
}

func (f *fakeDisputer) Open(
	_ context.Context,
	req *serviceports.OpenInvoiceDisputeRequest,
	_ *serviceports.RequestActor,
) (*invoice.InvoiceDispute, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.opened = req

	return f.preview.After, nil
}

func (f *fakeDisputer) Resolve(
	_ context.Context,
	req *serviceports.ResolveInvoiceDisputeRequest,
	_ *serviceports.RequestActor,
) (*invoice.InvoiceDispute, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.resolved = req

	return f.preview.After, nil
}

func (f *fakeDisputer) Withdraw(
	_ context.Context,
	req *serviceports.WithdrawInvoiceDisputeRequest,
	_ *serviceports.RequestActor,
) (*invoice.InvoiceDispute, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.withdrawn = req

	return f.preview.After, nil
}

func (f *fakeDisputer) PreviewOpen(
	context.Context,
	*serviceports.OpenInvoiceDisputeRequest,
	*serviceports.RequestActor,
) (*serviceports.InvoiceDisputePreview, error) {
	return f.plan()
}

func (f *fakeDisputer) PreviewResolve(
	context.Context,
	*serviceports.ResolveInvoiceDisputeRequest,
	*serviceports.RequestActor,
) (*serviceports.InvoiceDisputePreview, error) {
	return f.plan()
}

func (f *fakeDisputer) PreviewWithdraw(
	context.Context,
	*serviceports.WithdrawInvoiceDisputeRequest,
	*serviceports.RequestActor,
) (*serviceports.InvoiceDisputePreview, error) {
	return f.plan()
}

func disputedInvoice() *invoice.Invoice {
	return &invoice.Invoice{
		ID:            pulid.MustNew("inv_"),
		Number:        "INV-4100",
		Status:        invoice.StatusPosted,
		CurrencyCode:  "USD",
		DisputeStatus: invoice.DisputeStatusNone,
		Version:       3,
	}
}

func openDisputePlan() *serviceports.InvoiceDisputePreview {
	inv := disputedInvoice()
	after := *inv
	after.DisputeStatus = invoice.DisputeStatusDisputed

	return &serviceports.InvoiceDisputePreview{
		After: &invoice.InvoiceDispute{
			InvoiceID:           inv.ID,
			Status:              invoice.DisputeCaseStatusOpen,
			ReasonCode:          invoice.DisputeReasonAccessorialDisputed,
			DisputedAmount:      decimal.RequireFromString("150.00"),
			DisputedAmountMinor: 15000,
			Notes:               "Customer disputes the detention line",
		},
		InvoiceBefore: inv,
		InvoiceAfter:  &after,
	}
}

func openDisputeParams(invoiceID pulid.ID) serviceports.ToolExecuteParams {
	return executeParams(map[string]any{
		paramInvoiceID:         invoiceID.String(),
		paramDisputeReasonCode: string(invoice.DisputeReasonAccessorialDisputed),
		paramDisputedAmount:    "150.00",
		paramDisputeNotes:      " Customer disputes the detention line ",
	})
}

func TestOpenInvoiceDispute_PreviewShowsTheCaseAndTheFlag(t *testing.T) {
	t.Parallel()

	disputer := &fakeDisputer{preview: openDisputePlan(), guard: &writeGuard{}}
	tool := newOpenInvoiceDisputeTool(disputer).(serviceports.ToolPreviewer)
	params := openDisputeParams(disputer.preview.InvoiceBefore.ID)

	preview := previewWithoutWrites(t, disputer.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Equal(t, "Would open a dispute on Invoice INV-4100 for 150.00 USD (AccessorialDisputed).",
		preview.Summary)
	require.Len(t, preview.Changes, 2)
	created := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceInvoiceDispute, created.Resource)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)
	assert.Equal(t, "Open", fieldByPath(t, created, "status").After)
	require.NotNil(t, created.Money)
	flag := previewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceInvoice, flag.Resource)
	assert.Equal(t, "Disputed", fieldByPath(t, flag, fieldDisputeStatus).After)
}

func TestOpenInvoiceDispute_ARefusalIsAWouldFail(t *testing.T) {
	t.Parallel()

	disputer := &fakeDisputer{
		preview: openDisputePlan(),
		refusal: errortypes.NewValidationError(
			"disputedAmount",
			errortypes.ErrInvalid,
			"Disputed amount exceeds the open balance of 100.00",
		),
	}
	tool := newOpenInvoiceDisputeTool(disputer)
	params := openDisputeParams(disputer.preview.InvoiceBefore.ID)

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	assert.Empty(t, preview.Changes)

	err = tool.(serviceports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the open balance")
}

func TestOpenInvoiceDispute_RefusesBadArguments(t *testing.T) {
	t.Parallel()

	tool := newOpenInvoiceDisputeTool(&fakeDisputer{preview: openDisputePlan()})
	invoiceID := pulid.MustNew("inv_").String()
	cases := map[string]map[string]any{
		"an unknown reason": {
			paramInvoiceID: invoiceID, paramDisputeReasonCode: "TooExpensive",
			paramDisputedAmount: "10",
		},
		"a negative amount": {
			paramInvoiceID: invoiceID, paramDisputeReasonCode: "Other",
			paramDisputedAmount: "-10",
		},
		"no invoice": {paramDisputeReasonCode: "Other", paramDisputedAmount: "10"},
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
				executeParams(raw)))
		})
	}
}

func TestOpenInvoiceDispute_OpensAsTheActorWithTrimmedNotes(t *testing.T) {
	t.Parallel()

	disputer := &fakeDisputer{preview: openDisputePlan()}
	tool := newOpenInvoiceDisputeTool(disputer)
	params := openDisputeParams(disputer.preview.InvoiceBefore.ID)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, disputer.opened)
	assert.Equal(t, "Customer disputes the detention line", disputer.opened.Notes)
	assert.True(t, disputer.opened.DisputedAmount.Equal(decimal.RequireFromString("150")))
	assert.Equal(t, params.OrganizationID, disputer.opened.TenantInfo.OrgID)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceInvoiceDispute, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
	assert.NotNil(t, policy.TaintHold)
	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceInvoice, target.Resource)
}

func resolvedDisputePlan() *serviceports.InvoiceDisputePreview {
	plan := openDisputePlan()
	plan.InvoiceBefore.DisputeStatus = invoice.DisputeStatusDisputed
	plan.InvoiceAfter.DisputeStatus = invoice.DisputeStatusNone
	before := *plan.After
	before.ID = pulid.MustNew("idsp_")
	before.Version = 1
	after := before
	after.Status = invoice.DisputeCaseStatusResolved
	after.Resolution = invoice.DisputeResolutionInvoiceUpheld
	after.ResolutionNotes = "Detention was on the rate confirmation"
	plan.Before = &before
	plan.After = &after

	return plan
}

func TestResolveInvoiceDispute_OnlyAPersonResolves(t *testing.T) {
	t.Parallel()

	disputer := &fakeDisputer{preview: resolvedDisputePlan(), guard: &writeGuard{}}
	tool := newResolveInvoiceDisputeTool(disputer)
	params := executeParams(map[string]any{
		paramDisputeID:         disputer.preview.Before.ID.String(),
		paramDisputeResolution: string(invoice.DisputeResolutionInvoiceUpheld),
		paramResolutionNotes:   "Detention was on the rate confirmation",
	})

	preview := previewWithoutWrites(t, disputer.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, "Resolved", fieldByPath(t, change, "status").After)
	assert.Equal(t, "InvoiceUpheld", fieldByPath(t, change, paramDisputeResolution).After)
	assert.Equal(t, "None", fieldByPath(t, previewChange(t, preview, 1), fieldDisputeStatus).After)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	assert.Nil(t, disputer.resolved)

	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, disputer.resolved)
	assert.Equal(t, invoice.DisputeResolutionInvoiceUpheld, disputer.resolved.Resolution)

	policy := tool.Policy()
	assert.Equal(t, permission.OpApprove, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceInvoiceDispute, target.Resource)
}

func TestResolveInvoiceDispute_AnAgentPrincipalCannotEvenPreviewIt(t *testing.T) {
	t.Parallel()

	disputer := &fakeDisputer{preview: resolvedDisputePlan()}
	tool := newResolveInvoiceDisputeTool(disputer).(serviceports.ToolPreviewer)
	params := agentParamsFor(map[string]any{
		paramDisputeID:         disputer.preview.Before.ID.String(),
		paramDisputeResolution: string(invoice.DisputeResolutionInvoiceUpheld),
	})

	_, err := tool.Preview(t.Context(), params)
	require.ErrorIs(t, err, ErrAgentCannotApprove)
}

func TestWithdrawInvoiceDispute_WithdrawsWithItsNotes(t *testing.T) {
	t.Parallel()

	plan := resolvedDisputePlan()
	plan.After.Status = invoice.DisputeCaseStatusWithdrawn
	plan.After.Resolution = ""
	disputer := &fakeDisputer{preview: plan, guard: &writeGuard{}}
	tool := newWithdrawInvoiceDisputeTool(disputer)
	params := executeParams(map[string]any{
		paramDisputeID:    plan.Before.ID.String(),
		paramDisputeNotes: "Paid in full on the 12th",
	})

	preview := previewWithoutWrites(t, disputer.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would withdraw")
	assert.Equal(t, "Withdrawn", fieldByPath(t, previewChange(t, preview, 0), "status").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, disputer.withdrawn)
	assert.Equal(t, "Paid in full on the 12th", disputer.withdrawn.Notes)
	assert.Equal(t, permission.OpCancel, tool.Policy().Operation)
}
