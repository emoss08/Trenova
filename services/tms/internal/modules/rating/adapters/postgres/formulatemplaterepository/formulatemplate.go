package formulatemplaterepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen/formulatemplatecol"
	"github.com/emoss08/trenova/pkg/buncolgen/rateagreementcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) formulatemplateports.FormulaTemplateRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.formula-template-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *formulatemplateports.ListFormulaTemplatesRequest,
) *bun.SelectQuery {
	log := r.l.With(
		zap.String("operation", "filterQuery"),
		zap.Any("req", req),
	)

	q = querybuilder.ApplyFilters(q, "ft", req.Filter, (*formulatemplate.FormulaTemplate)(nil))

	if req.Status != "" {
		s, err := formulatemplate.StatusFromString(req.Status)
		if err != nil {
			log.Warn("rejected unknown template status filter", zap.String("status", req.Status))
			return q.Err(errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalid,
				"Unknown template status: {0}", req.Status,
			))
		}

		q = q.Where(formulatemplatecol.FormulaTemplateColumns.Status.Eq(), s)
	}

	q = q.Order(formulatemplatecol.FormulaTemplateColumns.CreatedAt.OrderDesc())

	return q.Limit(int64(req.Filter.Pagination.SafeLimit())).
		Offset(int64(req.Filter.Pagination.SafeOffset()))
}

func (r *repository) List(
	ctx context.Context,
	req *formulatemplateports.ListFormulaTemplatesRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
			log := r.l.With(
				zap.String("operation", "List"),
				zap.Any("request", req),
			)

			entities := make(
				[]*formulatemplate.FormulaTemplate,
				0,
				req.Filter.Pagination.SafeLimit(),
			)
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
				Total: int(total),
			}, nil
		},
	)
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *formulatemplateports.ListFormulaTemplateConnectionRequest,
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
	req *formulatemplateports.ListFormulaTemplateConnectionRequest,
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
	req *formulatemplateports.ListFormulaTemplateConnectionRequest,
) (*pagination.CursorListResult[*formulatemplate.FormulaTemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*formulatemplate.FormulaTemplate], error) {
			log := r.l.With(
				zap.String("operation", "ListConnection"),
				zap.Any("request", req),
			)

			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
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
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*formulatemplate.FormulaTemplate]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
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
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplate, error) {
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
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplate, error) {
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
				Where(formulatemplatecol.FormulaTemplateColumns.Version.Eq(), ov).
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

			if err = dberror.CheckRowsAffected(
				results,
				"FormulaTemplate",
				entity.ID.String(),
			); err != nil {
				return nil, err
			}

			return entity, nil
		},
	)
}

func applyClearableColumns(
	q *bun.UpdateQuery,
	entity *formulatemplate.FormulaTemplate,
) *bun.UpdateQuery {
	cols := formulatemplatecol.FormulaTemplateColumns
	return q.
		Value(cols.Description.String(), "?", entity.Description).
		Value(cols.ReviewComment.String(), "?", entity.ReviewComment).
		Value(cols.TestCount.String(), "?", entity.TestCount).
		Value(cols.SubmittedByID.String(), "?", entity.SubmittedByID).
		Value(cols.SubmittedAt.String(), "?", entity.SubmittedAt).
		Value(cols.ApprovedByID.String(), "?", entity.ApprovedByID).
		Value(cols.ApprovedAt.String(), "?", entity.ApprovedAt)
}

func (r *repository) GetByID(
	ctx context.Context,
	req formulatemplateports.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplate, error) {
			log := r.l.With(
				zap.String("operation", "GetByID"),
				zap.String("templateID", req.TemplateID.String()),
			)

			entity := new(formulatemplate.FormulaTemplate)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(entity).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulatemplatecol.FormulaTemplateScopeTenant(sq, req.TenantInfo).
						Where(formulatemplatecol.FormulaTemplateColumns.ID.Eq(), req.TemplateID)
				}).
				Scan(ctx)
			if err != nil {
				log.Error("failed to get formula template", zap.Error(err))
				return nil, dberror.HandleNotFoundError(err, "FormulaTemplate")
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req formulatemplateports.GetFormulaTemplatesByIDsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplate, error) {
			log := r.l.With(
				zap.String("operation", "GetByIDs"),
				zap.Any("request", req),
			)

			entities := make([]*formulatemplate.FormulaTemplate, 0, len(req.TemplateIDs))
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulatemplatecol.FormulaTemplateScopeTenant(sq, req.TenantInfo).
						Where(formulatemplatecol.FormulaTemplateColumns.ID.In(), bun.List(req.TemplateIDs))
				}).
				Scan(ctx)
			if err != nil {
				log.Error("failed to get formula templates", zap.Error(err))
				return nil, dberror.HandleNotFoundError(err, "FormulaTemplate")
			}

			return entities, nil
		},
	)
}

func (r *repository) FindByNames(
	ctx context.Context,
	req formulatemplateports.GetFormulaTemplatesByNamesRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplate, error) {
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
					return formulatemplatecol.FormulaTemplateScopeTenant(sq, req.TenantInfo).
						Where(formulatemplatecol.FormulaTemplateColumns.Name.In(), bun.List(req.Names))
				}).
				Scan(ctx)
			if err != nil {
				log.Error("failed to find formula templates by names", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *formulatemplateports.BulkUpdateFormulaTemplateStatusRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplate, error) {
			var entities []*formulatemplate.FormulaTemplate
			err := r.db.WithTx(
				ctx,
				kernelports.TxOptions{},
				func(txCtx context.Context, _ bun.Tx) error {
					updated, err := r.bulkUpdateStatus(txCtx, req)
					entities = updated
					return err
				},
			)
			if err != nil {
				return nil, err
			}
			return entities, nil
		},
	)
}

func (r *repository) bulkUpdateStatus(
	ctx context.Context,
	req *formulatemplateports.BulkUpdateFormulaTemplateStatusRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	cols := formulatemplatecol.FormulaTemplateColumns
	entities := make([]*formulatemplate.FormulaTemplate, 0, len(req.TemplateIDs))
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return formulatemplatecol.FormulaTemplateScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.TemplateIDs)).
				Where(cols.Status.In(), bun.List([]formulatemplate.Status{
					formulatemplate.StatusActive,
					formulatemplate.StatusInactive,
				}))
		}).
		Set(cols.Status.Set(), req.Status).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update formula template status", zap.Error(err))
		return nil, err
	}

	updatedIDs := make([]pulid.ID, 0, len(entities))
	for _, entity := range entities {
		updatedIDs = append(updatedIDs, entity.ID)
	}
	if skipped := sliceutils.Difference(
		sliceutils.Dedupe(req.TemplateIDs),
		updatedIDs,
	); len(skipped) > 0 {
		return nil, errortypes.NewValidationError(
			"templateIds",
			errortypes.ErrInvalid,
			"Templates {0} are no longer Active or Inactive, so none were updated; refresh and try again",
			strings.Join(pulid.Map(skipped, func(id pulid.ID) string { return id.String() }), ", "),
		)
	}

	return entities, nil
}

func (r *repository) BulkCreate(
	ctx context.Context,
	entities []*formulatemplate.FormulaTemplate,
) ([]*formulatemplate.FormulaTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplate, error) {
			if len(entities) == 0 {
				return entities, nil
			}

			_, err := r.db.DBForContext(ctx).
				NewInsert().
				Model(&entities).
				Returning("*").
				Exec(ctx)
			if err != nil {
				if dberror.IsUniqueConstraintViolation(err) {
					return nil, duplicateTemplateName(entities[0].Name)
				}
				r.l.Error("failed to bulk insert formula templates", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) TakenNames(
	ctx context.Context,
	req *formulatemplateports.ListTakenNamesRequest,
) (map[string]struct{}, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]struct{}, error) {
		taken := make(map[string]struct{}, len(req.Bases))
		if len(req.Bases) == 0 {
			return taken, nil
		}

		ft := formulatemplatecol.FormulaTemplateColumns
		var names []string
		query := r.db.DBForContext(ctx).
			NewSelect().
			Model((*formulatemplate.FormulaTemplate)(nil)).
			Column(ft.Name.String()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = formulatemplatecol.FormulaTemplateScopeTenant(sq, req.TenantInfo)
				return sq.WhereGroup(" AND ", func(orGroup *bun.SelectQuery) *bun.SelectQuery {
					for _, base := range req.Bases {
						orGroup = orGroup.WhereOr(ft.Name.Like(), base+"%")
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
	})
}

func (r *repository) CountUsages(
	ctx context.Context,
	req *formulatemplateports.GetTemplateUsageRequest,
) (*formulatemplateports.GetTemplateUsageResponse, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplateports.GetTemplateUsageResponse, error) {
			log := r.l.With(
				zap.String("operation", "CountUsages"),
				zap.String("templateID", req.TemplateID.String()),
			)

			type usageResult struct {
				Type  string `bun:"type"`
				Count int    `bun:"count"`
			}

			shipmentUsage := r.usageCount(ctx, req, "shipment",
				shipmentcol.ShipmentTable,
				shipmentcol.ShipmentColumns.FormulaTemplateID,
				shipmentcol.ShipmentScopeTenant,
			)
			ruleUsage := r.usageCount(ctx, req, "rate_agreement_rule",
				rateagreementcol.RateAgreementRuleTable,
				rateagreementcol.RateAgreementRuleColumns.FormulaTemplateID,
				rateagreementcol.RateAgreementRuleScopeTenant,
			)
			billingFallbackUsage := r.usageCount(ctx, req, "billing_control_fallback",
				tenantcol.BillingControlTable,
				tenantcol.BillingControlColumns.FallbackFormulaTemplateID,
				tenantcol.BillingControlScopeTenant,
			)

			var results []usageResult
			err := r.db.DBForContext(ctx).NewSelect().
				TableExpr("(?) AS shipment_usage", shipmentUsage).
				UnionAll(ruleUsage).
				UnionAll(billingFallbackUsage).
				Scan(ctx, &results)
			if err != nil {
				log.Error("failed to count usages", zap.Error(err))
				return nil, err
			}

			usages := make([]formulatemplateports.TemplateUsageCount, 0, len(results))
			inUse := false
			for _, res := range results {
				if res.Count > 0 {
					inUse = true
					usages = append(usages, formulatemplateports.TemplateUsageCount{
						Type:  res.Type,
						Count: res.Count,
					})
				}
			}

			return &formulatemplateports.GetTemplateUsageResponse{
				InUse:  inUse,
				Usages: usages,
			}, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *formulatemplateports.FormulaTemplateSelectOptionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
			cols := formulatemplatecol.FormulaTemplateColumns

			return dbhelper.SelectOptions[*formulatemplate.FormulaTemplate](
				ctx,
				r.db.DBForContext(ctx),
				req.SelectQueryRequest,
				&dbhelper.SelectOptionsConfig{
					Columns: []string{
						cols.ID.Name,
						cols.Name.Name,
						cols.Description.Name,
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
		},
	)
}

func duplicateTemplateName(name string) error {
	return errortypes.NewValidationError(
		"name",
		errortypes.ErrDuplicate,
		"A formula template named \"{0}\" already exists", name,
	)
}

func (r *repository) usageCount(
	ctx context.Context,
	req *formulatemplateports.GetTemplateUsageRequest,
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
