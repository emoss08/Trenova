package usageservice

const (
	ReasonQuotaExceeded  = "quota_exceeded"
	ReasonWithinPlan     = "within_plan_limit"
	ReasonUnlimitedPlan  = "unlimited_plan"
	ReasonNotPlanMetered = "not_plan_metered"
	ReasonDerivedUsage   = "derived_from_records"
)
