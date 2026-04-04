package formulatemplaterepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
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

func New(p Params) repositories.FormulaTemplateRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.formula-template-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListFormulaTemplatesRequest,
) *bun.SelectQuery {
	log := r.l.With(
		zap.String("operation", "filterQuery"),
		zap.Any("req", req),
	)

	q = querybuilder.ApplyFilters(q, "ft", req.Filter, (*formulatemplate.FormulaTemplate)(nil))

	if req.Type != "" {
		t, err := formulatemplate.TemplateTypeFromString(req.Type)
		if err != nil {
			log.Error("failed to parse template type", zap.Error(err))
			return q
		}

		q = q.Where("ft.type = ?", t)
	}

	if req.Status != "" {
		s, err := formulatemplate.StatusFromString(req.Status)
		if err != nil {
			log.Error("failed to parse template status", zap.Error(err))
			return q
		}

		q = q.Where("ft.status = ?", s)
	}

	q = q.Order("ft.created_at DESC")

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListFormulaTemplatesRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*formulatemplate.FormulaTemplate, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count formula templates", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*formulatemplate.FormulaTemplate]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	_, err := r.db.DB().NewInsert().Model(entity).Exec(ctx)
	if err != nil {
		log.Error("failed to create formula template", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update formula template", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "FormulaTemplate", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("templateID", req.TemplateID.String()),
	)

	entity := new(formulatemplate.FormulaTemplate)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("ft.id = ?", req.TemplateID).
				Where("ft.organization_id = ?", req.TenantInfo.OrgID).
				Where("ft.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "FormulaTemplate")
	}

	return entity, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetFormulaTemplatesByIDsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*formulatemplate.FormulaTemplate, 0, len(req.TemplateIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("ft.organization_id = ?", req.TenantInfo.OrgID).
				Where("ft.business_unit_id = ?", req.TenantInfo.BuID).
				Where("ft.id IN (?)", bun.List(req.TemplateIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get formula templates", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "FormulaTemplate")
	}

	return entities, nil
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateFormulaTemplateStatusRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*formulatemplate.FormulaTemplate, 0, len(req.TemplateIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("ft.organization_id = ?", req.TenantInfo.OrgID).
				Where("ft.business_unit_id = ?", req.TenantInfo.BuID).
				Where("ft.id IN (?)", bun.List(req.TemplateIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update formula template status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "FormulaTemplate", req.TemplateIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) BulkDuplicate(
	ctx context.Context,
	req *repositories.BulkDuplicateFormulaTemplateRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "BulkDuplicate"),
		zap.Any("request", req),
	)

	entities, err := r.GetByIDs(ctx, repositories.GetFormulaTemplatesByIDsRequest{
		TemplateIDs: req.TemplateIDs,
		TenantInfo:  req.TenantInfo,
	})
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, err
	}

	// Bulk insert new formula templates
	var newEntities []*formulatemplate.FormulaTemplate

	for _, e := range entities {
		newEntities = append(newEntities, &formulatemplate.FormulaTemplate{
			OrganizationID:      e.OrganizationID,
			BusinessUnitID:      e.BusinessUnitID,
			Name:                fmt.Sprintf("%s (Copy)", e.Name),
			Description:         e.Description,
			Type:                e.Type,
			Expression:          e.Expression,
			Status:              e.Status,
			SchemaID:            e.SchemaID,
			Version:             e.Version,
			VariableDefinitions: e.VariableDefinitions,
			Metadata:            e.Metadata,
		})
	}

	results, err := r.db.DB().NewInsert().Model(&newEntities).Returning("*").Exec(ctx)
	if err != nil {
		log.Error("failed to bulk insert formula templates", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "FormulaTemplate", req.TemplateIDs); err != nil {
		return nil, err
	}

	return newEntities, nil
}

func (r *repository) CountUsages(
	ctx context.Context,
	req *repositories.GetTemplateUsageRequest,
) (*repositories.GetTemplateUsageResponse, error) {
	log := r.l.With(
		zap.String("operation", "CountUsages"),
		zap.String("templateID", req.TemplateID.String()),
	)

	type usageResult struct {
		Type  string `bun:"type"`
		Count int    `bun:"count"`
	}

	shipmentUsage := r.db.DB().NewSelect().
		ColumnExpr("'shipment' as type").
		ColumnExpr("COUNT(*) as count").
		TableExpr("shipments").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("formula_template_id = ?", req.TemplateID).
				Where("organization_id = ?", req.TenantInfo.OrgID).
				Where("business_unit_id = ?", req.TenantInfo.BuID)
		})

	accessorialUsage := r.db.DB().NewSelect().
		ColumnExpr("'accessorial_charge' as type").
		ColumnExpr("COUNT(*) as count").
		TableExpr("accessorial_charges").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("formula_template_id = ?", req.TemplateID).
				Where("organization_id = ?", req.TenantInfo.OrgID).
				Where("business_unit_id = ?", req.TenantInfo.BuID)
		})

	var results []usageResult
	err := r.db.DB().NewSelect().
		TableExpr("(?) AS shipment_usage", shipmentUsage).
		UnionAll(accessorialUsage).
		Scan(ctx, &results)
	if err != nil {
		log.Error("failed to count usages", zap.Error(err))
		return nil, err
	}

	usages := make([]repositories.TemplateUsageCount, 0, len(results))
	inUse := false
	for _, res := range results {
		if res.Count > 0 {
			inUse = true
			usages = append(usages, repositories.TemplateUsageCount{
				Type:  res.Type,
				Count: res.Count,
			})
		}
	}

	return &repositories.GetTemplateUsageResponse{
		InUse:  inUse,
		Usages: usages,
	}, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.FormulaTemplateSelectOptionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return dbhelper.SelectOptions[*formulatemplate.FormulaTemplate](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"name",
				"description",
				"expression",
			},
			OrgColumn: "ft.organization_id",
			BuColumn:  "ft.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("ft.status = ?", formulatemplate.StatusActive.String())
			},
			EntityName: "FormulaTemplate",
			SearchColumns: []string{
				"ft.name",
				"ft.description",
			},
		},
	)
}
