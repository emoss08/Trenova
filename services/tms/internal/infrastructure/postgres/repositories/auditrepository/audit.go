package auditrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRepository(p Params) repositories.AuditRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.audit-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	opts *pagination.QueryOptions,
) *bun.SelectQuery {
	qb := querybuilder.New(q, "ae")
	qb.ApplyTenantFilters(opts.TenantOpts)

	q = qb.GetQuery()

	q = q.Relation("User")

	q = q.Order("ae.timestamp DESC")

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

func (r *repository) GetByID(
	ctx context.Context,
	opts repositories.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("auditEntryID", opts.ID.String()),
		zap.String("organizationID", opts.OrgID.String()),
		zap.String("businessUnitID", opts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	entity := new(audit.Entry)

	q := db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("ae.id = ?", opts.ID).
				Where("ae.organization_id = ?", opts.OrgID).
				Where("ae.business_unit_id = ?", opts.BuID)
		})

	q = q.Relation("User")

	if err = q.Scan(ctx); err != nil {
		log.Error("failed to get audit entry", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Audit Entry")
	}

	return entity, nil
}

func (r *repository) List(
	ctx context.Context,
	opts *pagination.QueryOptions,
) (*pagination.ListResult[*audit.Entry], error) {
	log := r.l.With(zap.String("operation", "List"))
	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	entities := make([]*audit.Entry, 0, opts.Limit)

	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, opts)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan audit entries", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*audit.Entry]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) ListByResourceID(
	ctx context.Context,
	opts repositories.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	log := r.l.With(
		zap.String("operation", "ListByResourceID"),
		zap.String("resourceID", opts.ResourceID.String()),
	)

	// TODO(Wolfred): We need to add a limit offset to this query
	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}
	entities := make([]*audit.Entry, 0)

	q := db.NewSelect().Model(&entities).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("ae.resource_id = ?", opts.ResourceID).
				Where("ae.organization_id = ?", opts.OrgID).
				Where("ae.business_unit_id = ?", opts.BuID)
		}).Relation("User")

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan audit entries", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*audit.Entry]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) InsertAuditEntries(ctx context.Context, entries []*audit.Entry) error {
	log := r.l.With(zap.String("operation", "InsertAuditEntries"))

	db, err := r.db.DB(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewInsert().Model(&entries).Exec(ctx)
	if err != nil {
		log.Error("failed to insert audit entries", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetByResourceAndOperation(
	ctx context.Context,
	req *repositories.GetAuditByResourceRequest,
) ([]*audit.Entry, error) {
	log := r.l.With(zap.String("operation", "GetByResourceAndOperation"))

	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]*audit.Entry, 0)

	q := db.NewSelect().Model(&entries).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ae.resource = ?", req.Resource).
				Where("ae.resource_id = ?", req.ResourceID).
				Where("ae.operation = ?", req.Operation).
				Where("ae.organization_id = ?", req.OrganizationID)
		}).
		Order("ae.timestamp ASC").
		Relation("User")

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err = q.Scan(ctx); err != nil {
		log.Error("failed to get audit entries by resource and operation", zap.Error(err))
		return nil, err
	}

	return entries, nil
}

func (r *repository) GetRecentEntries(
	ctx context.Context,
	req *repositories.GetRecentEntriesRequest,
) ([]*audit.Entry, error) {
	log := r.l.With(zap.String("operation", "GetRecentEntries"))

	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]*audit.Entry, 0, req.Limit)

	q := db.NewSelect().Model(&entries).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ae.timestamp > ?", req.SinceTimestamp).
				Where("ae.operation = ?", req.Operation)
		}).
		Order("ae.timestamp ASC").
		Relation("User")

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err = q.Scan(ctx); err != nil {
		log.Error("failed to get recent audit entries", zap.Error(err))
		return nil, err
	}

	return entries, nil
}

func (r *repository) DeleteAuditEntries(
	ctx context.Context,
	timestamp int64,
) (int64, error) {
	log := r.l.With(zap.String("operation", "DeleteAuditEntries"))

	db, err := r.db.DB(ctx)
	if err != nil {
		return 0, err
	}

	result, err := db.NewDelete().Model((*audit.Entry)(nil)).
		Where("ae.timestamp < ?", timestamp).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete audit entries", zap.Error(err))
		return 0, err
	}

	totalDeleted, err := result.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected", zap.Error(err))
		return 0, err
	}

	return totalDeleted, nil
}
