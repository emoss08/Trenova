package auditreviewservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/shared/i18n"
)

//nolint:cyclop // one sentence per alert kind, each a literal the catalog extracts
func Title(locale i18n.Locale, review *audit.Review) string {
	actor := strings.TrimSpace(review.ActorName)
	if actor == "" {
		actor = i18n.Translate(locale, "Someone")
	}
	ref := strings.TrimSpace(review.EntityRef)

	switch review.Kind {
	case audit.AlertRoleChanged:
		if ref != "" {
			return i18n.Translate(locale, "{0} changed a role: {1}", actor, ref)
		}
		return i18n.Translate(locale, "{0} changed a role", actor)
	case audit.AlertAPIKeyChanged:
		if ref != "" {
			return i18n.Translate(locale, "{0} changed the API key {1}", actor, ref)
		}
		return i18n.Translate(locale, "{0} changed an API key", actor)
	case audit.AlertMFAReset:
		if ref != "" {
			return i18n.Translate(locale, "{0} turned off or reset two-factor for {1}", actor, ref)
		}
		return i18n.Translate(locale, "{0} turned off or reset two-factor", actor)
	case audit.AlertIdentityConfigChanged:
		if ref != "" {
			return i18n.Translate(locale, "{0} changed sign-in settings: {1}", actor, ref)
		}
		return i18n.Translate(locale, "{0} changed sign-in settings", actor)
	case audit.AlertCrossTenantRefused:
		return i18n.Translate(locale, "{0} reached for another organization's records", actor)
	case audit.AlertMalwareRefused:
		return i18n.Translate(locale, "{0} sent a file the malware scanner refused", actor)
	case audit.AlertLegalHoldChanged:
		if review.Action == audit.ActionLegalHoldReleased {
			return i18n.Translate(locale, "{0} released a legal hold", actor)
		}
		return i18n.Translate(locale, "{0} placed a legal hold", actor)
	case audit.AlertBulkDelete:
		return i18n.Translate(
			locale,
			"{0} deleted {1, plural, one {# record} other {# records}} within {2, plural, one {# minute} other {# minutes}}",
			actor,
			review.Count,
			review.WindowMinutes,
		)
	case audit.AlertBulkExport:
		return i18n.Translate(
			locale,
			"{0} exported or downloaded {1, plural, one {# time} other {# times}} within {2, plural, one {# minute} other {# minutes}}",
			actor,
			review.Count,
			review.WindowMinutes,
		)
	default:
		return i18n.Translate(locale, "{0} set off a security alert", actor)
	}
}
