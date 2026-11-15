package workerperms

import "github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"

// workerFieldFeatures are the values of other features a worker holds, by API
// name, most of them on the worker's profile: what Safety & Compliance works
// out of the driver's files and tests, the safety rating and score, the
// training standing, and the leave the driver is on. The licence and medical
// dates a person enters, and whether the driver is qualified and may be
// assigned, are base.
func workerFieldFeatures() map[string]platformcatalog.FeatureKey {
	return map[string]platformcatalog.FeatureKey{
		"complianceStatus":          platformcatalog.FeatureWorkforceCompliance,
		"lastComplianceCheck":       platformcatalog.FeatureWorkforceCompliance,
		"lastMvrCheck":              platformcatalog.FeatureWorkforceCompliance,
		"lastDrugTest":              platformcatalog.FeatureWorkforceCompliance,
		"drugAlcoholStatus":         platformcatalog.FeatureWorkforceCompliance,
		"returnToDutyStatus":        platformcatalog.FeatureWorkforceCompliance,
		"lastClearinghouseQueryAt":  platformcatalog.FeatureWorkforceCompliance,
		"nextClearinghouseQueryDue": platformcatalog.FeatureWorkforceCompliance,
		"nextCredentialExpiry":      platformcatalog.FeatureWorkforceCompliance,
		"safetyRating":              platformcatalog.FeatureWorkforceSafety,
		"safetyScore":               platformcatalog.FeatureWorkforceSafety,
		"trainingHealth":            platformcatalog.FeatureWorkforceTalent,
		"nextTrainingDue":           platformcatalog.FeatureWorkforceTalent,
		"leaveType":                 platformcatalog.FeatureWorkforceTimeOff,
	}
}
