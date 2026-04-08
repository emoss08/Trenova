package invoice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDueDateFromPaymentTerm(t *testing.T) {
	t.Parallel()

	invoiceDate := int64(1_700_000_000)

	testCases := []struct {
		name     string
		term     PaymentTerm
		expected *int64
	}{
		{
			name:     "due on receipt",
			term:     PaymentTermDueOnReceipt,
			expected: int64Ptr(invoiceDate),
		},
		{
			name:     "net 30",
			term:     PaymentTermNet30,
			expected: int64Ptr(invoiceDate + 30*daySeconds),
		},
		{
			name:     "invalid payment term returns nil",
			term:     PaymentTerm("Invalid"),
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, DueDateFromPaymentTerm(invoiceDate, tc.term))
		})
	}
}

func TestDueDateFromPaymentTermRequiresInvoiceDate(t *testing.T) {
	t.Parallel()

	assert.Nil(t, DueDateFromPaymentTerm(0, PaymentTermNet30))
}

func int64Ptr(v int64) *int64 {
	return &v
}
