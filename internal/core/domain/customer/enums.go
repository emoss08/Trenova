// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package customer

type BillingCycleType string

const (
	// * BillingCycleTypeImmediate indicates that the customer wants to be billed immediately
	// * after the shipment is delivered
	BillingCycleTypeImmediate = BillingCycleType("Immediate")

	// * BillingCycleTypeDaily indicates that the customer wants to be billed daily
	BillingCycleTypeDaily = BillingCycleType("Daily")

	// * BillingCycleTypeWeekly indicates that the customer wants to be billed weekly
	BillingCycleTypeWeekly = BillingCycleType("Weekly")

	// * BillingCycleTypeMonthly indicates that the customer wants to be billed monthly
	BillingCycleTypeMonthly = BillingCycleType("Monthly")

	// * BillingCycleTypeQuarterly indicates that the customer wants to be billed quarterly
	BillingCycleTypeQuarterly = BillingCycleType("Quarterly")
)
