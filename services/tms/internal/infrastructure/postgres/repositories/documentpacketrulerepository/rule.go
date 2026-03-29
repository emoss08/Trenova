package documentpacketrulerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func (r *repository) filterQuery(q *bun.SelectQuery, req *repositories.ListDocumentPacketRulesRequest) *bun.SelectQuery {
	if req.ResourceType != "" {
		q = q.Where("dpr.resource_type = ?", req.ResourceType)
	}
	if req.Filter != nil && req.Filter.Query != "" {
		q = q.Where("CAST(dpr.document_type_id AS TEXT) ILIKE ?", "%"+req.Filter.Query+"%")
	}
	return q.Order("dpr.resource_type ASC, dpr.display_order ASC, dpr.created_at DESC").
		Limit(req.Filter.Pagination.Limit).
		Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListDocumentPacketRulesRequest,
) (*pagination.ListResult[*documentpacketrule.Rule], error) {
	items := make([]*documentpacketrule.Rule, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery { return r.filterQuery(sq, req) }).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*documentpacketrule.Rule]{
		Items: items,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetDocumentPacketRuleByIDRequest,
) (*documentpacketrule.Rule, error) {
	entity := new(documentpacketrule.Rule)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dpr.id = ?", req.ID).
		Where("dpr.organization_id = ?", req.TenantInfo.OrgID).
		Where("dpr.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Document packet rule")
	}
	return entity, nil
}

func (r *repository) ListByResourceType(
	ctx context.Context,
	req *repositories.ListDocumentPacketRulesByResourceRequest,
) ([]*documentpacketrule.Rule, error) {
	items := make([]*documentpacketrule.Rule, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dpr.organization_id = ?", req.TenantInfo.OrgID).
		Where("dpr.business_unit_id = ?", req.TenantInfo.BuID).
		Where("dpr.resource_type = ?", req.ResourceType).
		Order("dpr.display_order ASC, dpr.created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repository) Create(ctx context.Context, entity *documentpacketrule.Rule) (*documentpacketrule.Rule, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) Update(ctx context.Context, entity *documentpacketrule.Rule) (*documentpacketrule.Rule, error) {
	ov := entity.Version
	entity.Version++
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
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
	result, err := r.db.DBForContext(ctx).
		NewDelete().
		Table("document_packet_rules").
		Where("id = ?", req.ID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}

	if err = dberror.CheckRowsAffected(result, "DocumentPacketRule", req.ID.String()); err != nil {
		return err
	}
	return nil
}
