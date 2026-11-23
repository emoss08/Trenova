package toolartifacts

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openItem(id, number, open string) map[string]any {
	return map[string]any{
		"invoiceId": id, "invoiceNumber": number, "customerId": "cus_01JPEAK00000000000000001",
		"customer": "Peak Distributing", "settlementStatus": "Unpaid", "openAmount": open,
	}
}

func TestShownArtifact_KnowsAReceivablesTablesRowsAreInvoices(t *testing.T) {
	t.Parallel()

	table := ArtifactFromObservation(assistanttest.Observation("receivable_list_open_items",
		map[string]any{
			"columns": []any{
				"invoiceId", "invoiceNumber", "customerId", "customer", "settlementStatus",
				"openAmount",
			},
			"count": 2,
			"items": []any{
				openItem("inv_01JPEAK00000000000000005", "SEED-INV-0005", "2081.25"),
				openItem("inv_01JPEAK00000000000000006", "SEED-INV-0006", "2531.25"),
			},
		}))
	require.NotNil(t, table)
	require.Equal(t, assistantartifact.KindTableView, table.Kind)
	assert.Equal(t, "invoice", table.Payload["recordEntity"])
	rows, _ := table.Payload["rows"].([]any)
	require.Len(t, rows, 2)
	assert.Equal(t, "inv_01JPEAK00000000000000005", rows[0].(map[string]any)["id"],
		"each row keeps its invoice's id, which its link opens")

	shown := ShownArtifact(table)
	assert.Equal(t, "invoice", shown.RowKind)
	assert.Equal(t, []string{"SEED-INV-0005", "SEED-INV-0006"}, shown.Labels,
		"the first column is the invoice's number")
	assert.Equal(t, 2, shown.Rows)
}

func TestShownArtifact_GivesNoRowKindToATableOfNoRecords(t *testing.T) {
	t.Parallel()

	table := ArtifactFromObservation(assistanttest.Observation("receivable_list_open_items",
		map[string]any{
			"columns": []any{"customer", "openAmount"},
			"count":   1,
			"items":   []any{map[string]any{"customer": "Peak", "openAmount": "1.00"}},
		}))
	require.NotNil(t, table)
	assert.Empty(t, ShownArtifact(table).RowKind)
}
