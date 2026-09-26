package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
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
			tool := newResolveBankReceiptWorkItemTool(items).(*resolveBankReceiptWorkItemTool)
			params := memoryParams(map[string]any{
				"workItemId": item.ID.String(),
				"resolution": resolution,
				"note":       "Bank interest; not a customer payment.",
			})

			preview := previewWithoutWrites(t, &items.guard, func() (*agent.ToolPreview, error) {
				return tool.Preview(t.Context(), params)
			})

			change := previewChange(t, preview, 0)
			assert.Equal(t, permission.ResourceBankReceiptWorkItem, change.Resource)
			assert.Equal(t, agent.PreviewOperationArchive, change.Operation)
			assert.Equal(t, resolution, fieldByPath(t, change, "resolutionType").After)
			assert.Equal(t, "Bank interest; not a customer payment.",
				fieldByPath(t, change, "resolutionNote").After)

			require.NoError(t, tool.Execute(t.Context(), params))
			requireUpdateParity(t, change, &before, item,
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
	tool := newResolveBankReceiptWorkItemTool(items).(*resolveBankReceiptWorkItemTool)

	preview := previewWithoutWrites(t, &items.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), memoryParams(map[string]any{
			"workItemId": item.ID.String(),
			"resolution": "MarkedFalsePositive",
			"note":       "x",
		}))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}
