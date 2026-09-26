package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLateCharges struct {
	guard    *writeGuard
	plan     *serviceports.LateChargeAssessmentResult
	refusal  error
	assessed *serviceports.LateChargeAssessmentRequest
}

func (f *fakeLateCharges) Assess(
	_ context.Context,
	req *serviceports.LateChargeAssessmentRequest,
	_ *serviceports.RequestActor,
) (*serviceports.LateChargeAssessmentResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.assessed = req

	return f.plan, nil
}

func (f *fakeLateCharges) PlanAssess(
	context.Context,
	*serviceports.LateChargeAssessmentRequest,
	*serviceports.RequestActor,
) (*serviceports.LateChargeAssessmentResult, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.plan, nil
}

func lateChargePlan() *serviceports.LateChargeAssessmentResult {
	return &serviceports.LateChargeAssessmentResult{
		Preview:          true,
		Mode:             tenant.LateChargeAssessmentModeAutomatic,
		AutoPost:         true,
		TotalChargeMinor: 1500,
		Customers: []*serviceports.LateChargeCustomerResult{
			{
				CustomerID:       pulid.MustNew("cus_"),
				CustomerName:     "Acme Foods",
				CurrencyCode:     "USD",
				TotalChargeMinor: 1500,
				Lines: []*serviceports.LateChargeLine{
					{InvoiceNumber: "INV-1", ChargeMinor: 750},
					{InvoiceNumber: "INV-2", ChargeMinor: 750},
				},
			},
			{
				CustomerID:   pulid.MustNew("cus_"),
				CustomerName: "Beta Farms",
				Skipped:      true,
				SkipReason:   "Every overdue period has already been assessed",
			},
		},
	}
}

func TestAssessLateCharges_PreviewsEachMemoAndOnlyAPersonAssesses(t *testing.T) {
	t.Parallel()

	charges := &fakeLateCharges{plan: lateChargePlan(), guard: &writeGuard{}}
	tool := newAssessLateChargesTool(charges)
	params := executeParams(map[string]any{paramAsOfDate: "2026-09-25"})

	preview := previewWithoutWrites(t, charges.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would raise 1 late charge debit memo for 15.00")
	assert.Contains(t, preview.Summary, "posted as it is raised")
	assert.Contains(t, preview.Summary, "Beta Farms is skipped")
	require.Len(t, preview.Changes, 1)
	memo := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceInvoice, memo.Resource)
	assert.Equal(t, agent.PreviewOperationCreate, memo.Operation)
	require.NotNil(t, memo.Money)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, charges.assessed)
	assert.False(t, charges.assessed.Preview)
	assert.Equal(t, time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC).Unix(),
		charges.assessed.AsOfDate)

	policy := tool.Policy()
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
}

func TestAssessLateCharges_RefusesARunThatChargesNothing(t *testing.T) {
	t.Parallel()

	plan := lateChargePlan()
	plan.Customers = plan.Customers[1:]
	plan.TotalChargeMinor = 0
	tool := newAssessLateChargesTool(&fakeLateCharges{plan: plan})
	params := executeParams(map[string]any{paramAsOfDate: "2026-09-25"})

	err := tool.(serviceports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to charge")

	disabled := &fakeLateCharges{refusal: errortypes.NewValidationError(
		"mode", errortypes.ErrInvalidOperation, "Late charge assessment is disabled",
	)}
	preview, err := newAssessLateChargesTool(disabled).(serviceports.ToolPreviewer).
		Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

type fakeSharer struct {
	guard   *writeGuard
	plan    *serviceports.InvoiceSharePreview
	shared  *serviceports.ShareInvoiceRequest
	refusal error
}

func (f *fakeSharer) Share(
	_ context.Context,
	req *serviceports.ShareInvoiceRequest,
	_ *serviceports.RequestActor,
) (*serviceports.ShareInvoiceResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.shared = req

	return &serviceports.ShareInvoiceResult{}, nil
}

func (f *fakeSharer) PreviewShare(
	context.Context,
	*serviceports.ShareInvoiceRequest,
	*serviceports.RequestActor,
) (*serviceports.InvoiceSharePreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.plan, nil
}

func TestShareInvoice_PreviewsWhatEachTeammateReceives(t *testing.T) {
	t.Parallel()

	inv := receivableInvoice("INV-2026-1042", 100000, 0)
	dana := pulid.MustNew("usr_")
	sharer := &fakeSharer{guard: &writeGuard{}, plan: &serviceports.InvoiceSharePreview{
		Invoice:         inv,
		SharedByName:    "Marcus Bell",
		Note:            "Check the detention line",
		Tab:             invoice.ShareTabCharges,
		EmailConfigured: true,
		Recipients: []serviceports.InvoiceShareRecipientPreview{{
			UserID: dana, Name: "Dana Whitfield", EmailAddress: "dana@example.com",
			Emailed: true, Subject: "Marcus shared an invoice", Body: "Check the detention line",
		}},
	}}
	tool := newShareInvoiceTool(sharer)
	params := executeParams(map[string]any{
		paramInvoiceID: inv.ID.String(),
		paramUserIDs:   []any{dana.String()},
		paramShareNote: "Check the detention line",
		paramShareTab:  string(invoice.ShareTabCharges),
	})

	preview := previewWithoutWrites(t, sharer.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would share Invoice INV-2026-1042 with Dana Whitfield")
	change := previewChange(t, preview, 0)
	require.NotNil(t, change.Message)
	assert.Equal(t, []string{"dana@example.com"}, change.Message.To)
	assert.Equal(t, "Marcus shared an invoice", change.Message.Subject)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, sharer.shared)
	assert.Equal(t, []pulid.ID{dana}, sharer.shared.UserIDs)
	assert.Equal(t, invoice.ShareTabCharges, sharer.shared.Tab)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceInvoice, policy.Resource)
	assert.Equal(t, permission.OpRead, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
}
