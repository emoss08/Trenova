package auditlogresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditreviewservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

const requestCountSlack = int64(60 * 60)

func (r *Deps) setReviewed(
	ctx context.Context,
	id string,
	reviewed bool,
) (*auditreviewservice.View, error) {
	authCtx, err := r.RequirePermission(
		ctx,
		permission.ResourceSecurityAuditLog,
		permission.OpUpdate,
	)
	if err != nil {
		return nil, err
	}

	reviewID, err := pulid.MustParse(id)
	if err != nil {
		return nil, err
	}

	return r.AuditReviewService.SetReviewed(ctx, &auditreviewservice.SetReviewedRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		Actor:      actorutil.FromAuthContext(authCtx).AuditActor(),
		ID:         reviewID,
		Reviewed:   reviewed,
	})
}

func (r *Deps) countRequests(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	access auditports.EntryAccess,
	entries []*audit.Entry,
) error {
	if !requestCountRequested(ctx) || len(entries) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(entries))
	ids := make([]string, 0, len(entries))
	from, to := entries[0].Timestamp, entries[0].Timestamp
	for _, entry := range entries {
		if entry.CorrelationID == "" {
			continue
		}
		from, to = min(from, entry.Timestamp), max(to, entry.Timestamp)
		if _, ok := seen[entry.CorrelationID]; ok {
			continue
		}
		seen[entry.CorrelationID] = struct{}{}
		ids = append(ids, entry.CorrelationID)
	}
	if len(ids) == 0 {
		return nil
	}

	counts, err := r.AuditService.CountByRequest(ctx, &auditports.CountByRequestRequest{
		TenantInfo:     resolvercore.TenantInfo(authCtx),
		Access:         access,
		CorrelationIDs: ids,
		From:           from - requestCountSlack,
		To:             to + requestCountSlack,
	})
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entry.RequestEntryCount = counts[entry.CorrelationID]
	}

	return nil
}
