package invoiceassembly

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/stretchr/testify/assert"
)

func TestResolvePaymentTerm(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		customer *customer.Customer
		control  *billingcontrol.BillingControl
		expected string
	}{
		{
			name: "customer override wins",
			customer: &customer.Customer{
				BillingProfile: &customer.CustomerBillingProfile{
					PaymentTerm: customer.PaymentTermNet15,
				},
			},
			control: &billingcontrol.BillingControl{
				DefaultPaymentTerm: billingcontrol.PaymentTermNet60,
			},
			expected: "Net15",
		},
		{
			name: "tenant fallback used when customer term missing",
			customer: &customer.Customer{
				BillingProfile: &customer.CustomerBillingProfile{},
			},
			control: &billingcontrol.BillingControl{
				DefaultPaymentTerm: billingcontrol.PaymentTermNet45,
			},
			expected: "Net45",
		},
		{
			name:     "empty when neither source has a term",
			customer: &customer.Customer{},
			control:  nil,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, string(ResolvePaymentTerm(tc.customer, tc.control)))
		})
	}
}
