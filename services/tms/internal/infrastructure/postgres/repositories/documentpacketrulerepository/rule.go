package documentpacketrulerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
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

func New(p Params) repositories.DocumentPacketRuleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-packet-rule-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDocumentPacketRulesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.DocumentPacketRuleTable.Alias,
		req.Filter,
		(*documentpacketrule.DocumentPacketRule)(nil),
	)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListDocumentPacketRulesRequest,
) (*pagination.ListResult[*documentpacketrule.DocumentPacketRule], error) {
	items := make([]*documentpacketrule.DocumentPacketRule, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&items).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery { return r.filterQuery(sq, req) }).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*documentpacketrule.DocumentPacketRule]{
		Items: items,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetDocumentPacketRuleByIDRequest,
) (*documentpacketrule.DocumentPacketRule, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(documentpacketrule.DocumentPacketRule)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentPacketRuleScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.DocumentPacketRuleColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get document packet rule", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DocumentPacketRule")
	}
	return entity, nil
}

func (r *repository) ListByResourceType(
	ctx context.Context,
	req *repositories.ListDocumentPacketRulesByResourceRequest,
) ([]*documentpacketrule.DocumentPacketRule, error) {
	items := make([]*documentpacketrule.DocumentPacketRule, 0)

	cols := buncolgen.DocumentPacketRuleColumns

	err := r.db.DB().
		NewSelect().
		Model(&items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentPacketRuleScopeTenant(sq, req.TenantInfo).
				Where(cols.ResourceType.Eq(), req.ResourceType)
		}).
		OrderExpr(cols.DisplayOrder.OrderAsc(), cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *documentpacketrule.DocumentPacketRule,
) (*documentpacketrule.DocumentPacketRule, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("id", entity.ID.String()),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create document packet rule", zap.Error(err))
		return nil, err
	}
	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *documentpacketrule.DocumentPacketRule,
) (*documentpacketrule.DocumentPacketRule, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	result, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.DocumentPacketRuleColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update document packet rule", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "DocumentPacketRule", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.GetDocumentPacketRuleByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
	)

	result, err := r.db.DB().
		NewDelete().
		Model((*documentpacketrule.DocumentPacketRule)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.DocumentPacketRuleScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.DocumentPacketRuleColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete document packet rule", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "DocumentPacketRule", req.ID.String())
}
