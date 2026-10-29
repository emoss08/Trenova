package bankreceipttools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveBankReceiptWorkItem_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	for name, resolution := range map[string]string{
		"dismissed": "MarkedFalsePositive",
		"resolved":  "RequiresExternalFollowUp",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			item := &bankreceiptworkitem.WorkItem{
				ID:      pulid.MustNew("brwi_"),
				Status:  bankreceiptworkitem.StatusOpen,
				Version: 2,
			}
			before := *item
			items := &fakeWorkItemResolver{item: item}
			tool := NewResolveBankReceiptWorkItemTool(items).(*ResolveBankReceiptWorkItemTool)
			params := agenttoolstest.MemoryParams(map[string]any{
				"workItemId": item.ID.String(),
				"resolution": resolution,
				"note":       "Bank interest; not a customer payment.",
			})

			preview := agenttoolstest.PreviewWithoutWrites(
				t,
				&items.guard,
				func() (*agentpreview.ToolPreview, error) {
					return tool.Preview(t.Context(), params)
				},
			)

			change := agenttoolstest.PreviewChange(t, preview, 0)
			assert.Equal(t, permission.ResourceBankReceiptWorkItem, change.Resource)
			assert.Equal(t, agentpreview.PreviewOperationArchive, change.Operation)
			assert.Equal(
				t,
				resolution,
				agenttoolstest.FieldByPath(t, change, "resolutionType").After,
			)
			assert.Equal(t, "Bank interest; not a customer payment.",
				agenttoolstest.FieldByPath(t, change, "resolutionNote").After)

			require.NoError(t, tool.Execute(t.Context(), params))
			agenttoolstest.RequireUpdateParity(t, change, &before, item,
				toolpreview.Only(closedWorkItemFields...), toolpreview.Volatile("resolvedAt"))
		})
	}
}

func TestResolveBankReceiptWorkItem_PreviewWarnsOnAClosedItem(t *testing.T) {
	t.Parallel()

	item := &bankreceiptworkitem.WorkItem{
		ID:     pulid.MustNew("brwi_"),
		Status: bankreceiptworkitem.StatusResolved,
	}
	items := &fakeWorkItemResolver{item: item}
	tool := NewResolveBankReceiptWorkItemTool(items).(*ResolveBankReceiptWorkItemTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&items.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.MemoryParams(map[string]any{
				"workItemId": item.ID.String(),
				"resolution": "MarkedFalsePositive",
				"note":       "x",
			}))
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}
