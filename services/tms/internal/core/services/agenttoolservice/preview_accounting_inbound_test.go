package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func previewField(change *agent.RecordChange, path string) (agent.PreviewFieldChange, bool) {
	for _, field := range change.Fields {
		if field.Path == path {
			return field, true
		}
	}
	return agent.PreviewFieldChange{}, false
}

func TestApplyAccountingInboundChange_PreviewShowsWhatItPosts(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountingsync.InboundStatusProposed)
	operator := newInboundOperator(change, &serviceports.AccountingInboundApplyPreview{
		CanApply:       true,
		PaidAt:         1_790_208_000,
		CashMinor:      145_025,
		UnappliedMinor: 5_000,
		Lines: []serviceports.AccountingInboundPostingLine{{
			ObjectType:   accountingsync.SyncObjectInvoice,
			ObjectID:     pulid.MustNew("inv_"),
			ObjectNumber: "INV-1001",
			AmountMinor:  145_025,
			OpenMinor:    145_025,
		}},
	})
	tool := newApplyAccountingInboundChangeTool(operator)
	params := syncParams(map[string]any{"inboundChangeId": change.ID.String()})

	previewer, ok := tool.(serviceports.ToolPreviewer)
	require.True(t, ok)
	preview, err := previewer.Preview(t.Context(), params)
	require.NoError(t, err)

	assert.Contains(t, preview.Summary, "Would post Payment 10442 from Acme Foods for 1500.25 USD")
	assert.Contains(t, preview.Summary, "INV-1001")
	assert.True(t, preview.Partial, "the payment it posts is not itemised field by field")
	require.Len(t, preview.Changes, 1)
	status, ok := previewField(&preview.Changes[0], "status")
	require.True(t, ok)
	assert.Equal(t, "Proposed", status.Before)
	assert.Equal(t, "Applied", status.After)
	assert.Nil(t, operator.applied, "a preview posts nothing")
	assert.Equal(t, change.Status, accountingsync.InboundStatusProposed)
}

func TestApplyAccountingInboundChange_PreviewWarnsWhenItNoLongerMatches(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountingsync.InboundStatusProposed)
	operator := newInboundOperator(change, &serviceports.AccountingInboundApplyPreview{
		Blocker: "INV-1001 was paid in Trenova after the payment was read.",
	})
	tool := newApplyAccountingInboundChangeTool(operator)
	params := syncParams(map[string]any{"inboundChangeId": change.ID.String()})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.NotEmpty(t, preview.Warnings)
	assert.Equal(t, agent.PreviewWarningWouldFail, preview.Warnings[0].Code)
	assert.Contains(t, preview.Warnings[0].Message, "was paid in Trenova")
}

func TestIgnoreAccountingInboundChange_PreviewShowsTheNote(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountingsync.InboundStatusProposed)
	operator := newInboundOperator(change, &serviceports.AccountingInboundApplyPreview{})
	tool := newIgnoreAccountingInboundChangeTool(operator)
	params := syncParams(map[string]any{
		"inboundChangeId": change.ID.String(),
		"note":            "Keyed in Trenova by the AR team.",
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, preview.Changes, 1)
	status, ok := previewField(&preview.Changes[0], "status")
	require.True(t, ok)
	assert.Equal(t, "Ignored", status.After)
	note, ok := previewField(&preview.Changes[0], "note")
	require.True(t, ok)
	assert.Equal(t, "Keyed in Trenova by the AR team.", note.After)
	assert.Nil(t, operator.ignored, "a preview ignores nothing")
}

func TestIgnoreAccountingInboundChange_PreviewWarnsOnAClosedPayment(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountingsync.InboundStatusApplied)
	operator := newInboundOperator(change, &serviceports.AccountingInboundApplyPreview{})
	tool := newIgnoreAccountingInboundChangeTool(operator)
	params := syncParams(map[string]any{
		"inboundChangeId": change.ID.String(),
		"note":            "duplicate",
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.NotEmpty(t, preview.Warnings)
	assert.Equal(t, agent.PreviewWarningWouldFail, preview.Warnings[0].Code)
}
