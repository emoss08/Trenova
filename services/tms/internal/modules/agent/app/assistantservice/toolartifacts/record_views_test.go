package toolartifacts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A billing item is named as a biller knows it: its number and who it bills.
func TestEntityCardTitle_NamesABillingItemByNumberAndBillTo(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "INV2610000001 · Peak Distributing", entityCardTitle(
		"billing_queue_item",
		map[string]any{
			"number":    "INV2610000001",
			"billTo":    "Peak Distributing",
			"proNumber": "SEED-SHP-009",
		},
	))
	assert.Equal(t, "Billing queue item SEED-SHP-009", entityCardTitle(
		"billing_queue_item",
		map[string]any{
			"proNumber": "SEED-SHP-009",
		},
	), "without both it falls back to the record label")
}

// An invoice carries its load's pro number too; it is named by its own number.
func TestEntityCardTitle_NamesAnInvoiceByItsOwnNumber(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "INV2610000011 · Peak Distributing", entityCardTitle(
		"invoice",
		map[string]any{
			"number":     "INV2610000011",
			"billToName": "Peak Distributing",
			"proNumber":  "SEED-PAY-009",
		},
	))
}
