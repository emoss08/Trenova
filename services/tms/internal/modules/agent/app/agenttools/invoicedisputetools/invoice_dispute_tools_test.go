package invoicedisputetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDisputer struct {
	preview   *invoiceports.InvoiceDisputePreview
	refusal   error
	guard     *agenttoolstest.WriteGuard
	opened    *invoiceports.OpenInvoiceDisputeRequest
	resolved  *invoiceports.ResolveInvoiceDisputeRequest
	withdrawn *invoiceports.WithdrawInvoiceDisputeRequest
}

func (f *fakeDisputer) plan() (*invoiceports.InvoiceDisputePreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.preview, nil
}

func (f *fakeDisputer) Open(
	_ context.Context,
	req *invoiceports.OpenInvoiceDisputeRequest,
	_ *permissionports.RequestActor,
) (*invoicedispute.InvoiceDispute, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.opened = req

	return f.preview.After, nil
}

func (f *fakeDisputer) Resolve(
	_ context.Context,
	req *invoiceports.ResolveInvoiceDisputeRequest,
	_ *permissionports.RequestActor,
) (*invoicedispute.InvoiceDispute, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.resolved = req

	return f.preview.After, nil
}

func (f *fakeDisputer) Withdraw(
	_ context.Context,
	req *invoiceports.WithdrawInvoiceDisputeRequest,
	_ *permissionports.RequestActor,
) (*invoicedispute.InvoiceDispute, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.withdrawn = req

	return f.preview.After, nil
}

func (f *fakeDisputer) PreviewOpen(
	context.Context,
	*invoiceports.OpenInvoiceDisputeRequest,
	*permissionports.RequestActor,
) (*invoiceports.InvoiceDisputePreview, error) {
	return f.plan()
}

func (f *fakeDisputer) PreviewResolve(
	context.Context,
	*invoiceports.ResolveInvoiceDisputeRequest,
	*permissionports.RequestActor,
) (*invoiceports.InvoiceDisputePreview, error) {
	return f.plan()
}

func (f *fakeDisputer) PreviewWithdraw(
	context.Context,
	*invoiceports.WithdrawInvoiceDisputeRequest,
	*permissionports.RequestActor,
) (*invoiceports.InvoiceDisputePreview, error) {
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

func openDisputePlan() *invoiceports.InvoiceDisputePreview {
	inv := disputedInvoice()
	after := *inv
	after.DisputeStatus = invoice.DisputeStatusDisputed

	return &invoiceports.InvoiceDisputePreview{
		After: &invoicedispute.InvoiceDispute{
			InvoiceID:           inv.ID,
			Status:              invoicedispute.DisputeCaseStatusOpen,
			ReasonCode:          invoicedispute.DisputeReasonAccessorialDisputed,
			DisputedAmount:      decimal.RequireFromString("150.00"),
			DisputedAmountMinor: 15000,
			Notes:               "Customer disputes the detention line",
		},
		InvoiceBefore: inv,
		InvoiceAfter:  &after,
	}
}

func openDisputeParams(invoiceID pulid.ID) *agenttoolports.ToolExecuteParams {
	return agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamInvoiceID: invoiceID.String(),
		paramDisputeReasonCode:    string(invoicedispute.DisputeReasonAccessorialDisputed),
		paramDisputedAmount:       "150.00",
		toolparams.ParamNote:      " Customer disputes the detention line ",
	})
}

func TestOpenInvoiceDispute_PreviewShowsTheCaseAndTheFlag(t *testing.T) {
	t.Parallel()

	disputer := &fakeDisputer{preview: openDisputePlan(), guard: &agenttoolstest.WriteGuard{}}
	tool := NewOpenInvoiceDisputeTool(disputer).(agenttoolports.ToolPreviewer)
	params := openDisputeParams(disputer.preview.InvoiceBefore.ID)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		disputer.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	assert.Equal(
		t,
		"Would open a dispute on Invoice INV-4100 for 150.00 USD (AccessorialDisputed).",
		preview.Summary,
	)
	require.Len(t, preview.Changes, 2)
	created := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceInvoiceDispute, created.Resource)
	assert.Equal(t, agentpreview.PreviewOperationCreate, created.Operation)
	assert.Equal(t, "Open", agenttoolstest.FieldByPath(t, created, "status").After)
	require.NotNil(t, created.Money)
	flag := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceInvoice, flag.Resource)
	assert.Equal(t, "Disputed", agenttoolstest.FieldByPath(t, flag, fieldDisputeStatus).After)
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
	tool := NewOpenInvoiceDisputeTool(disputer)
	params := openDisputeParams(disputer.preview.InvoiceBefore.ID)

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
	assert.Empty(t, preview.Changes)

	err = tool.(agenttoolports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the open balance")
}

func TestOpenInvoiceDispute_RefusesBadArguments(t *testing.T) {
	t.Parallel()

	tool := NewOpenInvoiceDisputeTool(&fakeDisputer{preview: openDisputePlan()})
	invoiceID := pulid.MustNew("inv_").String()
	cases := map[string]map[string]any{
		"an unknown reason": {
			toolparams.ParamInvoiceID: invoiceID, paramDisputeReasonCode: "TooExpensive",
			paramDisputedAmount: "10",
		},
		"a negative amount": {
			toolparams.ParamInvoiceID: invoiceID, paramDisputeReasonCode: "Other",
			paramDisputedAmount: "-10",
		},
		"no invoice": {paramDisputeReasonCode: "Other", paramDisputedAmount: "10"},
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
				agenttoolstest.ExecuteParams(raw)))
		})
	}
}

func TestOpenInvoiceDispute_OpensAsTheActorWithTrimmedNotes(t *testing.T) {
	t.Parallel()

	disputer := &fakeDisputer{preview: openDisputePlan()}
	tool := NewOpenInvoiceDisputeTool(disputer)
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
	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceInvoice, target.Resource)
}

func resolvedDisputePlan() *invoiceports.InvoiceDisputePreview {
	plan := openDisputePlan()
	plan.InvoiceBefore.DisputeStatus = invoice.DisputeStatusDisputed
	plan.InvoiceAfter.DisputeStatus = invoice.DisputeStatusNone
	before := *plan.After
	before.ID = pulid.MustNew("idsp_")
	before.Version = 1
	after := before
	after.Status = invoicedispute.DisputeCaseStatusResolved
	after.Resolution = invoicedispute.DisputeResolutionInvoiceUpheld
	after.ResolutionNotes = "Detention was on the rate confirmation"
	plan.Before = &before
	plan.After = &after

	return plan
}

func TestResolveInvoiceDispute_OnlyAPersonResolves(t *testing.T) {
	t.Parallel()

	disputer := &fakeDisputer{preview: resolvedDisputePlan(), guard: &agenttoolstest.WriteGuard{}}
	tool := NewResolveInvoiceDisputeTool(disputer)
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamDisputeID: disputer.preview.Before.ID.String(),
		paramDisputeResolution:    string(invoicedispute.DisputeResolutionInvoiceUpheld),
		paramResolutionNotes:      "Detention was on the rate confirmation",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		disputer.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Resolved", agenttoolstest.FieldByPath(t, change, "status").After)
	assert.Equal(
		t,
		"InvoiceUpheld",
		agenttoolstest.FieldByPath(t, change, paramDisputeResolution).After,
	)
	assert.Equal(
		t,
		"None",
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 1),
			fieldDisputeStatus,
		).After,
	)

	require.ErrorIs(t, tool.Execute(t.Context(), params), toolspec.ErrNeedsAPersonsApproval)
	assert.Nil(t, disputer.resolved)

	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, disputer.resolved)
	assert.Equal(t, invoicedispute.DisputeResolutionInvoiceUpheld, disputer.resolved.Resolution)

	policy := tool.Policy()
	assert.Equal(t, permission.OpApprove, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceInvoiceDispute, target.Resource)
}

func TestResolveInvoiceDispute_AnAgentPrincipalCannotEvenPreviewIt(t *testing.T) {
	t.Parallel()

	disputer := &fakeDisputer{preview: resolvedDisputePlan()}
	tool := NewResolveInvoiceDisputeTool(disputer).(agenttoolports.ToolPreviewer)
	params := agenttoolstest.AgentParamsFor(map[string]any{
		toolparams.ParamDisputeID: disputer.preview.Before.ID.String(),
		paramDisputeResolution:    string(invoicedispute.DisputeResolutionInvoiceUpheld),
	})

	_, err := tool.Preview(t.Context(), params)
	require.ErrorIs(t, err, toolguard.ErrAgentCannotApprove)
}

func TestWithdrawInvoiceDispute_WithdrawsWithItsNotes(t *testing.T) {
	t.Parallel()

	plan := resolvedDisputePlan()
	plan.After.Status = invoicedispute.DisputeCaseStatusWithdrawn
	plan.After.Resolution = ""
	disputer := &fakeDisputer{preview: plan, guard: &agenttoolstest.WriteGuard{}}
	tool := NewWithdrawInvoiceDisputeTool(disputer)
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamDisputeID: plan.Before.ID.String(),
		toolparams.ParamNote:      "Paid in full on the 12th",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		disputer.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would withdraw")
	assert.Equal(
		t,
		"Withdrawn",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "status").After,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, disputer.withdrawn)
	assert.Equal(t, "Paid in full on the 12th", disputer.withdrawn.Notes)
	assert.Equal(t, permission.OpCancel, tool.Policy().Operation)
}
