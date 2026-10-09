package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func savedDraftAdjuster() (*fakeAdjuster, *invoiceadjustment.InvoiceAdjustment) {
	inv := adjustedInvoice()
	detail := &invoiceadjustment.InvoiceAdjustment{
		ID:                pulid.MustNew("iadj_"),
		OriginalInvoiceID: inv.ID,
		Kind:              invoiceadjustment.KindCreditOnly,
		Status:            invoiceadjustment.StatusDraft,
		Reason:            "Customer disputes the lumper fee",
	}

	return &fakeAdjuster{
		guard:  &writeGuard{},
		detail: detail,
		draft: &serviceports.InvoiceAdjustmentDraftPreview{
			Before:  detail,
			After:   detail,
			Invoice: inv,
			Figures: adjustmentFigures(false),
		},
	}, detail
}

/*
A model that read a draft off its invoice sent the draft's id and the
invoice's id together, and the rewrite was refused for naming both. They
name the same thing, so the rewrite is taken.
*/
func TestSaveInvoiceAdjustmentDraft_TakesTheDraftsOwnInvoiceBesideIt(t *testing.T) {
	t.Parallel()

	adjuster, detail := savedDraftAdjuster()
	tool := newSaveInvoiceAdjustmentDraftTool(adjuster)
	params := agentParamsFor(map[string]any{
		paramAdjustmentID:   detail.ID.String(),
		paramInvoiceID:      detail.OriginalInvoiceID.String(),
		paramAdjustmentKind: string(invoiceadjustment.KindCreditOnly),
		paramReason:         "Customer disputes the lumper fee",
	})

	require.NoError(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))

	_, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, adjuster.saved)
	assert.Equal(t, detail.ID, adjuster.saved.AdjustmentID)
}

func TestSaveInvoiceAdjustmentDraft_ARefusalForADraftOnAnotherInvoiceShowsBothCalls(t *testing.T) {
	t.Parallel()

	adjuster, detail := savedDraftAdjuster()
	tool := newSaveInvoiceAdjustmentDraftTool(adjuster)
	other := pulid.MustNew("inv_")
	params := agentParamsFor(map[string]any{
		paramAdjustmentID:   detail.ID.String(),
		paramInvoiceID:      other.String(),
		paramAdjustmentKind: string(invoiceadjustment.KindCreditOnly),
		paramReason:         "Customer disputes the lumper fee",
	})

	err := tool.(serviceports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send adjustmentId without invoiceId")
	assert.Contains(t, err.Error(), "send invoiceId without adjustmentId")
	assert.Contains(t, err.Error(), other.String())

	_, err = tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.Error(t, err)
	assert.Nil(t, adjuster.saved, "a draft is never rewritten under another invoice's name")

	require.ErrorIs(t,
		tool.(serviceports.ToolValidator).Validate(t.Context(), agentParamsFor(map[string]any{
			paramAdjustmentKind: string(invoiceadjustment.KindCreditOnly),
		})),
		errNoDraftTarget,
	)
}

func TestSubmitInvoiceAdjustment_ARefusalForBothSelectionsShowsBothCalls(t *testing.T) {
	t.Parallel()

	tool := newSubmitInvoiceAdjustmentTool(&fakeAdjuster{figures: adjustmentFigures(false)})

	err := tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(map[string]any{
		paramDraftAdjustmentID: pulid.MustNew("iadj_").String(),
		paramAdjustments: []any{map[string]any{
			paramInvoiceID:      pulid.MustNew("inv_").String(),
			paramAdjustmentKind: string(invoiceadjustment.KindCreditOnly),
			paramReason:         "x",
		}},
	}))
	require.ErrorIs(t, err, errDraftAndAdjustments)
	assert.Contains(t, err.Error(), "send draftAdjustmentId without adjustments")
	assert.Contains(t, err.Error(), "send adjustments without draftAdjustmentId")

	require.ErrorIs(t,
		tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(map[string]any{})),
		errDraftOrAdjustments,
	)
}
