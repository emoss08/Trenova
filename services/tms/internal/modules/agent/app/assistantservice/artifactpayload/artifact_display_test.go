package artifactpayload

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordEntityOf_FindsTheRegistrysName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "shipment", RecordEntityOf("shipments"))
	assert.Equal(t, "worker", RecordEntityOf("worker"))
	assert.Equal(t, "rate_matrix", RecordEntityOf("rate_matrices"))
	assert.Empty(t, RecordEntityOf("insights"))
	assert.Empty(t, RecordEntityOf(""))
}

func TestRowsEntityOf_ReadsRowsLedByAKindsIDAsThatKind(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "shipment", RowsEntityOf("shipments", []string{"id", "proNumber"}))
	assert.Equal(t, "invoice",
		RowsEntityOf("open_items", []string{"invoiceId", "invoiceNumber", "customerId"}))
	assert.Empty(t, RowsEntityOf("open_items", []string{"invoiceNumber", "invoiceId"}),
		"only a list that leads with a kind's id declares its rows as that kind")
	assert.Empty(t, RowsEntityOf("open_items", []string{"widgetId"}))
	assert.Empty(t, RowsEntityOf("open_items", nil))
}

func TestRecordIDOf_ReadsARowsOwnIDOrItsKindsID(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "inv_1", RecordIDOf(map[string]any{"id": "inv_1", "invoiceId": "x"}, "invoice"))
	assert.Equal(t, "inv_2", RecordIDOf(map[string]any{"invoiceId": "inv_2"}, "invoice"))
	assert.Equal(t, "je_3", RecordIDOf(map[string]any{"journalEntryId": "je_3"}, "journal_entry"))
	assert.Empty(t, RecordIDOf(map[string]any{"customerId": "cus_1"}, "invoice"))
}
