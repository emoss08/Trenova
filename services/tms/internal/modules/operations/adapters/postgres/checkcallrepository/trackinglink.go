package checkcallrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall/checkcalldb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const defaultLinkPurgeLimit = 500

type linkRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewTrackingLinks(p Params) checkcallports.TrackingLinkRepository {
	return &linkRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-tracking-link-repository"),
	}
}

func (r *linkRepository) Create(ctx context.Context, entity *checkcall.TrackingLink) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := checkcalldb.TrackingLinks.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return fmt.Errorf("insert carrier tracking link: %w", err)
		}
		return nil
	})
}

func (r *linkRepository) GetByHash(
	ctx context.Context,
	tokenHash string,
) (*checkcall.TrackingLink, error) {
	ctx = dbscope.WithSystem(ctx, "resolve a carrier tracking link before its tenant is known")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*checkcall.TrackingLink, error) {
		tl := checkcalldb.TrackingLinks
		entity, err := tl.AllTenants().Where(tl.TokenHash.Eq(tokenHash)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("get carrier tracking link: %w", err)
		}
		return entity, nil
	})
}

func (r *linkRepository) GetActiveForMove(
	ctx context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
	now int64,
) (*checkcall.TrackingLink, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*checkcall.TrackingLink, error) {
		tl := checkcalldb.TrackingLinks
		entity, err := tl.Where(
			tl.ShipmentMoveID.Eq(moveID),
			tl.RevokedAt.IsNull(),
			tl.ExpiresAt.Gt(now),
		).
			Order(tl.CreatedAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("get active carrier tracking link: %w", err)
		}
		return entity, nil
	})
}

func (r *linkRepository) RevokeForMove(
	ctx context.Context,
	req *checkcallports.RevokeTrackingLinksRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		tl := checkcalldb.TrackingLinks
		rows, err := tl.Where(tl.ShipmentMoveID.Eq(req.MoveID), tl.RevokedAt.IsNull()).
			Update(ctx, r.db.Keel(ctx), tl.RevokedAt.To(req.At), tl.UpdatedAt.To(req.At))
		if err != nil {
			return 0, fmt.Errorf("revoke carrier tracking links: %w", err)
		}
		return int(rows), nil
	})
}

func (r *linkRepository) Touch(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	at int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		tl := checkcalldb.TrackingLinks
		_, err := tl.Where(tl.ID.Eq(id)).
			Update(ctx, r.db.Keel(ctx), tl.LastUsedAt.To(at), tl.UpdatedAt.To(at))
		if err != nil {
			return fmt.Errorf("touch carrier tracking link: %w", err)
		}
		return nil
	})
}

func (r *linkRepository) PurgeDead(ctx context.Context, before int64, limit int) (int, error) {
	ctx = dbscope.WithSystem(ctx, "purge dead carrier tracking links across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if limit <= 0 {
			limit = defaultLinkPurgeLimit
		}

		tl := checkcalldb.TrackingLinks
		dead := tl.As("dead")
		rows, err := tl.AllTenants().
			Where(tl.ID.InQuery(
				dead.AllTenants().
					Pick(dead.ID).
					Where(keel.DynOf(dead.RevokedAt).Coalesce(dead.ExpiresAt).Lt(before)).
					Limit(limit),
			)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("purge dead carrier tracking links: %w", err)
		}
		return int(rows), nil
	})
}
