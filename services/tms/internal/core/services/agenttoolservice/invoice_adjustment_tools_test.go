package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAdjuster struct {
	guard     *writeGuard
	figures   *serviceports.InvoiceAdjustmentPreview
	draft     *serviceports.InvoiceAdjustmentDraftPreview
	detail    *invoiceadjustment.InvoiceAdjustment
	decision  *serviceports.InvoiceAdjustmentDecisionPreview
	saved     *serviceports.SaveInvoiceAdjustmentDraftRequest
	submitted []*serviceports.InvoiceAdjustmentRequest
	bulk      *serviceports.InvoiceAdjustmentBulkRequest
	drafted   *serviceports.GetInvoiceAdjustmentDetailRequest
	approved  *serviceports.ApproveInvoiceAdjustmentRequest
	rejected  *serviceports.RejectInvoiceAdjustmentRequest
}

func (f *fakeAdjuster) SaveDraft(
	_ context.Context,
	req *serviceports.SaveInvoiceAdjustmentDraftRequest,
	_ *serviceports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.saved = req
	saved := *f.draft.After
	saved.ID = pulid.MustNew("iadj_")

	return &saved, nil
}

func (f *fakeAdjuster) PreviewSaveDraft(
	context.Context,
	*serviceports.SaveInvoiceAdjustmentDraftRequest,
	*serviceports.RequestActor,
) (*serviceports.InvoiceAdjustmentDraftPreview, error) {
	return f.draft, nil
}

func (f *fakeAdjuster) GetDetail(
	context.Context,
	*serviceports.GetInvoiceAdjustmentDetailRequest,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	return f.detail, nil
}

func (f *fakeAdjuster) PreviewDraft(
	context.Context,
	*serviceports.GetInvoiceAdjustmentDetailRequest,
	*serviceports.RequestActor,
) (*serviceports.InvoiceAdjustmentPreview, error) {
	return f.figures, nil
}

func (f *fakeAdjuster) SubmitDraft(
	_ context.Context,
	req *serviceports.GetInvoiceAdjustmentDetailRequest,
	_ *serviceports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.drafted = req

	return f.detail, nil
}

func (f *fakeAdjuster) Preview(
	_ context.Context,
	req *serviceports.InvoiceAdjustmentRequest,
	_ *serviceports.RequestActor,
) (*serviceports.InvoiceAdjustmentPreview, error) {
	figures := *f.figures
	figures.InvoiceID = req.InvoiceID
	figures.Kind = req.Kind

	return &figures, nil
}

func (f *fakeAdjuster) Submit(
	_ context.Context,
	req *serviceports.InvoiceAdjustmentRequest,
	_ *serviceports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.submitted = append(f.submitted, req)

	return &invoiceadjustment.InvoiceAdjustment{
		ID:                pulid.MustNew("iadj_"),
		OriginalInvoiceID: req.InvoiceID,
		Status:            invoiceadjustment.StatusPendingApproval,
	}, nil
}

func (f *fakeAdjuster) BulkPreview(
	ctx context.Context,
	req *serviceports.InvoiceAdjustmentBulkRequest,
	actor *serviceports.RequestActor,
) ([]*serviceports.InvoiceAdjustmentPreview, error) {
	previews := make([]*serviceports.InvoiceAdjustmentPreview, 0, len(req.Items))
	for _, item := range req.Items {
		preview, err := f.Preview(ctx, item, actor)
		if err != nil {
			return nil, err
		}
		previews = append(previews, preview)
	}

	return previews, nil
}

func (f *fakeAdjuster) BulkSubmit(
	_ context.Context,
	req *serviceports.InvoiceAdjustmentBulkRequest,
	_ *serviceports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.bulk = req

	return &invoiceadjustment.InvoiceAdjustmentBatch{ID: pulid.MustNew("iadjb_")}, nil
}

func (f *fakeAdjuster) Approve(
	_ context.Context,
	req *serviceports.ApproveInvoiceAdjustmentRequest,
	_ *serviceports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.approved = req

	return f.decision.Adjustment, nil
}

func (f *fakeAdjuster) Reject(
	_ context.Context,
	req *serviceports.RejectInvoiceAdjustmentRequest,
	_ *serviceports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.rejected = req

	return f.decision.Adjustment, nil
}

func (f *fakeAdjuster) PreviewDecision(
	context.Context,
	*serviceports.InvoiceAdjustmentDecisionRequest,
) (*serviceports.InvoiceAdjustmentDecisionPreview, error) {
	return f.decision, nil
}

func adjustmentFigures(approval bool) *serviceports.InvoiceAdjustmentPreview {
	return &serviceports.InvoiceAdjustmentPreview{
		Kind:              invoiceadjustment.KindCreditOnly,
		CreditTotalAmount: decimal.RequireFromString("125.00"),
		RebillTotalAmount: decimal.Zero,
		NetDeltaAmount:    decimal.RequireFromString("-125.00"),
		RequiresApproval:  approval,
		Errors:            map[string][]string{},
	}
}

func adjustedInvoice() *invoice.Invoice {
	inv := receivableInvoice("INV-5100", 90000, 0)
	return inv
}

func TestSubmitInvoiceAdjustment_OneAdjustmentPreviewsItsFigures(t *testing.T) {
	t.Parallel()

	adjuster := &fakeAdjuster{figures: adjustmentFigures(true), guard: &writeGuard{}}
	tool := newSubmitInvoiceAdjustmentTool(adjuster)
	invoiceID := pulid.MustNew("inv_")
	params := executeParams(map[string]any{
		paramAdjustments: []any{map[string]any{
			paramInvoiceID:      invoiceID.String(),
			paramAdjustmentKind: string(invoiceadjustment.KindCreditOnly),
			paramReason:         "Detention was billed twice",
			paramAdjustmentLines: []any{map[string]any{
				paramInvoiceLineID: pulid.MustNew("invl_").String(),
				paramCreditAmount:  "125.00",
			}},
		}},
	})

	preview := previewWithoutWrites(t, adjuster.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would submit 1 invoice adjustment")
	assert.Contains(t, preview.Summary, "waits for approval")
	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceInvoice, change.Resource)
	assert.Equal(t, "PendingApproval", fieldByPath(t, change, "status").After)
	require.NotNil(t, change.Money)

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, invoiceID, target.ID)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	params.IdempotencyKey = params.ProposalID.String()
	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, adjuster.submitted, 1)
	submitted := adjuster.submitted[0]
	assert.Equal(t, "Detention was billed twice", submitted.Reason)
	assert.Equal(t, invoiceadjustment.RebillStrategyCloneExact, submitted.RebillStrategy)
	assert.Contains(t, submitted.IdempotencyKey, params.ProposalID.String())
	require.Len(t, submitted.Lines, 1)
	assert.True(t, submitted.Lines[0].CreditAmount.Equal(decimal.RequireFromString("125")))

	policy := tool.Policy()
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.True(t, policy.Idempotent)
}

func TestSubmitInvoiceAdjustment_ManyGoAsOneBatch(t *testing.T) {
	t.Parallel()

	adjuster := &fakeAdjuster{figures: adjustmentFigures(false)}
	tool := newSubmitInvoiceAdjustmentTool(adjuster)
	items := make([]any, 0, 3)
	for range 3 {
		items = append(items, map[string]any{
			paramInvoiceID:      pulid.MustNew("inv_").String(),
			paramAdjustmentKind: string(invoiceadjustment.KindWriteOff),
			paramReason:         "Small balance write-off",
		})
	}
	params := executeParams(map[string]any{paramAdjustments: items})
	params.ProposalID = pulid.MustNew("ap_")
	params.IdempotencyKey = params.ProposalID.String()

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "Would submit 3 invoice adjustments")
	assert.Len(t, preview.Changes, 3)
	_, targeted := tool.(serviceports.TargetedTool).Target(params.Params)
	assert.False(t, targeted)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, adjuster.bulk)
	require.Len(t, adjuster.bulk.Items, 3)
	assert.NotEqual(t, adjuster.bulk.Items[0].IdempotencyKey, adjuster.bulk.Items[1].IdempotencyKey)
	assert.Empty(t, adjuster.submitted)
}

func TestSubmitInvoiceAdjustment_PreviewErrorsRefuseIt(t *testing.T) {
	t.Parallel()

	figures := adjustmentFigures(false)
	figures.Errors = map[string][]string{"invoiceId": {"Only posted invoices may be adjusted"}}
	adjuster := &fakeAdjuster{figures: figures}
	tool := newSubmitInvoiceAdjustmentTool(adjuster)
	params := executeParams(map[string]any{
		paramAdjustments: []any{map[string]any{
			paramInvoiceID:      pulid.MustNew("inv_").String(),
			paramAdjustmentKind: string(invoiceadjustment.KindCreditOnly),
			paramReason:         "Wrong rate",
		}},
	})

	err := tool.(serviceports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only posted invoices may be adjusted")
	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestSubmitInvoiceAdjustment_TakesADraftOrAdjustmentsNotBoth(t *testing.T) {
	t.Parallel()

	tool := newSubmitInvoiceAdjustmentTool(&fakeAdjuster{figures: adjustmentFigures(false)})
	both := executeParams(map[string]any{
		paramDraftAdjustmentID: pulid.MustNew("iadj_").String(),
		paramAdjustments: []any{map[string]any{
			paramInvoiceID:      pulid.MustNew("inv_").String(),
			paramAdjustmentKind: string(invoiceadjustment.KindCreditOnly),
			paramReason:         "x",
		}},
	})
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), both))
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(map[string]any{})))
}

func TestSubmitInvoiceAdjustment_SubmitsASavedDraft(t *testing.T) {
	t.Parallel()

	detail := &invoiceadjustment.InvoiceAdjustment{
		ID:                pulid.MustNew("iadj_"),
		OriginalInvoiceID: pulid.MustNew("inv_"),
		Kind:              invoiceadjustment.KindCreditOnly,
		Status:            invoiceadjustment.StatusDraft,
		Reason:            "Fuel surcharge disputed",
	}
	adjuster := &fakeAdjuster{figures: adjustmentFigures(false), detail: detail}
	tool := newSubmitInvoiceAdjustmentTool(adjuster)
	params := executeParams(map[string]any{paramDraftAdjustmentID: detail.ID.String()})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "executes at once")

	params.ProposalID = pulid.MustNew("ap_")
	params.IdempotencyKey = params.ProposalID.String()
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, adjuster.drafted)
	assert.Equal(t, detail.ID, adjuster.drafted.AdjustmentID)
}

func TestSaveInvoiceAdjustmentDraft_AnAgentSavesADraftAndSaysWhere(t *testing.T) {
	t.Parallel()

	inv := adjustedInvoice()
	adjuster := &fakeAdjuster{
		guard: &writeGuard{},
		draft: &serviceports.InvoiceAdjustmentDraftPreview{
			After: &invoiceadjustment.InvoiceAdjustment{
				OriginalInvoiceID: inv.ID,
				Kind:              invoiceadjustment.KindCreditOnly,
				Status:            invoiceadjustment.StatusDraft,
				RebillStrategy:    invoiceadjustment.RebillStrategyCloneExact,
				Reason:            "Customer disputes the lumper fee",
			},
			Invoice: inv,
			Figures: adjustmentFigures(false),
		},
	}
	tool := newSaveInvoiceAdjustmentDraftTool(adjuster)
	params := agentParamsFor(map[string]any{
		paramInvoiceID:      inv.ID.String(),
		paramAdjustmentKind: string(invoiceadjustment.KindCreditOnly),
		paramReason:         "Customer disputes the lumper fee",
	})

	preview := previewWithoutWrites(t, adjuster.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would save a draft CreditOnly adjustment of Invoice INV-5100")
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 0).Operation)

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, adjuster.saved)
	assert.Equal(t, invoiceadjustment.RebillStrategyCloneExact, adjuster.saved.RebillStrategy)
	assert.NotEmpty(t, result.IDs[paramAdjustmentID])
	require.NotNil(t, result.Record)
	assert.Equal(t, inv.ID.String(), result.Record.ID)

	policy := tool.Policy()
	assert.Equal(t, agent.TierAutoExecute, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
	assert.NotNil(t, policy.TaintHold)
}

func pendingDecision() *serviceports.InvoiceAdjustmentDecisionPreview {
	inv := adjustedInvoice()
	return &serviceports.InvoiceAdjustmentDecisionPreview{
		Adjustment: &invoiceadjustment.InvoiceAdjustment{
			ID:                pulid.MustNew("iadj_"),
			OriginalInvoiceID: inv.ID,
			Kind:              invoiceadjustment.KindCreditOnly,
			Status:            invoiceadjustment.StatusPendingApproval,
			ApprovalStatus:    invoiceadjustment.ApprovalStatusPending,
			CreditTotalAmount: decimal.RequireFromString("125.00"),
			NetDeltaAmount:    decimal.RequireFromString("-125.00"),
			ApprovalRequired:  true,
		},
		Invoice: inv,
		Figures: adjustmentFigures(true),
	}
}

func TestApproveInvoiceAdjustment_OnlyAPersonApprovesWhatExecutionWouldDo(t *testing.T) {
	t.Parallel()

	adjuster := &fakeAdjuster{decision: pendingDecision(), guard: &writeGuard{}}
	tool := newApproveInvoiceAdjustmentTool(adjuster)
	params := executeParams(map[string]any{
		paramAdjustmentID: adjuster.decision.Adjustment.ID.String(),
	})

	preview := previewWithoutWrites(t, adjuster.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would approve the CreditOnly adjustment of Invoice INV-5100")
	change := previewChange(t, preview, 0)
	assert.Equal(t, "Executed", fieldByPath(t, change, "status").After)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, adjuster.approved)

	agentParams := agentParamsFor(params.Params)
	_, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), agentParams)
	require.ErrorIs(t, err, ErrAgentCannotApprove)
	assert.Equal(t, permission.OpApprove, tool.Policy().Operation)
}

func TestApproveInvoiceAdjustment_ARevalidationFailureRefusesIt(t *testing.T) {
	t.Parallel()

	decision := pendingDecision()
	decision.Figures.Errors = map[string][]string{
		"lines[0].creditAmount": {"Credit exceeds what is left of the line"},
	}
	tool := newApproveInvoiceAdjustmentTool(&fakeAdjuster{decision: decision})
	params := executeParams(map[string]any{paramAdjustmentID: decision.Adjustment.ID.String()})

	err := tool.(serviceports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Credit exceeds what is left of the line")
}

func TestRejectInvoiceAdjustment_RecordsTheReason(t *testing.T) {
	t.Parallel()

	adjuster := &fakeAdjuster{decision: pendingDecision(), guard: &writeGuard{}}
	tool := newRejectInvoiceAdjustmentTool(adjuster)
	params := executeParams(map[string]any{
		paramAdjustmentID: adjuster.decision.Adjustment.ID.String(),
		paramReason:       "The customer agreed to pay the lumper fee",
	})

	preview := previewWithoutWrites(t, adjuster.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Rejected", fieldByPath(t, previewChange(t, preview, 0), "status").After)

	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, adjuster.rejected)
	assert.Equal(t, "The customer agreed to pay the lumper fee", adjuster.rejected.Reason)
}
