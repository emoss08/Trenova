package customertrackingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/customertrackingcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type linkRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewLinks(p Params) customertrackingports.LinkRepository {
	return &linkRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.customer-tracking-link-repository"),
	}
}

func (r *linkRepository) Create(ctx context.Context, entity *customertracking.Link) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
			return fmt.Errorf("insert customer tracking link: %w", err)
		}
		return nil
	})
}

func (r *linkRepository) GetByHash(
	ctx context.Context,
	tokenHash string,
) (*customertracking.Link, error) {
	ctx = dbscope.WithSystem(ctx, "resolve a customer tracking link before its tenant is known")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*customertracking.Link, error) {
		cols := customertrackingcol.LinkColumns
		entity := new(customertracking.Link)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			Where(cols.TokenHash.Eq(), tokenHash).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("get customer tracking link: %w", err)
		}
		return entity, nil
	})
}

func (r *linkRepository) GetByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*customertracking.Link, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*customertracking.Link, error) {
		cols := customertrackingcol.LinkColumns
		entity := new(customertracking.Link)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return customertrackingcol.LinkScopeTenant(sq, tenantInfo).
					Where(cols.ID.Eq(), id)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Customer tracking link")
		}
		return entity, nil
	})
}

func (r *linkRepository) ListForShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*customertracking.Link, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*customertracking.Link, error) {
		cols := customertrackingcol.LinkColumns
		entities := make([]*customertracking.Link, 0, 4)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return customertrackingcol.LinkScopeTenant(sq, tenantInfo).
					Where(cols.ShipmentID.Eq(), shipmentID)
			}).
			Order(cols.CreatedAt.OrderDesc(), cols.ID.OrderDesc()).
			Limit(maxShipmentLinks).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list customer tracking links: %w", err)
		}
		return entities, nil
	})
}

func (r *linkRepository) Revoke(
	ctx context.Context,
	req *customertrackingports.RevokeLinksRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := customertrackingcol.LinkColumns
		result, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*customertracking.Link)(nil)).
			Set(cols.RevokedAt.Set(), req.At).
			Set(cols.UpdatedAt.Set(), req.At).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				uq = customertrackingcol.LinkScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ShipmentID.Eq(), req.ShipmentID).
					Where(cols.RevokedAt.IsNull())
				if !req.LinkID.IsNil() {
					uq = uq.Where(cols.ID.Eq(), req.LinkID)
				}
				return uq
			}).
			Exec(ctx)
		if err != nil {
			return 0, fmt.Errorf("revoke customer tracking links: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("revoke customer tracking links rows affected: %w", err)
		}
		return int(rows), nil
	})
}

func (r *linkRepository) RecordView(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	at int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := customertrackingcol.LinkColumns
		_, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*customertracking.Link)(nil)).
			Set(cols.LastViewedAt.Set(), at).
			Set(cols.ViewCount.Inc(1)).
			Set(cols.UpdatedAt.Set(), at).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return customertrackingcol.LinkScopeTenantUpdate(uq, tenantInfo).
					Where(cols.ID.Eq(), id)
			}).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("record customer tracking link view: %w", err)
		}
		return nil
	})
}

func (r *linkRepository) PurgeDead(ctx context.Context, before int64, limit int) (int, error) {
	ctx = dbscope.WithSystem(ctx, "purge dead customer tracking links across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if limit <= 0 {
			limit = defaultLinkPurgeLimit
		}
		cols := customertrackingcol.LinkColumns
		deadAt := buncolgen.Expr("COALESCE({0}, {1})", cols.RevokedAt, cols.ExpiresAt)
		dba := r.db.DBForContext(ctx)
		dead := dba.NewSelect().
			Model((*customertracking.Link)(nil)).
			Column(cols.ID.Bare()).
			Where(deadAt+" < ?", before).
			Limit(int64(limit))
		result, err := dba.NewDelete().
			Model((*customertracking.Link)(nil)).
			Where(cols.ID.Expr("{} IN (?)"), dead).
			Exec(ctx)
		if err != nil {
			return 0, fmt.Errorf("purge dead customer tracking links: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("purge dead customer tracking links rows affected: %w", err)
		}
		return int(rows), nil
	})
}
