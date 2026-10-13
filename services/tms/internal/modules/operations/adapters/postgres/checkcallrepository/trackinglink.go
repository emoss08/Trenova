package checkcallrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/checkcallcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
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
		cols := checkcallcol.TrackingLinkColumns
		entity := new(checkcall.TrackingLink)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			Where(cols.TokenHash.Eq(), tokenHash).
			Scan(ctx)
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
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	now int64,
) (*checkcall.TrackingLink, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*checkcall.TrackingLink, error) {
		cols := checkcallcol.TrackingLinkColumns
		entities := make([]*checkcall.TrackingLink, 0, 1)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return checkcallcol.TrackingLinkScopeTenant(sq, tenantInfo).
					Where(cols.ShipmentMoveID.Eq(), moveID).
					Where(cols.RevokedAt.IsNull()).
					Where(cols.ExpiresAt.Gt(), now)
			}).
			Order(cols.CreatedAt.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("get active carrier tracking link: %w", err)
		}
		if len(entities) == 0 {
			return nil, nil
		}
		return entities[0], nil
	})
}

func (r *linkRepository) RevokeForMove(
	ctx context.Context,
	req *checkcallports.RevokeTrackingLinksRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := checkcallcol.TrackingLinkColumns
		result, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*checkcall.TrackingLink)(nil)).
			Set(cols.RevokedAt.Set(), req.At).
			Set(cols.UpdatedAt.Set(), req.At).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return checkcallcol.TrackingLinkScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ShipmentMoveID.Eq(), req.MoveID).
					Where(cols.RevokedAt.IsNull())
			}).
			Exec(ctx)
		if err != nil {
			return 0, fmt.Errorf("revoke carrier tracking links: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("revoke carrier tracking links rows affected: %w", err)
		}
		return int(rows), nil
	})
}

func (r *linkRepository) Touch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	at int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := checkcallcol.TrackingLinkColumns
		_, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*checkcall.TrackingLink)(nil)).
			Set(cols.LastUsedAt.Set(), at).
			Set(cols.UpdatedAt.Set(), at).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return checkcallcol.TrackingLinkScopeTenantUpdate(uq, tenantInfo).
					Where(cols.ID.Eq(), id)
			}).
			Exec(ctx)
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
		cols := checkcallcol.TrackingLinkColumns
		deadAt := buncolgen.Expr("COALESCE({0}, {1})", cols.RevokedAt, cols.ExpiresAt)
		dba := r.db.DBForContext(ctx)
		dead := dba.NewSelect().
			Model((*checkcall.TrackingLink)(nil)).
			Column(cols.ID.Bare()).
			Where(deadAt+" < ?", before).
			Limit(int64(limit))
		result, err := dba.NewDelete().
			Model((*checkcall.TrackingLink)(nil)).
			Where(cols.ID.Expr("{} IN (?)"), dead).
			Exec(ctx)
		if err != nil {
			return 0, fmt.Errorf("purge dead carrier tracking links: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("purge dead carrier tracking links rows affected: %w", err)
		}
		return int(rows), nil
	})
}
