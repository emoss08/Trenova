package agenttoolstest

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func ReceivableInvoice(number string, totalMinor, appliedMinor int64) *invoice.Invoice {
	return &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		Number:             number,
		Status:             invoice.StatusPosted,
		BillType:           billingqueue.BillTypeInvoice,
		CurrencyCode:       "USD",
		TotalAmount:        decimal.NewFromInt(totalMinor / 100),
		TotalAmountMinor:   totalMinor,
		AppliedAmount:      decimal.NewFromInt(appliedMinor / 100),
		AppliedAmountMinor: appliedMinor,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		Version:            4,
	}
}
