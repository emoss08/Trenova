package customertrackingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking/customertrackingdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var linkErrors = dberror.Mapping{Entity: "Customer tracking link"}

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
		if err := customertrackingdb.Links.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return fmt.Errorf(
				"insert customer tracking link: %w",
				linkErrors.Map(err, entity.ID.String()),
			)
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
		l := customertrackingdb.Links
		entity, err := l.AllTenants().Where(l.TokenHash.Eq(tokenHash)).One(ctx, r.db.Keel(ctx))
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
	_ pagination.TenantInfo,
	id pulid.ID,
) (*customertracking.Link, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*customertracking.Link, error) {
		entity, err := customertrackingdb.Links.Get(ctx, r.db.Keel(ctx), id)
		if err != nil {
			return nil, linkErrors.Map(err, id.String())
		}
		return entity, nil
	})
}

func (r *linkRepository) ListForShipment(
	ctx context.Context,
	_ pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*customertracking.Link, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*customertracking.Link, error) {
		l := customertrackingdb.Links
		entities, err := l.Where(l.ShipmentID.Eq(shipmentID)).
			Order(l.CreatedAt.Desc(), l.ID.Desc()).
			Limit(maxShipmentLinks).
			List(ctx, r.db.Keel(ctx))
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
		l := customertrackingdb.Links
		rows, err := l.Where(
			l.ShipmentID.Eq(req.ShipmentID),
			l.RevokedAt.IsNull(),
			keel.When(!req.LinkID.IsNil(), l.ID.Eq(req.LinkID)),
		).Update(ctx, r.db.Keel(ctx), l.RevokedAt.To(req.At), l.UpdatedAt.To(req.At))
		if err != nil {
			return 0, fmt.Errorf("revoke customer tracking links: %w", err)
		}
		return int(rows), nil
	})
}

func (r *linkRepository) RecordView(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	at int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		l := customertrackingdb.Links
		_, err := l.Where(l.ID.Eq(id)).Update(
			ctx,
			r.db.Keel(ctx),
			l.LastViewedAt.To(at),
			l.ViewCount.Add(1),
			l.UpdatedAt.To(at),
		)
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

		l := customertrackingdb.Links
		dead := l.As("dead")
		rows, err := l.AllTenants().
			Where(l.ID.InQuery(
				dead.AllTenants().
					Pick(dead.ID).
					Where(dead.RevokedAt.Coalesce(dead.ExpiresAt).Lt(before)).
					Limit(limit),
			)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("purge dead customer tracking links: %w", err)
		}
		return int(rows), nil
	})
}
