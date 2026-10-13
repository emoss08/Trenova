package invoice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/stretchr/testify/assert"
)

func TestBilledIn(t *testing.T) {
	t.Parallel()

	usd := &invoice.Invoice{CurrencyCode: "USD"}
	assert.True(t, usd.BilledIn("usd"))
	assert.True(t, usd.BilledIn(""), "a blank currency is saved as the default")
	assert.False(t, usd.BilledIn("CAD"))
	assert.True(t, (&invoice.Invoice{}).BilledIn("USD"))
	assert.False(t, (&invoice.Invoice{CurrencyCode: "MXN"}).BilledIn(""))
}
