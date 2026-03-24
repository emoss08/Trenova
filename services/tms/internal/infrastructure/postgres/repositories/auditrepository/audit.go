package auditrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

func New(p Params) repositories.AuditRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.audit-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *pagination.QueryOptions,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"ae",
		req,
		(*audit.Entry)(nil),
	)

	q = q.Relation("User")
	q = q.Relation("APIKey")

	return q.Limit(req.Pagination.Limit).Offset(req.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*audit.Entry, 0, req.Filter.Pagination.Limit)

	total, err := r.db.DB().NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req.Filter)
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

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.Any("request", req),
	)

	entity := new(audit.Entry)

	q := r.db.DB().NewSelect().Model(entity).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("ae.id = ?", req.EntryID).
				Where("ae.organization_id = ?", req.TenantInfo.OrgID).
				Where("ae.business_unit_id = ?", req.TenantInfo.BuID)
		})

	q = q.Relation("User")
	q = q.Relation("APIKey")

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get audit entry", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Audit Entry")
	}

	return entity, nil
}

func (r *repository) ListByResourceID(
	ctx context.Context,
	req *repositories.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	log := r.l.With(
		zap.String("operation", "ListByResourceID"),
		zap.String("resourceID", req.ResourceID.String()),
	)

	entities := make([]*audit.Entry, 0, req.Filter.Pagination.Limit)

	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Where("ae.resource_id = ?", req.ResourceID).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req.Filter)
		})

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

	_, err := r.db.DB().NewInsert().Model(&entries).Exec(ctx)
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

	entries := make([]*audit.Entry, 0)

	q := r.db.DB().NewSelect().Model(&entries).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ae.resource = ?", req.Resource).
				Where("ae.resource_id = ?", req.ResourceID).
				Where("ae.operation = ?", req.Operation).
				Where("ae.organization_id = ?", req.OrganizationID)
		}).
		Order("ae.timestamp ASC").
		Relation("User").
		Relation("APIKey")

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
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

	entries := make([]*audit.Entry, 0, req.Limit)

	q := r.db.DB().NewSelect().Model(&entries).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ae.timestamp > ?", req.SinceTimestamp).
				Where("ae.operation = ?", req.Operation)
		}).
		Order("ae.timestamp ASC").
		Relation("User").
		Relation("APIKey")

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
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

	result, err := r.db.DB().NewDelete().Model((*audit.Entry)(nil)).
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
