package tenantkeys

const (
	userWithoutUnit = "the table has organization_id and no business_unit_id, and users has no " +
		"organization_id, so no column pair can hold the person to the row's tenant"
	unitMayBeEmpty = "the table's business_unit_id may be empty (a row written before the person " +
		"or tenant is known), and a composite key is not checked for a row whose business unit is empty"
	heldByDetection = "; held instead by the tenant removal's cross-tenant count, the archive's " +
		"tenant refusal and the nightly trenova_tenancy_archive_refusals_total metric"
)

var exceptions = map[string]string{
	"mfa_authenticators_user_id_fkey": userWithoutUnit + "; written only for the signed-in " +
		"person (the policy admits trenova_rls.user_id())" + heldByDetection,
	"fk_mfa_recovery_codes_user": userWithoutUnit + "; written only for the signed-in person " +
		"(the policy admits trenova_rls.user_id())" + heldByDetection,
	"fk_user_role_assignments_user": userWithoutUnit + "; the role service assigns only " +
		"members of the organization" + heldByDetection,
	"fk_user_role_assignments_assigned_by": userWithoutUnit + "; the assigner is the " +
		"session's own person" + heldByDetection,
	"auth_events_user_id_fkey":              unitMayBeEmpty + heldByDetection,
	"auth_events_identity_provider_id_fkey": unitMayBeEmpty + heldByDetection,
	"fk_notifications_target_user_id":       unitMayBeEmpty + heldByDetection,
	"risk_decisions_user_id_fkey":           unitMayBeEmpty + heldByDetection,
}

func Exceptions() map[string]string {
	out := make(map[string]string, len(exceptions))
	for name, reason := range exceptions {
		out[name] = reason
	}
	return out
}
