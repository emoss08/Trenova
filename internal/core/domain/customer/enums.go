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
