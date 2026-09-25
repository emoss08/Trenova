package watchtowersources

import (
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	accountingSyncSuffix      = ":sync:"
	accountingDeadLetterKey   = "DeadLettered"
	accountingPausedSuffix    = ":paused"
	accountingSafetyNetSuffix = ":safety-net"
	AccountingSyncLedgerPath  = "/accounting/sync"
	accountingPausedAfter     = int64(24 * 60 * 60)
	maxSummaryLength          = 500
)

func AccountingSyncAttentionKeys() []string {
	keys := make([]string, 0, len(accountingsync.AllSyncErrorCategories())+1)
	for _, category := range accountingsync.AllSyncErrorCategories() {
		if !category.Retries() && !category.WaitsOnConnection() {
			keys = append(keys, string(category))
		}
	}
	return append(keys, accountingDeadLetterKey)
}

func AccountingSyncAttentionKey(group *repositories.AccountingSyncAttentionGroup) string {
	if group.Status == accountingsync.SyncStatusDeadLettered {
		return accountingDeadLetterKey
	}
	return string(group.ErrorCategory)
}

func AccountingSyncAttentionSourceID(conn *accountingsync.AccountingConnection, key string) string {
	return conn.ID.String() + accountingSyncSuffix + key
}

func AccountingSyncPausedSourceID(conn *accountingsync.AccountingConnection) string {
	return conn.ID.String() + accountingPausedSuffix
}

func AccountingSafetyNetSourceID(conn *accountingsync.AccountingConnection) string {
	return conn.ID.String() + accountingSafetyNetSuffix
}

func tenantOf(conn *accountingsync.AccountingConnection) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
}

func documentsLabel(count int) string {
	if count == 1 {
		return "1 document"
	}
	return strconv.Itoa(count) + " documents"
}

func DescribeAccountingSyncAttention(
	conn *accountingsync.AccountingConnection,
	key string,
	groups []repositories.AccountingSyncAttentionGroup,
) (services.WatchtowerItemInput, bool) {
	if len(groups) == 0 {
		return services.WatchtowerItemInput{}, false
	}

	total := 0
	top := &groups[0]
	oldest := groups[0].OldestQueuedAt
	for idx := range groups {
		total += groups[idx].Count
		if groups[idx].Count > top.Count {
			top = &groups[idx]
		}
		oldest = min(oldest, groups[idx].OldestQueuedAt)
	}
	if total == 0 {
		return services.WatchtowerItemInput{}, false
	}

	provider := accountingsync.ProviderName(conn.IntegrationType)
	severity := watchtower.SeverityWarning
	eventKind := agent.EventAccountingSyncBlocked
	title := documentsLabel(total) + " blocked from " + provider
	if key == accountingDeadLetterKey {
		severity = watchtower.SeverityCritical
		eventKind = agent.EventAccountingSyncFailed
		title = documentsLabel(total) + " stopped retrying to reach " + provider
	}

	summary := top.Resolution
	if summary == "" {
		summary = "Open the sync ledger to see why and fix it."
	}
	if len(groups) > 1 {
		summary += " " + strconv.Itoa(len(groups)-1) + " other causes are in the ledger."
	}

	return services.WatchtowerItemInput{
		TenantInfo:  tenantOf(conn),
		SourceKind:  watchtower.SourceAccountingSync,
		SourceID:    AccountingSyncAttentionSourceID(conn, key),
		Severity:    severity,
		Title:       stringutils.TruncateRunes(title, 200),
		Summary:     stringutils.OneLine(summary, maxSummaryLength),
		SubjectType: agent.SubjectAccountingSyncRecord,
		SubjectID:   top.SampleRecordID,
		EventKind:   eventKind,
		Path:        AccountingSyncLedgerPath,
		OccurredAt:  oldest,
	}, true
}

func DescribeAccountingSyncPaused(
	conn *accountingsync.AccountingConnection,
	now int64,
) (services.WatchtowerItemInput, bool) {
	if conn.PausedAt == nil || !conn.IsActive() || now-*conn.PausedAt < accountingPausedAfter {
		return services.WatchtowerItemInput{}, false
	}

	provider := accountingsync.ProviderName(conn.IntegrationType)
	summary := "Documents keep queuing but nothing is sent until sync resumes."
	if conn.PausedReason != "" {
		summary = "Paused: " + conn.PausedReason + ". " + summary
	}
	return services.WatchtowerItemInput{
		TenantInfo:  tenantOf(conn),
		SourceKind:  watchtower.SourceAccountingSync,
		SourceID:    AccountingSyncPausedSourceID(conn),
		Severity:    watchtower.SeverityWarning,
		Title:       "Sync to " + provider + " has been paused for over a day",
		Summary:     stringutils.OneLine(summary, maxSummaryLength),
		SubjectType: agent.SubjectAccountingConnection,
		SubjectID:   conn.ID,
		EventKind:   agent.EventAccountingConnectionDegraded,
		Path:        AccountingSyncLedgerPath,
		OccurredAt:  *conn.PausedAt,
	}, true
}

func DescribeAccountingSafetyNet(
	conn *accountingsync.AccountingConnection,
	found int,
	at int64,
) (services.WatchtowerItemInput, bool) {
	if found <= 0 {
		return services.WatchtowerItemInput{}, false
	}
	return services.WatchtowerItemInput{
		TenantInfo: tenantOf(conn),
		SourceKind: watchtower.SourceAccountingSync,
		SourceID:   AccountingSafetyNetSourceID(conn),
		Severity:   watchtower.SeverityInfo,
		Title:      documentsLabel(found) + " were posted without being queued for the books",
		Summary: "The hourly check found and queued them, so they will still arrive. " +
			"A posting path skipped the queue; report it so it can be fixed.",
		SubjectType: agent.SubjectAccountingConnection,
		SubjectID:   conn.ID,
		EventKind:   agent.EventAccountingConnectionDegraded,
		Path:        AccountingSyncLedgerPath,
		OccurredAt:  at,
	}, true
}

func GroupAccountingSyncAttention(
	groups []repositories.AccountingSyncAttentionGroup,
) map[string][]repositories.AccountingSyncAttentionGroup {
	byKey := make(map[string][]repositories.AccountingSyncAttentionGroup, len(groups))
	for idx := range groups {
		key := AccountingSyncAttentionKey(&groups[idx])
		byKey[key] = append(byKey[key], groups[idx])
	}
	return byKey
}
