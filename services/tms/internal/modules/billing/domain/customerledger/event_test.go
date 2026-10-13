package customerledger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Invoice", EventLabel("InvoicePosted"))
	assert.Equal(t, "Payment", EventLabel("CustomerPaymentPosted"))
	assert.Equal(t, "Payment applied", EventLabel(EventPaymentApplied))
	assert.Equal(t, "Short payment written off", EventLabel("CustomerShortPayRecognized"))
	assert.Equal(t, "Write off approved", EventLabel("WriteOffApproved"))
	assert.Empty(t, EventLabel("  "))
}
