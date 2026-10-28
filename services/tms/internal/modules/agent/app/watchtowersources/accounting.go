package watchtowersources

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountinginboundports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	accountingReconnectSuffix = ":reconnect"
	reconnectDateLayout       = "January 2, 2006"
)

type AccountingSyncSource struct {
	repo    accountingconnectionports.AccountingConnectionRepository
	records accountingsyncports.AccountingSyncRecordRepository
	inbound accountinginboundports.AccountingInboundChangeRepository
}

func NewAccountingSyncSource(
	repo accountingconnectionports.AccountingConnectionRepository,
	records accountingsyncports.AccountingSyncRecordRepository,
	inbound accountinginboundports.AccountingInboundChangeRepository,
) watchtowerports.WatchtowerSource {
	return &AccountingSyncSource{repo: repo, records: records, inbound: inbound}
}

func (s *AccountingSyncSource) Kind() watchtower.SourceKind {
	return watchtower.SourceAccountingSync
}

func (s *AccountingSyncSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	connections, err := s.repo.ListByTenant(ctx, tenant)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	items := make([]watchtowerports.WatchtowerItemInput, 0, len(connections))
	for _, conn := range connections {
		if item, open := DescribeAccountingConnectionHealth(conn); open {
			items = append(items, item)
		}
		if item, open := DescribeAccountingReconnect(conn, now); open {
			items = append(items, item)
		}
		if !conn.IsSyncing() {
			continue
		}
		if item, open := DescribeAccountingSyncPaused(conn, now); open {
			items = append(items, item)
		}
		groups, listErr := s.records.ListAttention(
			ctx,
			accountingsyncports.ListAccountingSyncAttentionRequest{
				TenantInfo:   tenant,
				ConnectionID: conn.ID,
			},
		)
		if listErr != nil {
			return nil, listErr
		}
		for key, keyed := range GroupAccountingSyncAttention(groups) {
			if item, open := DescribeAccountingSyncAttention(conn, key, keyed); open {
				items = append(items, item)
			}
		}
		inbound, inboundErr := s.inbound.ListAttention(
			ctx,
			&accountinginboundports.ListAccountingInboundAttentionRequest{
				TenantInfo:     tenant,
				ConnectionID:   conn.ID,
				DetectedBefore: now - int64(AccountingInboundAfter/time.Second),
			},
		)
		if inboundErr != nil {
			return nil, inboundErr
		}
		for idx := range inbound {
			if item, open := DescribeAccountingInboundAttention(conn, &inbound[idx]); open {
				items = append(items, item)
			}
		}
	}

	return items, nil
}

func AccountingConnectionHealthSourceID(conn *accountingsync.AccountingConnection) string {
	return conn.ID.String()
}

func AccountingReconnectSourceID(conn *accountingsync.AccountingConnection) string {
	return conn.ID.String() + accountingReconnectSuffix
}

func DescribeAccountingConnectionHealth(
	conn *accountingsync.AccountingConnection,
) (watchtowerports.WatchtowerItemInput, bool) {
	provider := accountingsync.ProviderName(conn.IntegrationType)

	var severity watchtower.Severity
	var title, summary string
	switch conn.Status { //nolint:exhaustive // connected and disconnected raise nothing
	case accountingsync.ConnectionStatusDegraded:
		severity = watchtower.SeverityWarning
		title = provider + " is not answering reliably"
		summary = "The last check failed; Trenova keeps retrying. " + conn.LastErrorMessage
	case accountingsync.ConnectionStatusFailing:
		severity = watchtower.SeverityCritical
		title = provider + " connection is failing"
		summary = "Several checks in a row failed, so nothing reaches your books until it recovers. " +
			conn.LastErrorMessage
	case accountingsync.ConnectionStatusRevoked:
		severity = watchtower.SeverityCritical
		title = provider + " access was revoked"
		summary = "The authorization was revoked or expired. Reconnect " + provider +
			" to resume sync; nothing is lost while it is disconnected."
		if conn.UsesCertificate() {
			summary = provider + " refused Trenova's certificate. If the account was refreshed " +
				"from production, set up the certificate mapping again, then renew the " +
				"certificate here; nothing is lost while it is disconnected."
		}
	default:
		return watchtowerports.WatchtowerItemInput{}, false
	}

	occurredAt := conn.UpdatedAt
	if conn.LastFailureAt != nil {
		occurredAt = *conn.LastFailureAt
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID},
		SourceKind:  watchtower.SourceAccountingSync,
		SourceID:    AccountingConnectionHealthSourceID(conn),
		Severity:    severity,
		Title:       title,
		Summary:     stringutils.OneLine(summary, 500),
		SubjectType: agent.SubjectAccountingConnection,
		SubjectID:   conn.ID,
		EventKind:   agent.EventAccountingConnectionDegraded,
		Path:        accountingsync.SetupPath(conn.IntegrationType),
		OccurredAt:  occurredAt,
	}, true
}

func DescribeAccountingReconnect(
	conn *accountingsync.AccountingConnection,
	now int64,
) (watchtowerports.WatchtowerItemInput, bool) {
	if !conn.IsActive() || !conn.RefreshTokenNearAbsoluteExpiry(now) {
		return watchtowerports.WatchtowerItemInput{}, false
	}

	provider := accountingsync.ProviderName(conn.IntegrationType)
	expires := time.Unix(conn.RefreshTokenAbsoluteExpiresAt, 0).UTC().Format(reconnectDateLayout)
	title := "Reconnect " + provider + " before its authorization expires"
	summary := "The " + provider + " authorization expires on " + expires +
		" and cannot be extended. Reconnect before it lapses so sync does not stop."
	if conn.UsesCertificate() {
		title = "Renew the " + provider + " certificate before it expires"
		summary = "The certificate Trenova connects to " + provider + " with expires on " +
			expires + ". Generate a new certificate, upload it to the same mapping in " +
			provider + ", and renew it here so sync does not stop."
	}
	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID},
		SourceKind:  watchtower.SourceAccountingSync,
		SourceID:    AccountingReconnectSourceID(conn),
		Severity:    watchtower.SeverityWarning,
		Title:       title,
		Summary:     summary,
		SubjectType: agent.SubjectAccountingConnection,
		SubjectID:   conn.ID,
		EventKind:   agent.EventAccountingConnectionDegraded,
		Path:        accountingsync.SetupPath(conn.IntegrationType),
		OccurredAt:  conn.RefreshTokenAbsoluteExpiresAt - conn.ReconnectWarningSeconds(),
	}, true
}
