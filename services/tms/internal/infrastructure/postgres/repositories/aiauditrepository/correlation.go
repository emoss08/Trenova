package aiauditrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
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
	req *repositories.ListCorrelatedAuditEntriesRequest,
) ([]*audit.Entry, error) {
	matches := make([]repositories.AuditEntryMatch, 0, len(req.Matches))
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

	cols := buncolgen.EntryColumns
	condition := buncolgen.Expr(
		"{0} = ? AND ({1} = ? OR {2} = ?) AND {3} BETWEEN ? AND ?",
		cols.ResourceID, cols.PrincipalID, cols.UserID, cols.Timestamp,
	)

	entries := make([]*audit.Entry, 0, len(matches))
	err := r.db.DBForContext(ctx).NewSelect().
		Model(&entries).
		Apply(buncolgen.EntryApplyTenant(req.TenantInfo)).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			for _, match := range matches {
				end := match.WindowEnd
				if end < match.WindowStart {
					end = match.WindowStart
				}
				q = q.WhereOr(condition,
					match.ResourceID,
					match.PrincipalID,
					match.PrincipalID,
					match.WindowStart-correlationSlackSeconds,
					end+correlationSlackSeconds,
				)
			}

			return q
		}).
		Order(cols.Timestamp.OrderAsc(), cols.ID.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list audit entries correlated with the AI audit trail: %w", err)
	}

	return entries, nil
}
