package aiauditrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
)

const (
	defaultCorrelationLimit = 500
	correlationSlackSeconds = 1
)

// ListCorrelatedAuditEntries finds the audit log rows written for the same
// record, by the same principal, inside each event's window widened by a
// second either side. The match is by time, not by a shared key: the audit
// log is written by every service and carries no reference to the trail.
func (r *repository) ListCorrelatedAuditEntries(
	ctx context.Context,
	req *aiauditports.ListCorrelatedAuditEntriesRequest,
) ([]*audit.Entry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*audit.Entry, error) {
		matches := make([]aiauditports.AuditEntryMatch, 0, len(req.Matches))
		for _, match := range req.Matches {
			if match.ResourceID == "" || match.PrincipalID.IsNil() || match.WindowStart <= 0 {
				continue
			}
			matches = append(matches, match)
		}
		if len(matches) == 0 {
			return []*audit.Entry{}, nil
		}

		limit := req.Limit
		if limit <= 0 {
			limit = defaultCorrelationLimit
		}

		e := auditdb.Entries
		windows := make([]keel.Pred, 0, len(matches))
		for _, match := range matches {
			end := max(match.WindowEnd, match.WindowStart)
			windows = append(windows, keel.And(
				e.ResourceID.Eq(match.ResourceID),
				keel.Or(e.PrincipalID.Eq(match.PrincipalID), e.UserID.Eq(match.PrincipalID)),
				e.Timestamp.Between(
					match.WindowStart-correlationSlackSeconds,
					end+correlationSlackSeconds,
				),
			))
		}

		entries, err := e.Where(keel.Or(windows...)).
			Order(e.Timestamp.Asc(), e.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list audit entries correlated with the AI audit trail: %w", err)
		}

		return entries, nil
	})
}
