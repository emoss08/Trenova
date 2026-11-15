package billingperms

import "github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"

// invoiceFieldFeatures are the values of other features an invoice holds, by
// API name: its factoring, what Accounting & Cash applied to it and still
// holds open and its EDI 210 delivery. What the tariffs that rated its
// shipments said is base, since every plan rates with the standard templates.
// A line's or charge group's fields are named by their own
// name. settlementStatus is in the invoice's reference view, so it stays
// base.
func invoiceFieldFeatures() map[string]platformcatalog.FeatureKey {
	return map[string]platformcatalog.FeatureKey{
		"factoringCompanyId":      platformcatalog.FeatureFactoring,
		"factoringAgreementId":    platformcatalog.FeatureFactoring,
		"factoringOverride":       platformcatalog.FeatureFactoring,
		"factoringOverrideReason": platformcatalog.FeatureFactoring,
		"appliedAmount":           platformcatalog.FeatureAccounting,
		"appliedAmountMinor":      platformcatalog.FeatureAccounting,
		"balanceDueMinor":         platformcatalog.FeatureAccounting,
		"openBalance":             platformcatalog.FeatureAccounting,
		"creditRemaining":         platformcatalog.FeatureAccounting,
		"daysPastDue":             platformcatalog.FeatureAccounting,
		"ediSendStatus":           platformcatalog.FeatureEDIIntegration,
		"lastEdiMessageId":        platformcatalog.FeatureEDIIntegration,
		"ediSentAt":               platformcatalog.FeatureEDIIntegration,
		"lastEdiError":            platformcatalog.FeatureEDIIntegration,
	}
}

// customerFieldFeatures are the values of other features a customer holds, by
// API name: what Visibility & Tracking tells the customer, whether Carrier &
// Brokerage vets them, and on the billing profile the credit, late charge,
// statement, reminder and ledger settings of Accounting & Cash, the factor
// their invoices go to and whether they are sent by EDI.
func customerFieldFeatures() map[string]platformcatalog.FeatureKey {
	return map[string]platformcatalog.FeatureKey{
		"statusUpdatePreference":  platformcatalog.FeatureCustomerTracking,
		"statusUpdateRecipients":  platformcatalog.FeatureCustomerTracking,
		"checkCallIntervalHours":  platformcatalog.FeatureCustomerTracking,
		"etaRiskThresholdMinutes": platformcatalog.FeatureCustomerTracking,
		"trackingProfile":         platformcatalog.FeatureCustomerTracking,
		"brokerVettingEnabled":    platformcatalog.FeatureCarrierBrokerage,
		"creditLimit":             platformcatalog.FeatureAccounting,
		"creditBalance":           platformcatalog.FeatureAccounting,
		"creditStatus":            platformcatalog.FeatureAccounting,
		"enforceCreditLimit":      platformcatalog.FeatureAccounting,
		"autoCreditHold":          platformcatalog.FeatureAccounting,
		"creditHoldReason":        platformcatalog.FeatureAccounting,
		"creditHoldAutomatic":     platformcatalog.FeatureAccounting,
		"creditReviewedAt":        platformcatalog.FeatureAccounting,
		"applyLateCharges":        platformcatalog.FeatureAccounting,
		"lateChargeRate":          platformcatalog.FeatureAccounting,
		"gracePeriodDays":         platformcatalog.FeatureAccounting,
		"sendPaymentReminders":    platformcatalog.FeatureAccounting,
		"sendStatements":          platformcatalog.FeatureAccounting,
		"revenueAccountId":        platformcatalog.FeatureAccounting,
		"arAccountId":             platformcatalog.FeatureAccounting,
		"factoringCompanyId":      platformcatalog.FeatureFactoring,
		"factoringCompany":        platformcatalog.FeatureFactoring,
		"ediInvoiceEnabled":       platformcatalog.FeatureEDIIntegration,
	}
}
