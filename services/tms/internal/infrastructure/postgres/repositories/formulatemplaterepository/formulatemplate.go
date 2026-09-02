package formulatemplaterepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
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
			log.Warn("rejected unknown template type filter", zap.String("type", req.Type))
			return q.Err(errortypes.NewValidationError(
				"type",
				errortypes.ErrInvalid,
				"Unknown template type: "+req.Type,
			))
		}

		q = q.Where(buncolgen.FormulaTemplateColumns.Type.Eq(), t)
	}

	if req.Status != "" {
		s, err := formulatemplate.StatusFromString(req.Status)
		if err != nil {
			log.Warn("rejected unknown template status filter", zap.String("status", req.Status))
			return q.Err(errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalid,
				"Unknown template status: "+req.Status,
			))
		}

		q = q.Where(buncolgen.FormulaTemplateColumns.Status.Eq(), s)
	}

	q = q.Order(buncolgen.FormulaTemplateColumns.CreatedAt.OrderDesc())

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
	total, err := r.db.DBForContext(ctx).
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

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListFormulaTemplateConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		"ft",
		req.Filter,
		req.Cursor,
		(*formulatemplate.FormulaTemplate)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListFormulaTemplateConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		"ft",
		req.Filter,
		(*formulatemplate.FormulaTemplate)(nil),
	)
}

func applyFormulaTemplateColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListFormulaTemplateConnectionRequest,
) (*pagination.CursorListResult[*formulatemplate.FormulaTemplate], error) {
	log := r.l.With(
		zap.String("operation", "ListConnection"),
		zap.Any("request", req),
	)

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*formulatemplate.FormulaTemplate)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count formula templates", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*formulatemplate.FormulaTemplate]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*formulatemplate.FormulaTemplate) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyFormulaTemplateColumns(sq, req.FormulaTemplateColumns)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		})
	if err != nil {
		log.Error("failed to scan formula templates", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	_, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateTemplateName(entity.Name)
		}
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

	query := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.FormulaTemplateColumns.Version.Eq(), ov).
		OmitZero()

	results, err := applyClearableColumns(query, entity).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateTemplateName(entity.Name)
		}
		log.Error("failed to update formula template", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "FormulaTemplate", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func applyClearableColumns(
	q *bun.UpdateQuery,
	entity *formulatemplate.FormulaTemplate,
) *bun.UpdateQuery {
	cols := buncolgen.FormulaTemplateColumns
	return q.
		Value(cols.Description.String(), "?", entity.Description).
		Value(cols.ReviewComment.String(), "?", entity.ReviewComment).
		Value(cols.MinCharge.String(), "?", entity.MinCharge).
		Value(cols.MaxCharge.String(), "?", entity.MaxCharge).
		Value(cols.RoundingMode.String(), "?", entity.RoundingMode).
		Value(cols.RoundingPrecision.String(), "?", entity.RoundingPrecision).
		Value(cols.SubmittedByID.String(), "?", entity.SubmittedByID).
		Value(cols.SubmittedAt.String(), "?", entity.SubmittedAt).
		Value(cols.ApprovedByID.String(), "?", entity.ApprovedByID).
		Value(cols.ApprovedAt.String(), "?", entity.ApprovedAt)
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
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FormulaTemplateScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.FormulaTemplateColumns.ID.Eq(), req.TemplateID)
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
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FormulaTemplateScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.FormulaTemplateColumns.ID.In(), bun.List(req.TemplateIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get formula templates", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "FormulaTemplate")
	}

	return entities, nil
}

func (r *repository) FindByNames(
	ctx context.Context,
	req repositories.GetFormulaTemplatesByNamesRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "FindByNames"),
		zap.Any("request", req),
	)

	if len(req.Names) == 0 {
		return []*formulatemplate.FormulaTemplate{}, nil
	}

	entities := make([]*formulatemplate.FormulaTemplate, 0, len(req.Names))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FormulaTemplateScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.FormulaTemplateColumns.Name.In(), bun.List(req.Names))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to find formula templates by names", zap.Error(err))
		return nil, err
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
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.FormulaTemplateScopeTenantUpdate(uq, req.TenantInfo).
				Where(buncolgen.FormulaTemplateColumns.ID.In(), bun.List(req.TemplateIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update formula template status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(
		results,
		"FormulaTemplate",
		req.TemplateIDs,
	); err != nil {
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

	newEntities := make([]*formulatemplate.FormulaTemplate, 0, len(entities))
	taken, err := r.namesLike(ctx, req.TenantInfo, entities)
	if err != nil {
		log.Error("failed to read existing template names", zap.Error(err))
		return nil, err
	}

	for _, e := range entities {
		name := nextAvailableName(taken, e.Name+" (Copy)")
		taken[name] = struct{}{}
		seed := formulatemplate.SeedFromTemplate(e)
		seed.Name = name
		newEntities = append(newEntities, seed.Build())
	}

	results, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&newEntities).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk insert formula templates", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(
		results,
		"FormulaTemplate",
		req.TemplateIDs,
	); err != nil {
		return nil, err
	}

	return newEntities, nil
}

// namesLike collects every template name in the tenant that starts with a
// source's copy name, so the batch can pick names nothing already holds.
func (r *repository) namesLike(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sources []*formulatemplate.FormulaTemplate,
) (map[string]struct{}, error) {
	taken := make(map[string]struct{}, len(sources))
	if len(sources) == 0 {
		return taken, nil
	}

	ft := buncolgen.FormulaTemplateColumns
	var names []string
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model((*formulatemplate.FormulaTemplate)(nil)).
		Column(ft.Name.String()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.FormulaTemplateScopeTenant(sq, tenantInfo)
			return sq.WhereGroup(" AND ", func(orGroup *bun.SelectQuery) *bun.SelectQuery {
				for _, source := range sources {
					orGroup = orGroup.WhereOr(ft.Name.Like(), source.Name+" (Copy%")
				}
				return orGroup
			})
		})
	if err := query.Scan(ctx, &names); err != nil {
		return nil, err
	}

	for _, name := range names {
		taken[name] = struct{}{}
	}

	return taken, nil
}

// nextAvailableName returns base when it is free, otherwise "base 2",
// "base 3", and so on: a second duplicate of the same template gets its own
// name rather than colliding with the first.
func nextAvailableName(taken map[string]struct{}, base string) string {
	if _, exists := taken[base]; !exists {
		return base
	}

	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s %d", base, suffix)
		if _, exists := taken[candidate]; !exists {
			return candidate
		}
	}
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

	shipmentUsage := r.usageCount(ctx, req, "shipment",
		buncolgen.ShipmentTable,
		buncolgen.ShipmentColumns.FormulaTemplateID,
		buncolgen.ShipmentScopeTenant,
	)
	matrixUsage := r.usageCount(ctx, req, "rate_matrix",
		buncolgen.RateMatrixTable,
		buncolgen.RateMatrixColumns.FormulaTemplateID,
		buncolgen.RateMatrixScopeTenant,
	)
	ruleUsage := r.usageCount(ctx, req, "rate_agreement_rule",
		buncolgen.RateAgreementRuleTable,
		buncolgen.RateAgreementRuleColumns.FormulaTemplateID,
		buncolgen.RateAgreementRuleScopeTenant,
	)
	agreementAccessorialUsage := r.usageCount(ctx, req, "rate_agreement_accessorial",
		buncolgen.RateAgreementAccessorialTable,
		buncolgen.RateAgreementAccessorialColumns.FormulaTemplateID,
		buncolgen.RateAgreementAccessorialScopeTenant,
	)

	var results []usageResult
	err := r.db.DBForContext(ctx).NewSelect().
		TableExpr("(?) AS shipment_usage", shipmentUsage).
		UnionAll(matrixUsage).
		UnionAll(ruleUsage).
		UnionAll(agreementAccessorialUsage).
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
	cols := buncolgen.FormulaTemplateColumns

	return dbhelper.SelectOptions[*formulatemplate.FormulaTemplate](
		ctx,
		r.db.DBForContext(ctx),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				cols.ID.Name,
				cols.Name.Name,
				cols.Description.Name,
				cols.Expression.Name,
			},
			OrgColumn: cols.OrganizationID.Qualified(),
			BuColumn:  cols.BusinessUnitID.Qualified(),
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.Status.Eq(), formulatemplate.StatusActive.String())
			},
			EntityName: "FormulaTemplate",
			SearchColumns: []string{
				cols.Name.Qualified(),
				cols.Description.Qualified(),
			},
		},
	)
}

func duplicateTemplateName(name string) error {
	return errortypes.NewValidationError(
		"name",
		errortypes.ErrDuplicate,
		fmt.Sprintf("A formula template named %q already exists", name),
	)
}

// usageCount is one branch of the usage union: how many rows of a table point
// at the template, labelled so the caller can tell shipments from rules.
func (r *repository) usageCount(
	ctx context.Context,
	req *repositories.GetTemplateUsageRequest,
	label string,
	table buncolgen.TableInfo,
	templateColumn buncolgen.Column,
	scope func(*bun.SelectQuery, pagination.TenantInfo) *bun.SelectQuery,
) *bun.SelectQuery {
	return r.db.DBForContext(ctx).NewSelect().
		ColumnExpr("? as type", label).
		ColumnExpr("COUNT(*) as count").
		TableExpr("? AS ?", bun.Ident(table.Name), bun.Ident(table.Alias)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return scope(sq, req.TenantInfo).
				Where(templateColumn.Eq(), req.TemplateID)
		})
}
