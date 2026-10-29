package accountingtools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckAccountingConnection_PreviewMatchesWhatASuccessSaves(t *testing.T) {
	t.Parallel()

	status := connectedStatus(accountingconnection.ConnectionStatusDegraded)
	status.Connection.ConsecutiveFailures = 2
	status.Connection.LastErrorMessage = "timeout"
	before := *status.Connection
	checker := &fakeAccountingChecker{status: status}
	tool := newCheckAccountingConnectionTool(checker, checker).(*CheckAccountingConnectionTool)
	params := agenttoolstest.ExecuteParams(map[string]any{"system": "QuickBooksOnline"})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&checker.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	assert.True(t, preview.Partial)
	assert.Contains(t, preview.Summary, "Acme Freight")
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceAccountingIntegration, change.Resource)
	assert.Equal(t, "Connected", agenttoolstest.FieldByPath(t, change, "status").After)
	assert.True(t, agenttoolstest.FieldByPath(t, change, "lastCheckedAt").Volatile)
	assert.Empty(t, checker.checked)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, status.Connection,
		toolpreview.Only(accountingCheckFields...),
		toolpreview.Volatile(accountingCheckVolatileFields...))
}

func TestCheckAccountingConnection_PreviewWarnsWhenNotConnected(t *testing.T) {
	t.Parallel()

	checker := &fakeAccountingChecker{
		status: connectedStatus(accountingconnection.ConnectionStatusRevoked),
	}
	tool := newCheckAccountingConnectionTool(checker, checker).(*CheckAccountingConnectionTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&checker.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(
				t.Context(),
				agenttoolstest.ExecuteParams(map[string]any{"system": "QuickBooksOnline"}),
			)
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}

func TestSetAccountingMapping_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	before := *row
	writer := &fakeMappingWriter{row: row, refs: []*accountingmapping.AccountingReferenceObject{
		customerRef("51", "Acme Logistics LLC", true),
	}}
	tool := newSetAccountingMappingTool(writer, writer, writer).(*SetAccountingMappingTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"system":     "QuickBooksOnline",
		"mappingId":  row.ID.String(),
		"externalId": "51",
		"reason":     "The LLC is the entity Acme invoices under.",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&writer.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Acme Logistics", change.Label)
	mapped := agenttoolstest.FieldByPath(t, change, "externalName")
	assert.Equal(t, "Mapped to", mapped.Label)
	assert.Equal(t, "Acme Logistics, Inc.", mapped.Before)
	assert.Equal(t, "Acme Logistics LLC", mapped.After)
	assert.Equal(t, "Confirmed", agenttoolstest.FieldByPath(t, change, "state").After)
	assert.Nil(t, writer.set)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, writer.row, MappingOptions()...)
}

func TestClearAccountingMapping_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	row.State = accountingmapping.MappingStateConfirmed
	before := *row
	writer := &fakeMappingWriter{row: row}
	tool := newClearAccountingMappingTool(writer, writer, writer).(*ClearAccountingMappingTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"system":    "QuickBooksOnline",
		"mappingId": row.ID.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&writer.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Unmatched", agenttoolstest.FieldByPath(t, change, "state").After)
	assert.Equal(
		t,
		"Acme Logistics, Inc.",
		agenttoolstest.FieldByPath(t, change, "externalName").Before,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, writer.row, MappingOptions()...)
}

func TestAccountingMapping_PreviewWarnsWhenDocumentsWereAlreadySent(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	row.State = accountingmapping.MappingStateConfirmed
	writer := &fakeMappingWriter{
		row:  row,
		used: 4,
		refs: []*accountingmapping.AccountingReferenceObject{
			customerRef("51", "Acme Logistics LLC", true),
		},
	}
	setTool := newSetAccountingMappingTool(writer, writer, writer).(*SetAccountingMappingTool)
	clearTool := newClearAccountingMappingTool(writer, writer, writer).(*ClearAccountingMappingTool)

	for _, preview := range []*agentpreview.ToolPreview{
		agenttoolstest.PreviewWithoutWrites(t, &writer.guard, func() (*agentpreview.ToolPreview, error) {
			return setTool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"system":     "QuickBooksOnline",
				"mappingId":  row.ID.String(),
				"externalId": "51",
				"reason":     "The LLC is the entity Acme invoices under.",
			}))
		}),
		agenttoolstest.PreviewWithoutWrites(t, &writer.guard, func() (*agentpreview.ToolPreview, error) {
			return clearTool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"system":    "QuickBooksOnline",
				"mappingId": row.ID.String(),
			}))
		}),
	} {
		assert.Empty(t, preview.Changes)
		agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
		assert.Contains(t, preview.Warnings[0].Message, "already sent")
	}
}

func TestCreateAccountingReferenceRecord_PreviewShowsTheRecordAndTheMapping(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	row.State = accountingmapping.MappingStateUnmatched
	row.ExternalName = ""
	row.ExternalID = ""
	before := *row
	writer := &fakeMappingWriter{row: row}
	tool := newCreateAccountingReferenceRecordTool(writer, writer).(*CreateAccountingReferenceRecordTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"system":    "QuickBooksOnline",
		"mappingId": row.ID.String(),
		"name":      "Acme Logistics",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&writer.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	assert.True(t, preview.Partial)
	require.Len(t, preview.Changes, 2)
	created := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agentpreview.PreviewOperationCreate, created.Operation)
	assert.Equal(t, "Acme Logistics", agenttoolstest.FieldByPath(t, created, "name").After)
	assert.Equal(t, "Customer", agenttoolstest.FieldByPath(t, created, "kind").After)
	mapping := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, "Acme Logistics", agenttoolstest.FieldByPath(t, mapping, "externalName").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, mapping, &before, writer.row, MappingOptions()...)
}

func TestRefreshAccountingReferenceData_PreviewNamesTheConnection(t *testing.T) {
	t.Parallel()

	writer := &fakeMappingWriter{summary: &accountingmappingports.AccountingMappingSummary{
		ProviderName: "QuickBooks Online",
		Connection: &accountingconnection.AccountingConnection{
			ID:                  pulid.MustNew("acctc_"),
			Status:              accountingconnection.ConnectionStatusConnected,
			ExternalCompanyName: "Acme Freight",
		},
	}}
	tool := newRefreshAccountingReferenceDataTool(writer, writer).(*RefreshAccountingReferenceDataTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&writer.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(
				t.Context(),
				agenttoolstest.ExecuteParams(map[string]any{"system": "QuickBooksOnline"}),
			)
		},
	)

	assert.True(t, preview.Partial)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agentpreview.PreviewOperationRun, change.Operation)
	assert.Equal(t, writer.summary.Connection.ID, change.EntityID)
	assert.Contains(t, preview.Summary, "Acme Freight")
	assert.Nil(t, writer.refreshed)
}
