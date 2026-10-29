package billingqueue

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// PayerRef names a payer without the rest of the customer record.
type PayerRef struct {
	ID   pulid.ID `json:"id"`
	Name string   `json:"name"`
	Code string   `json:"code"`
}

// PayerShareParty is one payer's part of one charge.
type PayerShareParty struct {
	PayerID   pulid.ID            `json:"payerId"`
	PayerName string              `json:"payerName"`
	PayerCode string              `json:"payerCode"`
	Amount    decimal.Decimal     `json:"amount"`
	Percent   decimal.NullDecimal `json:"percent"`
}

// PayerShareLine is one charge as a single payer's bill sees it: the whole
// charge, this payer's part of it, and everyone who pays some of it.
type PayerShareLine struct {
	Kind               shipmentvocab.ChargeAllocationKind   `json:"kind"`
	AdditionalChargeID pulid.ID                             `json:"additionalChargeId"`
	Description        string                               `json:"description"`
	ChargeTotal        decimal.Decimal                      `json:"chargeTotal"`
	Amount             decimal.Decimal                      `json:"amount"`
	Percent            decimal.NullDecimal                  `json:"percent"`
	Method             shipmentvocab.ChargeAllocationMethod `json:"method"`
	Partial            bool                                 `json:"partial"`
	Payers             []PayerShareParty                    `json:"payers"`
}

// PayerShare is what one billing queue item bills: the charges its payer owes,
// computed by the same resolver that builds the invoice lines, so the review
// screen and the invoice can never disagree. Charges the payer owes none of are
// kept separately so a reviewer can still see the whole shipment.
type PayerShare struct {
	PayerID           pulid.ID          `json:"payerId"`
	IsSplit           bool              `json:"isSplit"`
	Payers            []PayerRef        `json:"payers"`
	Lines             []*PayerShareLine `json:"lines"`
	OtherPayerLines   []*PayerShareLine `json:"otherPayerLines"`
	FreightAmount     decimal.Decimal   `json:"freightAmount"`
	AccessorialAmount decimal.Decimal   `json:"accessorialAmount"`
	TotalAmount       decimal.Decimal   `json:"totalAmount"`
	ShipmentTotal     decimal.Decimal   `json:"shipmentTotal"`
	// ResolutionError explains why the shipment's split cannot be divided right
	// now, such as an amount split that no longer adds up after a charge changed.
	ResolutionError string `json:"resolutionError"`
}
