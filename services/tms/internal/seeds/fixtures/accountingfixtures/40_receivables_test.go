package accountingfixtures

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func agingBucket(ageDays int64) string {
	pastDue := ageDays - 30
	switch {
	case pastDue <= 0:
		return "current"
	case pastDue <= 30:
		return "1-30"
	case pastDue <= 60:
		return "31-60"
	case pastDue <= 90:
		return "61-90"
	default:
		return "over-90"
	}
}

func TestReceivablesSeed_SpreadsEachCustomerAcrossTheAgingBuckets(t *testing.T) {
	t.Parallel()

	buckets := map[string]int{}
	customers := map[string]int{}
	numbers := map[string]struct{}{}
	for _, spec := range seedReceivables {
		buckets[agingBucket(spec.ageDays)]++
		customers[spec.customer]++
		_, duplicate := numbers[spec.number]
		require.Falsef(t, duplicate, "%s is seeded twice", spec.number)
		numbers[spec.number] = struct{}{}
	}

	for _, bucket := range []string{"current", "1-30", "31-60", "61-90", "over-90"} {
		assert.Positivef(t, buckets[bucket], "no seeded invoice is %s", bucket)
	}
	for _, code := range []string{"ACME", "PEAK", "FRSH"} {
		assert.GreaterOrEqualf(t, customers[code], 3, "%s has fewer than three invoices", code)
	}
}

func TestReceivablesSeed_SettlesOnlyInvoicesItSeeds(t *testing.T) {
	t.Parallel()

	byNumber := make(map[string]seedReceivable, len(seedReceivables))
	for _, spec := range seedReceivables {
		byNumber[spec.number] = spec
	}

	partial, full, unapplied := 0, 0, 0
	for _, receipt := range seedReceipts {
		if receipt.unapplied != "" {
			unapplied++
		}
		for _, app := range receipt.applications {
			target, ok := byNumber[app.invoice]
			require.Truef(
				t,
				ok,
				"%s applies to %s, which is not seeded",
				receipt.reference,
				app.invoice,
			)
			assert.Equal(t, target.customer, receipt.customer, receipt.reference)
			assert.Less(t, receipt.ageDays, target.ageDays, "paid before it was invoiced")
			if app.amount == "" {
				full++
				continue
			}
			partial++
			assert.True(t, decimal.RequireFromString(app.amount).
				LessThan(decimal.RequireFromString(target.freight)), app.invoice)
		}
	}
	assert.Positive(t, partial, "no invoice is partly paid")
	assert.Positive(t, full, "no invoice is paid in full")
	assert.Positive(t, unapplied, "no payment leaves cash unapplied")

	for _, memo := range seedCreditMemos {
		target, ok := byNumber[memo.against]
		require.Truef(t, ok, "%s credits %s, which is not seeded", memo.number, memo.against)
		assert.Less(t, memo.ageDays, target.ageDays, memo.number)
	}
}
