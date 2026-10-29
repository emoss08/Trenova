package billingpayershare

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// BuildPayerShare divides the shipment's charges and returns the payer's part.
// The shipment must carry its additional charges and charge allocations. Names
// come from the map first and then from the shipment's customer relations.
func BuildPayerShare(
	ctx context.Context,
	shp *shipment.Shipment,
	payerID pulid.ID,
	names map[pulid.ID]billingqueue.PayerRef,
) (*billingqueue.PayerShare, error) {
	result := &billingqueue.PayerShare{
		PayerID:         payerID,
		Payers:          []billingqueue.PayerRef{},
		Lines:           []*billingqueue.PayerShareLine{},
		OtherPayerLines: []*billingqueue.PayerShareLine{},
	}
	if shp == nil {
		return result, nil
	}
	result.ShipmentTotal = shp.TotalChargeAmount.Decimal

	resolution, err := chargeallocation.ResolveShares(ctx, shp, shp.ChargeAllocations)
	var refused *errortypes.Error
	switch {
	case errors.As(err, &refused):
		result.ResolutionError = errortypes.LocalizedText(ctx, refused)
		return result, nil
	case err != nil:
		return nil, err
	}

	lookup := payerNameLookup(shp, names)
	lines := make(map[string]*billingqueue.PayerShareLine, len(shp.AdditionalCharges)+1)
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
			line.Payers = append(line.Payers, billingqueue.PayerShareParty{
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

	return result, nil
}

func newPayerShareLine(charge *chargeallocation.AllocatedCharge) *billingqueue.PayerShareLine {
	line := &billingqueue.PayerShareLine{
		Kind:        charge.Kind,
		ChargeTotal: charge.ChargeTotal,
		Method:      charge.Method,
		Description: "Freight",
		Payers:      make([]billingqueue.PayerShareParty, 0, 2),
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

func lineKey(charge *chargeallocation.AllocatedCharge) string {
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

func paysAny(line *billingqueue.PayerShareLine, payerID pulid.ID) bool {
	for _, party := range line.Payers {
		if party.PayerID == payerID {
			return true
		}
	}

	return false
}

func payerNameLookup(
	shp *shipment.Shipment,
	names map[pulid.ID]billingqueue.PayerRef,
) func(pulid.ID) billingqueue.PayerRef {
	return func(id pulid.ID) billingqueue.PayerRef {
		if ref, ok := names[id]; ok && ref.Name != "" {
			ref.ID = id
			return ref
		}
		switch {
		case shp.BillToCustomer != nil && shp.BillToCustomer.ID == id:
			return billingqueue.PayerRef{
				ID:   id,
				Name: shp.BillToCustomer.Name,
				Code: shp.BillToCustomer.Code,
			}
		case shp.Customer != nil && shp.Customer.ID == id:
			return billingqueue.PayerRef{ID: id, Name: shp.Customer.Name, Code: shp.Customer.Code}
		}
		for _, allocation := range shp.ChargeAllocations {
			if allocation != nil && allocation.BillToCustomer != nil &&
				allocation.BillToCustomerID == id {
				return billingqueue.PayerRef{
					ID:   id,
					Name: allocation.BillToCustomer.Name,
					Code: allocation.BillToCustomer.Code,
				}
			}
		}

		return billingqueue.PayerRef{ID: id}
	}
}

const freightLineKey = "freight"
