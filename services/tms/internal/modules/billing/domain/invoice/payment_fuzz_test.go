//go:build fuzz

package invoice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/shared/money"
)

func FuzzPaymentsKeepTheInvoiceBalanced(f *testing.F) {
	f.Add(int64(10_000), []byte{0, 1, 2, 3, 0x81, 0x82})
	f.Add(int64(0), []byte{5, 0x85})
	f.Add(int64(1), []byte{0xff, 0x7f})
	f.Fuzz(func(t *testing.T, total int64, steps []byte) {
		total %= 1_000_000_000_00
		if total < 0 {
			total = -total
		}
		inv := &Invoice{
			BillType:         billingvocab.BillTypeInvoice,
			Status:           StatusPosted,
			TotalAmount:      money.DecimalFromMinor(total),
			SettlementStatus: SettlementStatusUnpaid,
		}
		inv.SyncMinorAmounts()
		for _, step := range steps {
			amount := int64(step&0x7f) * (total/64 + 1)
			if step&0x80 == 0 {
				inv.ApplyPaymentMinor(amount)
			} else {
				inv.RemovePaymentMinor(amount)
			}

			if inv.AppliedAmountMinor < 0 {
				t.Fatalf("the invoice holds %d applied", inv.AppliedAmountMinor)
			}
			if money.MinorUnits(inv.AppliedAmount) != inv.AppliedAmountMinor {
				t.Fatalf("applied %s does not match %d minor units",
					inv.AppliedAmount, inv.AppliedAmountMinor)
			}
			want := SettlementStatusPartiallyPaid
			switch {
			case inv.AppliedAmountMinor == 0:
				want = SettlementStatusUnpaid
			case inv.AppliedAmountMinor >= inv.TotalAmountMinor:
				want = SettlementStatusPaid
			}
			if inv.SettlementStatus != want {
				t.Fatalf("applied %d of %d reads %s, not %s",
					inv.AppliedAmountMinor, inv.TotalAmountMinor, inv.SettlementStatus, want)
			}
			if open := inv.OpenBalanceMinor(); open < 0 || open > inv.TotalAmountMinor {
				t.Fatalf("an open balance of %d on a total of %d", open, inv.TotalAmountMinor)
			}
		}
	})
}
