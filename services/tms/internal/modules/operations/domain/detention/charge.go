package detention

import "github.com/shopspring/decimal"

// Chargeable reports whether the occurrence is billed on its shipment: a
// billable status, a policy it was computed under, and an amount to bill.
func (o *DetentionOccurrence) Chargeable() bool {
	return o != nil &&
		o.PolicySnapshot != nil &&
		o.Status.IsBillable() &&
		o.BillableAmount.GreaterThan(decimal.Zero)
}

// ChargedAmount is what the occurrence adds to its shipment's detention
// charge: its billable amount when it is chargeable, and nothing otherwise.
func (o *DetentionOccurrence) ChargedAmount() decimal.Decimal {
	if !o.Chargeable() {
		return decimal.Zero
	}

	return o.BillableAmount
}
