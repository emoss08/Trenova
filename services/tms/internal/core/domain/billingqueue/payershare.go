package billingqueue

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const freightLineKey = "freight"

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
	Kind               shipment.ChargeAllocationKind   `json:"kind"`
	AdditionalChargeID pulid.ID                        `json:"additionalChargeId"`
	Description        string                          `json:"description"`
	ChargeTotal        decimal.Decimal                 `json:"chargeTotal"`
	Amount             decimal.Decimal                 `json:"amount"`
	Percent            decimal.NullDecimal             `json:"percent"`
	Method             shipment.ChargeAllocationMethod `json:"method"`
	Partial            bool                            `json:"partial"`
	Payers             []PayerShareParty               `json:"payers"`
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

// BuildPayerShare divides the shipment's charges and returns the payer's part.
// The shipment must carry its additional charges and charge allocations. Names
// come from the map first and then from the shipment's customer relations.
func BuildPayerShare(
	shp *shipment.Shipment,
	payerID pulid.ID,
	names map[pulid.ID]PayerRef,
) *PayerShare {
	result := &PayerShare{
		PayerID:         payerID,
		Payers:          []PayerRef{},
		Lines:           []*PayerShareLine{},
		OtherPayerLines: []*PayerShareLine{},
	}
	if shp == nil {
		return result
	}
	result.ShipmentTotal = shp.TotalChargeAmount.Decimal

	resolution, err := shipment.ResolveShares(shp, shp.ChargeAllocations)
	if err != nil {
		result.ResolutionError = err.Error()
		return result
	}

	lookup := payerNameLookup(shp, names)
	lines := make(map[string]*PayerShareLine, len(shp.AdditionalCharges)+1)
	for _, share := range resolution.Shares {
		if share == nil || len(share.Charges) == 0 {
			continue
		}
		ref := lookup(share.PayerID)
		result.Payers = append(result.Payers, ref)
		for i := range share.Charges {
			charge := &share.Charges[i]
			key := lineKey(charge)
			line, ok := lines[key]
			if !ok {
				line = newPayerShareLine(charge)
				lines[key] = line
			}
			line.Payers = append(line.Payers, PayerShareParty{
				PayerID:   ref.ID,
				PayerName: ref.Name,
				PayerCode: ref.Code,
				Amount:    charge.Amount,
				Percent:   charge.Percent,
			})
			if share.PayerID != payerID {
				continue
			}
			line.Amount = charge.Amount
			line.Percent = charge.Percent
			line.Partial = charge.Partial
		}
		if share.PayerID == payerID {
			result.FreightAmount = share.FreightAmount
			result.AccessorialAmount = share.AccessorialAmount
			result.TotalAmount = share.TotalAmount
		}
	}
	result.IsSplit = resolution.IsSplit && len(result.Payers) > 1

	for _, key := range orderedLineKeys(shp) {
		line, ok := lines[key]
		if !ok {
			continue
		}
		if paysAny(line, payerID) {
			result.Lines = append(result.Lines, line)
		} else {
			result.OtherPayerLines = append(result.OtherPayerLines, line)
		}
	}

	return result
}

func newPayerShareLine(charge *shipment.AllocatedCharge) *PayerShareLine {
	line := &PayerShareLine{
		Kind:        charge.Kind,
		ChargeTotal: charge.ChargeTotal,
		Method:      charge.Method,
		Description: "Freight",
		Payers:      make([]PayerShareParty, 0, 2),
	}
	if charge.Charge != nil {
		line.AdditionalChargeID = charge.Charge.ID
		line.Description = accessorialDescription(charge.Charge)
	}

	return line
}

func accessorialDescription(charge *shipment.AdditionalCharge) string {
	if charge.AccessorialCharge == nil {
		return "Accessorial charge"
	}
	if charge.AccessorialCharge.Description != "" {
		return charge.AccessorialCharge.Description
	}
	if charge.AccessorialCharge.Code != "" {
		return charge.AccessorialCharge.Code
	}

	return "Accessorial charge"
}

func lineKey(charge *shipment.AllocatedCharge) string {
	if charge.Charge != nil {
		return charge.Charge.ID.String()
	}

	return freightLineKey
}

func orderedLineKeys(shp *shipment.Shipment) []string {
	keys := make([]string, 0, len(shp.AdditionalCharges)+1)
	keys = append(keys, freightLineKey)
	for _, charge := range shp.AdditionalCharges {
		if charge != nil {
			keys = append(keys, charge.ID.String())
		}
	}

	return keys
}

func paysAny(line *PayerShareLine, payerID pulid.ID) bool {
	for _, party := range line.Payers {
		if party.PayerID == payerID {
			return true
		}
	}

	return false
}

func payerNameLookup(
	shp *shipment.Shipment,
	names map[pulid.ID]PayerRef,
) func(pulid.ID) PayerRef {
	return func(id pulid.ID) PayerRef {
		if ref, ok := names[id]; ok && ref.Name != "" {
			ref.ID = id
			return ref
		}
		switch {
		case shp.BillToCustomer != nil && shp.BillToCustomer.ID == id:
			return PayerRef{ID: id, Name: shp.BillToCustomer.Name, Code: shp.BillToCustomer.Code}
		case shp.Customer != nil && shp.Customer.ID == id:
			return PayerRef{ID: id, Name: shp.Customer.Name, Code: shp.Customer.Code}
		}
		for _, allocation := range shp.ChargeAllocations {
			if allocation != nil && allocation.BillToCustomer != nil && allocation.BillToCustomerID == id {
				return PayerRef{
					ID:   id,
					Name: allocation.BillToCustomer.Name,
					Code: allocation.BillToCustomer.Code,
				}
			}
		}

		return PayerRef{ID: id}
	}
}
