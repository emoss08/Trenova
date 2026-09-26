package watchtowersources

import (
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	accountingInboundSuffix = ":inbound:"
	AccountingInboundPath   = "/accounting/sync/inbound"
	AccountingInboundAfter  = 24 * time.Hour
)

func AccountingInboundAttentionSourceID(
	conn *accountingsync.AccountingConnection,
	reason accountingsync.InboundChangeReason,
) string {
	return conn.ID.String() + accountingInboundSuffix + string(reason)
}

func paymentsLabel(count int) string {
	if count == 1 {
		return "1 payment"
	}
	return strconv.Itoa(count) + " payments"
}

func inboundAttentionText(
	provider string,
	reason accountingsync.InboundChangeReason,
	count int,
) (severity watchtower.Severity, title, summary string) {
	payments := paymentsLabel(count)
	switch reason { //nolint:exhaustive // every other reason shares the general wording
	case accountingsync.InboundReasonPolicyPropose:
		return watchtower.SeverityInfo,
			payments + " recorded in " + provider + " wait to be applied in Trenova",
			"Apply each one to post it against the documents it pays, or ignore it if it " +
				"was entered in Trenova too."
	case accountingsync.InboundReasonPeriodNotOpen:
		return watchtower.SeverityWarning,
			payments + " recorded in " + provider + " fall in a period that is not open",
			"Trenova does not post into a locked or closed period. Reopen the period, " +
				"then apply them."
	case accountingsync.InboundReasonApplyFailed:
		return watchtower.SeverityWarning,
			payments + " recorded in " + provider + " could not be applied",
			"Trenova refused them when it tried. Open each one to see why."
	default:
		return watchtower.SeverityWarning,
			payments + " recorded in " + provider + " do not match Trenova",
			"What they pay differs from what Trenova holds. Open each one to see the " +
				"difference, then fix it in " + provider + " or ignore the payment."
	}
}

func DescribeAccountingInboundAttention(
	conn *accountingsync.AccountingConnection,
	group *repositories.AccountingInboundAttentionGroup,
) (services.WatchtowerItemInput, bool) {
	if group == nil || group.Count <= 0 {
		return services.WatchtowerItemInput{}, false
	}
	provider := accountingsync.ProviderName(conn.IntegrationType)
	severity, title, summary := inboundAttentionText(provider, group.Reason, group.Count)
	return services.WatchtowerItemInput{
		TenantInfo:  tenantOf(conn),
		SourceKind:  watchtower.SourceAccountingSync,
		SourceID:    AccountingInboundAttentionSourceID(conn, group.Reason),
		Severity:    severity,
		Title:       stringutils.TruncateRunes(title, 200),
		Summary:     stringutils.OneLine(summary, maxSummaryLength),
		SubjectType: agent.SubjectAccountingInbound,
		SubjectID:   group.SampleID,
		EventKind:   agent.EventAccountingPaymentProposed,
		Path:        AccountingInboundPath,
		OccurredAt:  group.OldestDetectedAt,
	}, true
}

func AccountingInboundAttentionReasons() []accountingsync.InboundChangeReason {
	out := make(
		[]accountingsync.InboundChangeReason,
		0,
		len(accountingsync.AllInboundChangeReasons()),
	)
	for _, reason := range accountingsync.AllInboundChangeReasons() {
		if reason != accountingsync.InboundReasonNotTrenovaDocument &&
			reason != accountingsync.InboundReasonSentFromTrenova &&
			reason != accountingsync.InboundReasonVoided {
			out = append(out, reason)
		}
	}
	return out
}
