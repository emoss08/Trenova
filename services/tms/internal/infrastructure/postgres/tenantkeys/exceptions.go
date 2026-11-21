package tenantkeys

const (
	unitMayBeEmpty = "the table's business_unit_id may be empty, and a composite key is not " +
		"checked for a row whose business unit is empty"
	heldByDetection = "; held instead by the tenant removal's cross-tenant count, the archive " +
		"export's tenant refusal, the archive import's within-tenant reference check and the " +
		"nightly trenova_tenancy_archive_refusals_total metric"
)

var exceptions = map[string]string{
	"auth_events_user_id_fkey": unitMayBeEmpty + " (a sign-in attempt that matched no " +
		"organization, or a throttled one, is recorded before any tenant is known); " +
		"autheventservice names a person only together with that person's own organization " +
		"and business unit (authservice.authAttempt.forUser) or the session's" + heldByDetection,
	"auth_events_identity_provider_id_fkey": unitMayBeEmpty + "; no writer sets the column " +
		"(autheventservice never fills identity_provider_id)" + heldByDetection,
	"fk_notifications_target_user_id": unitMayBeEmpty + " (the column is nullable); every " +
		"producer writes through notificationrepository.insert, which writes a recipient only " +
		"when that person is of the notification's organization's business unit" +
		heldByDetection,
	"risk_decisions_user_id_fkey": unitMayBeEmpty + " (organization_id may be empty too); no " +
		"code writes the table, it is only read" + heldByDetection,
}

func Exceptions() map[string]string {
	out := make(map[string]string, len(exceptions))
	for name, reason := range exceptions {
		out[name] = reason
	}
	return out
}
