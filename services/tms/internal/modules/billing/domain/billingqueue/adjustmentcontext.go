package billingqueue

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type AdjustmentContext struct {
	Rebill *AdjustmentRebill `json:"rebill,omitempty"`
	Memo   *AdjustmentMemo   `json:"memo,omitempty"`
}

type AdjustmentRebill struct {
	ReplacementLines []*invoice.InvoiceLine `json:"replacementLines"`
	SubtotalAmount   decimal.Decimal        `json:"subtotalAmount"`
	OtherAmount      decimal.Decimal        `json:"otherAmount"`
	TotalAmount      decimal.Decimal        `json:"totalAmount"`
	AccountingDate   int64                  `json:"accountingDate"`
}

type AdjustmentMemo struct {
	Kind       invoice.MemoKind `json:"kind"`
	RaisedByID pulid.ID         `json:"raisedById,omitempty"`
}
