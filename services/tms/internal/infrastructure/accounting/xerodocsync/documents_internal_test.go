package xerodocsync

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPurchaseCreditFlagReadsNeutralAndLegacyRefs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		refs map[string]string
		want bool
		err  bool
	}{
		{map[string]string{}, false, false},
		{map[string]string{accountingsync.ExternalRefCreditDocument: "true"}, true, false},
		{map[string]string{accountingsync.ExternalRefCreditDocument: "false"}, false, false},
		{map[string]string{accountingsync.ExternalRefDocumentType: "ACCPAYCREDIT"}, true, false},
		{map[string]string{accountingsync.ExternalRefCreditDocument: "maybe"}, false, true},
	}
	for _, tc := range cases {
		got, err := PurchaseIsCredit(tc.refs)
		if tc.err {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
}

func TestPaymentReference(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ACH 1 req", PaymentReference(" ACH\t\"1\\ ", "req"))
	assert.Equal(t, "req", PaymentReference("", "req"))
	assert.Equal(t, "ACH", PaymentReference("ACH", ""))
	requestID := "trn-0123456789abcdef0123456789abcdef01234567"
	long := PaymentReference(strings.Repeat("x", 400), requestID)
	assert.Len(t, []rune(long), 255)
	assert.True(t, strings.HasSuffix(long, " "+requestID))
}
