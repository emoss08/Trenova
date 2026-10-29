package watchtoweraccounting

import (
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountinginboundports"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtoweritem"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	accountingInboundSuffix = ":inbound:"
	AccountingInboundPath   = "/accounting/sync/inbound"
	AccountingInboundAfter  = 24 * time.Hour
)

func AccountingInboundAttentionSourceID(
	conn *accountingconnection.AccountingConnection,
	reason accountinginbound.InboundChangeReason,
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
	reason accountinginbound.InboundChangeReason,
	count int,
) (severity watchtower.Severity, title, summary string) {
	payments := paymentsLabel(count)
	switch reason { //nolint:exhaustive // every other reason shares the general wording
	case accountinginbound.InboundReasonPolicyPropose:
		return watchtower.SeverityInfo,
			payments + " recorded in " + provider + " wait to be applied in Trenova",
			"Apply each one to post it against the documents it pays, or ignore it if it " +
				"was entered in Trenova too."
	case accountinginbound.InboundReasonPeriodNotOpen:
		return watchtower.SeverityWarning,
			payments + " recorded in " + provider + " fall in a period that is not open",
			"Trenova does not post into a locked or closed period. Reopen the period, " +
				"then apply them."
	case accountinginbound.InboundReasonApplyFailed:
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
	conn *accountingconnection.AccountingConnection,
	group *accountinginboundports.AccountingInboundAttentionGroup,
) (watchtowerports.WatchtowerItemInput, bool) {
	if group == nil || group.Count <= 0 {
		return watchtowerports.WatchtowerItemInput{}, false
	}
	provider := accountingprovider.ProviderName(conn.IntegrationType)
	severity, title, summary := inboundAttentionText(provider, group.Reason, group.Count)
	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenantOf(conn),
		SourceKind:  watchtower.SourceAccountingSync,
		SourceID:    AccountingInboundAttentionSourceID(conn, group.Reason),
		Severity:    severity,
		Title:       stringutils.TruncateRunes(title, 200),
		Summary:     stringutils.OneLine(summary, watchtoweritem.MaxSummaryLength),
		SubjectType: agent.SubjectAccountingInbound,
		SubjectID:   group.SampleID,
		EventKind:   agenttrigger.EventAccountingPaymentProposed,
		Path:        AccountingInboundPath,
		OccurredAt:  group.OldestDetectedAt,
	}, true
}

func AccountingInboundAttentionReasons() []accountinginbound.InboundChangeReason {
	out := make(
		[]accountinginbound.InboundChangeReason,
		0,
		len(accountinginbound.AllInboundChangeReasons()),
	)
	for _, reason := range accountinginbound.AllInboundChangeReasons() {
		if reason != accountinginbound.InboundReasonNotTrenovaDocument &&
			reason != accountinginbound.InboundReasonSentFromTrenova &&
			reason != accountinginbound.InboundReasonVoided {
			out = append(out, reason)
		}
	}
	return out
}
