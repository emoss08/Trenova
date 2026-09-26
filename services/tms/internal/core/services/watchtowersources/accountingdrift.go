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
	accountingDriftSuffix = ":drift:"
	AccountingDriftPath   = "/accounting/sync/drift"
	AccountingDriftAfter  = 24 * time.Hour
)

func AccountingDriftAttentionSourceID(
	conn *accountingsync.AccountingConnection,
	kind accountingsync.DriftKind,
) string {
	return conn.ID.String() + accountingDriftSuffix + string(kind)
}

func customersLabel(count int) string {
	if count == 1 {
		return "1 customer's balance"
	}
	return strconv.Itoa(count) + " customers' balances"
}

func driftAttentionText(
	provider string,
	kind accountingsync.DriftKind,
	count int,
) (severity watchtower.Severity, title, summary string) {
	documents := documentsLabel(count)
	switch kind {
	case accountingsync.DriftAmountMismatch:
		return watchtower.SeverityWarning,
			documents + " have a different total in " + provider,
			"Someone changed them there after Trenova sent them. Send Trenova's total " +
				"again, match Trenova to the books, or dismiss the difference."
	case accountingsync.DriftDeletedInProvider:
		return watchtower.SeverityCritical,
			documents + " were deleted in " + provider,
			"The books no longer hold documents Trenova posted. Create them there again, " +
				"or void them in Trenova."
	case accountingsync.DriftVoidedInProvider:
		return watchtower.SeverityCritical,
			documents + " were voided in " + provider,
			"The books voided documents that are live in Trenova. Create them there " +
				"again, or void them in Trenova."
	case accountingsync.DriftStatusMismatch:
		return watchtower.SeverityWarning,
			documents + " voided in Trenova are still live in " + provider,
			"Send the void to the books, or dismiss the difference if it was kept on " +
				"purpose."
	case accountingsync.DriftCustomerBalanceMismatch:
		return watchtower.SeverityWarning,
			customersLabel(count) + " differ between Trenova and " + provider,
			"The documents both sides hold add up to different open balances. Open each " +
				"one to see which documents differ."
	default:
		return watchtower.SeverityWarning,
			documents + " differ between Trenova and " + provider,
			"Open each one to see both values."
	}
}

func DescribeAccountingDriftAttention(
	conn *accountingsync.AccountingConnection,
	group *repositories.AccountingDriftAttentionGroup,
) (services.WatchtowerItemInput, bool) {
	if group == nil || group.Count <= 0 {
		return services.WatchtowerItemInput{}, false
	}
	provider := accountingsync.ProviderName(conn.IntegrationType)
	severity, title, summary := driftAttentionText(provider, group.Kind, group.Count)
	return services.WatchtowerItemInput{
		TenantInfo:  tenantOf(conn),
		SourceKind:  watchtower.SourceAccountingSync,
		SourceID:    AccountingDriftAttentionSourceID(conn, group.Kind),
		Severity:    severity,
		Title:       stringutils.TruncateRunes(title, 200),
		Summary:     stringutils.OneLine(summary, maxSummaryLength),
		SubjectType: agent.SubjectAccountingDrift,
		SubjectID:   group.SampleID,
		EventKind:   agent.EventAccountingDriftDetected,
		Path:        AccountingDriftPath,
		OccurredAt:  group.OldestDetectedAt,
	}, true
}
