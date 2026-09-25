package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckAccountingConnection_PreviewMatchesWhatASuccessSaves(t *testing.T) {
	t.Parallel()

	status := connectedStatus(accountingsync.ConnectionStatusDegraded)
	status.Connection.ConsecutiveFailures = 2
	status.Connection.LastErrorMessage = "timeout"
	before := *status.Connection
	checker := &fakeAccountingChecker{status: status}
	tool := newCheckAccountingConnectionTool(checker).(*checkAccountingConnectionTool)
	params := executeParams(map[string]any{"system": "QuickBooksOnline"})

	preview := previewWithoutWrites(t, &checker.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.True(t, preview.Partial)
	assert.Contains(t, preview.Summary, "Acme Freight")
	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceAccountingIntegration, change.Resource)
	assert.Equal(t, "Connected", fieldByPath(t, change, "status").After)
	assert.True(t, fieldByPath(t, change, "lastCheckedAt").Volatile)
	assert.Empty(t, checker.checked)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, status.Connection,
		toolpreview.Only(accountingCheckFields...),
		toolpreview.Volatile(accountingCheckVolatileFields...))
}

func TestCheckAccountingConnection_PreviewWarnsWhenNotConnected(t *testing.T) {
	t.Parallel()

	checker := &fakeAccountingChecker{status: connectedStatus(accountingsync.ConnectionStatusRevoked)}
	tool := newCheckAccountingConnectionTool(checker).(*checkAccountingConnectionTool)

	preview := previewWithoutWrites(t, &checker.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{"system": "QuickBooksOnline"}))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestSetAccountingMapping_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	before := *row
	writer := &fakeMappingWriter{row: row, refs: []*accountingsync.AccountingReferenceObject{
		customerRef("51", "Acme Logistics LLC", true),
	}}
	tool := newSetAccountingMappingTool(writer).(*setAccountingMappingTool)
	params := executeParams(map[string]any{
		"system":     "QuickBooksOnline",
		"mappingId":  row.ID.String(),
		"externalId": "51",
		"reason":     "The LLC is the entity Acme invoices under.",
	})

	preview := previewWithoutWrites(t, &writer.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, "Acme Logistics", change.Label)
	mapped := fieldByPath(t, change, "externalName")
	assert.Equal(t, "Mapped to", mapped.Label)
	assert.Equal(t, "Acme Logistics, Inc.", mapped.Before)
	assert.Equal(t, "Acme Logistics LLC", mapped.After)
	assert.Equal(t, "Confirmed", fieldByPath(t, change, "state").After)
	assert.Nil(t, writer.set)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, writer.row, mappingOptions()...)
}

func TestClearAccountingMapping_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	row.State = accountingsync.MappingStateConfirmed
	before := *row
	writer := &fakeMappingWriter{row: row}
	tool := newClearAccountingMappingTool(writer).(*clearAccountingMappingTool)
	params := executeParams(map[string]any{
		"system":    "QuickBooksOnline",
		"mappingId": row.ID.String(),
	})

	preview := previewWithoutWrites(t, &writer.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, "Unmatched", fieldByPath(t, change, "state").After)
	assert.Equal(t, "Acme Logistics, Inc.", fieldByPath(t, change, "externalName").Before)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, writer.row, mappingOptions()...)
}

func TestAccountingMapping_PreviewWarnsWhenDocumentsWereAlreadySent(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	row.State = accountingsync.MappingStateConfirmed
	writer := &fakeMappingWriter{
		row:  row,
		used: 4,
		refs: []*accountingsync.AccountingReferenceObject{
			customerRef("51", "Acme Logistics LLC", true),
		},
	}
	setTool := newSetAccountingMappingTool(writer).(*setAccountingMappingTool)
	clearTool := newClearAccountingMappingTool(writer).(*clearAccountingMappingTool)

	for _, preview := range []*agent.ToolPreview{
		previewWithoutWrites(t, &writer.guard, func() (*agent.ToolPreview, error) {
			return setTool.Preview(t.Context(), executeParams(map[string]any{
				"system":     "QuickBooksOnline",
				"mappingId":  row.ID.String(),
				"externalId": "51",
				"reason":     "The LLC is the entity Acme invoices under.",
			}))
		}),
		previewWithoutWrites(t, &writer.guard, func() (*agent.ToolPreview, error) {
			return clearTool.Preview(t.Context(), executeParams(map[string]any{
				"system":    "QuickBooksOnline",
				"mappingId": row.ID.String(),
			}))
		}),
	} {
		assert.Empty(t, preview.Changes)
		requireWarning(t, preview, agent.PreviewWarningWouldFail)
		assert.Contains(t, preview.Warnings[0].Message, "already sent")
	}
}

func TestCreateAccountingReferenceRecord_PreviewShowsTheRecordAndTheMapping(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	row.State = accountingsync.MappingStateUnmatched
	row.ExternalName = ""
	row.ExternalID = ""
	before := *row
	writer := &fakeMappingWriter{row: row}
	tool := newCreateAccountingReferenceRecordTool(writer).(*createAccountingReferenceRecordTool)
	params := executeParams(map[string]any{
		"system":    "QuickBooksOnline",
		"mappingId": row.ID.String(),
		"name":      "Acme Logistics",
	})

	preview := previewWithoutWrites(t, &writer.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.True(t, preview.Partial)
	require.Len(t, preview.Changes, 2)
	created := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)
	assert.Equal(t, "Acme Logistics", fieldByPath(t, created, "name").After)
	assert.Equal(t, "Customer", fieldByPath(t, created, "kind").After)
	mapping := previewChange(t, preview, 1)
	assert.Equal(t, "Acme Logistics", fieldByPath(t, mapping, "externalName").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, mapping, &before, writer.row, mappingOptions()...)
}

func TestRefreshAccountingReferenceData_PreviewNamesTheConnection(t *testing.T) {
	t.Parallel()

	writer := &fakeMappingWriter{summary: &serviceports.AccountingMappingSummary{
		ProviderName: "QuickBooks Online",
		Connection: &accountingsync.AccountingConnection{
			ID:                  pulid.MustNew("acctc_"),
			Status:              accountingsync.ConnectionStatusConnected,
			ExternalCompanyName: "Acme Freight",
		},
	}}
	tool := newRefreshAccountingReferenceDataTool(writer).(*refreshAccountingReferenceDataTool)

	preview := previewWithoutWrites(t, &writer.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{"system": "QuickBooksOnline"}))
	})

	assert.True(t, preview.Partial)
	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationRun, change.Operation)
	assert.Equal(t, writer.summary.Connection.ID, change.EntityID)
	assert.Contains(t, preview.Summary, "Acme Freight")
	assert.Nil(t, writer.refreshed)
}
