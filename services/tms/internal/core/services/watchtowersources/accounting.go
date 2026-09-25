package watchtowersources

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const accountingReconnectSuffix = ":reconnect"

type AccountingSyncSource struct {
	repo repositories.AccountingConnectionRepository
}

func NewAccountingSyncSource(
	repo repositories.AccountingConnectionRepository,
) services.WatchtowerSource {
	return &AccountingSyncSource{repo: repo}
}

func (s *AccountingSyncSource) Kind() watchtower.SourceKind {
	return watchtower.SourceAccountingSync
}

func (s *AccountingSyncSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	connections, err := s.repo.ListByTenant(ctx, tenant)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	items := make([]services.WatchtowerItemInput, 0, len(connections))
	for _, conn := range connections {
		if item, open := DescribeAccountingConnectionHealth(conn); open {
			items = append(items, item)
		}
		if item, open := DescribeAccountingReconnect(conn, now); open {
			items = append(items, item)
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
) (services.WatchtowerItemInput, bool) {
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
	default:
		return services.WatchtowerItemInput{}, false
	}

	occurredAt := conn.UpdatedAt
	if conn.LastFailureAt != nil {
		occurredAt = *conn.LastFailureAt
	}

	return services.WatchtowerItemInput{
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
) (services.WatchtowerItemInput, bool) {
	if !conn.IsActive() || !conn.RefreshTokenNearAbsoluteExpiry(now) {
		return services.WatchtowerItemInput{}, false
	}

	provider := accountingsync.ProviderName(conn.IntegrationType)
	return services.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID},
		SourceKind: watchtower.SourceAccountingSync,
		SourceID:   AccountingReconnectSourceID(conn),
		Severity:   watchtower.SeverityWarning,
		Title:      "Reconnect " + provider + " before its authorization expires",
		Summary: provider + " authorizations last five years from the day they were granted. " +
			"Reconnect before it lapses so sync does not stop.",
		SubjectType: agent.SubjectAccountingConnection,
		SubjectID:   conn.ID,
		EventKind:   agent.EventAccountingConnectionDegraded,
		Path:        accountingsync.SetupPath(conn.IntegrationType),
		OccurredAt:  conn.RefreshTokenAbsoluteExpiresAt - accountingsync.RefreshTokenWarningWindow,
	}, true
}
