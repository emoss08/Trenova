package rateagreementrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.RateAgreementRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.rateagreement-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListRateAgreementRequest,
) *bun.SelectQuery {
	cols := buncolgen.RateAgreementColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.RateAgreementTable.Alias,
		req.Filter,
		(*rateagreement.RateAgreement)(nil),
	)

	q = q.Apply(buncolgen.RateAgreementApplyTenant(req.Filter.TenantInfo))

	if req.PartyType != "" {
		q = q.Where(cols.PartyType.Eq(), req.PartyType)
	}

	if req.Status != "" {
		q = q.Where(cols.Status.Eq(), req.Status)
	}

	if req.CustomerID != nil {
		q = q.Where(cols.CustomerID.Eq(), *req.CustomerID)
	}

	if req.CarrierID != nil {
		q = q.Where(cols.CarrierID.Eq(), *req.CarrierID)
	}

	return q.
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListRateAgreementRequest,
) (*pagination.ListResult[*rateagreement.RateAgreement], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*rateagreement.RateAgreement, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count rate agreements", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*rateagreement.RateAgreement]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListRateAgreementConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.RateAgreementTable.Alias,
		req.Filter,
		req.Cursor,
		(*rateagreement.RateAgreement)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListRateAgreementConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.RateAgreementTable.Alias,
		req.Filter,
		(*rateagreement.RateAgreement)(nil),
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListRateAgreementConnectionRequest,
) (*pagination.CursorListResult[*rateagreement.RateAgreement], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*rateagreement.RateAgreement)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count rate agreements", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*rateagreement.RateAgreement]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*rateagreement.RateAgreement) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyRateAgreementColumns(sq, req.RateAgreementColumns)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		},
	)
	if err != nil {
		log.Error("failed to scan rate agreements", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetRateAgreementByIDRequest,
) (*rateagreement.RateAgreement, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.RateAgreementID.String()),
	)

	entity := new(rateagreement.RateAgreement)
	cols := buncolgen.RateAgreementColumns

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateAgreementScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RateAgreementID)
		})

	if req.IncludeChildren {
		q = q.
			Relation(
				buncolgen.Rel(buncolgen.RateAgreementRelations.Rules),
				func(sq *bun.SelectQuery) *bun.SelectQuery {
					if req.AsOf > 0 {
						sq = applyRuleWindow(sq, req.AsOf)
					} else {
						// With no moment given this is an editor load, which
						// wants every lane that still has a future — current
						// and forthcoming — and none of the closed-out history
						// an amendment leaves behind.
						sq = applyOpenRuleWindow(sq, timeutils.NowUnix())
					}
					return sq.Order(
						buncolgen.RateAgreementRuleColumns.SpecificityScore.OrderDesc(),
					)
				},
			).
			Relation(buncolgen.Rel(
				buncolgen.RateAgreementRelations.Rules,
				buncolgen.RateAgreementRuleRelations.Breaks,
			)).
			Relation(buncolgen.Rel(buncolgen.RateAgreementRelations.Accessorials)).
			Relation(buncolgen.Rel(buncolgen.RateAgreementRelations.FuelBinding))
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get rate agreement", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "RateAgreement")
	}

	return entity, nil
}

func (r *repository) ListRules(
	ctx context.Context,
	req *repositories.ListRateAgreementRulesRequest,
) ([]*rateagreement.RateAgreementRule, error) {
	log := r.l.With(zap.String("operation", "ListRules"))

	cols := buncolgen.RateAgreementRuleColumns
	entities := make([]*rateagreement.RateAgreementRule, 0)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.Rel(buncolgen.RateAgreementRuleRelations.Breaks)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.RateAgreementRuleScopeTenant(sq, req.TenantInfo).
				Where(cols.RateAgreementID.Eq(), req.RateAgreementID)

			if !req.IncludeInactive {
				sq = sq.Where(cols.Status.Eq(), rateagreement.RuleStatusActive)
			}

			if req.LaneKey != "" {
				sq = sq.Where(cols.LaneKey.Eq(), req.LaneKey)
			}

			return sq
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			// History keeps the closed-out rules — the lineage is the point —
			// so the window filter only applies to a point-in-time read.
			if req.IncludeSuperseded {
				return sq
			}

			return applyRuleWindow(sq, req.AsOf)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			if req.IncludeSuperseded {
				return sq.Order(cols.EffectiveFrom.OrderDesc())
			}

			return sq.
				Order(cols.SpecificityScore.OrderDesc()).
				Order(cols.ID.OrderAsc())
		})

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to list rate agreement rules", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetRuleByID(
	ctx context.Context,
	req *repositories.GetRateAgreementRuleByIDRequest,
) (*rateagreement.RateAgreementRule, error) {
	log := r.l.With(
		zap.String("operation", "GetRuleByID"),
		zap.String("id", req.RuleID.String()),
	)

	entity := new(rateagreement.RateAgreementRule)
	cols := buncolgen.RateAgreementRuleColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.Rel(buncolgen.RateAgreementRuleRelations.Breaks)).
		Relation(buncolgen.Rel(buncolgen.RateAgreementRuleRelations.Agreement)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateAgreementRuleScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RuleID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get rate agreement rule", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "RateAgreementRule")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
) (*rateagreement.RateAgreement, error) {
	log := r.l.With(zap.String("operation", "Create"), zap.String("code", entity.Code))

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}

		if cErr := r.writeChildren(c, entity, false); cErr != nil {
			return cErr
		}

		if len(entity.Rules) == 0 {
			return nil
		}

		return r.insertRules(c, &repositories.AmendRateAgreementRulesRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			RateAgreementID: entity.ID,
			EffectiveFrom:   entity.EffectiveFrom,
			Rules:           entity.Rules,
		})
	})
	if err != nil {
		log.Error("failed to create rate agreement", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Rate agreement is busy. Retry the request.",
		)
	}

	return entity, nil
}

// Update rewrites the header and the sets that hang directly off it.
//
// Rules are deliberately not touched here. They carry their own effective
// windows and are amended through AmendRules, so editing a contract's name or
// its currency can never silently rewrite the history of what it charged.
func (r *repository) Update(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
) (*rateagreement.RateAgreement, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++
	cols := buncolgen.RateAgreementColumns

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		results, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(cols.Version.Eq(), ov).
			OmitZero().
			Returning("*").
			Exec(c)
		if uErr != nil {
			return uErr
		}

		if uErr = dberror.CheckRowsAffected(
			results,
			"RateAgreement",
			entity.ID.String(),
		); uErr != nil {
			return uErr
		}

		return r.writeChildren(c, entity, true)
	})
	if err != nil {
		log.Error("failed to update rate agreement", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Rate agreement is busy. Retry the request.",
		)
	}

	return entity, nil
}

// writeChildren replaces the accessorial schedule and the fuel binding.
//
// Both are small, unreferenced sets whose rows carry no identity a user would
// recognise, so rebuilding them is simpler than diffing and cannot leave a
// stale row behind still pricing something the contract dropped.
func (r *repository) writeChildren(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
	replace bool,
) error {
	if replace {
		if err := r.deleteChildren(ctx, entity); err != nil {
			return err
		}
	}

	stampAccessorials(entity, replace)

	if len(entity.Accessorials) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.Accessorials).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}
	}

	if entity.FuelBinding == nil {
		return nil
	}

	if replace {
		entity.FuelBinding.ID = pulid.Nil
	}
	entity.FuelBinding.RateAgreementID = entity.ID
	entity.FuelBinding.OrganizationID = entity.OrganizationID
	entity.FuelBinding.BusinessUnitID = entity.BusinessUnitID

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity.FuelBinding).
		Returning("*").
		Exec(ctx)

	return err
}

func (r *repository) deleteChildren(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
) error {
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	accessorialCols := buncolgen.RateAgreementAccessorialColumns
	if _, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*rateagreement.RateAgreementAccessorial)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.RateAgreementAccessorialScopeTenantDelete(dq, tenantInfo).
				Where(accessorialCols.RateAgreementID.Eq(), entity.ID)
		}).
		Exec(ctx); err != nil {
		return err
	}

	bindingCols := buncolgen.RateAgreementFuelBindingColumns
	_, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*rateagreement.RateAgreementFuelBinding)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.RateAgreementFuelBindingScopeTenantDelete(dq, tenantInfo).
				Where(bindingCols.RateAgreementID.Eq(), entity.ID)
		}).
		Exec(ctx)

	return err
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*rateagreement.RateAgreement], error) {
	cols := buncolgen.RateAgreementColumns

	return dbhelper.SelectOptions[*rateagreement.RateAgreement](
		ctx,
		r.db.DB(),
		req,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Code,
				cols.Name,
				cols.PartyType,
				cols.Status,
				cols.Currency,
			},
			OrgColumnRef:     &cols.OrganizationID,
			BuColumnRef:      &cols.BusinessUnitID,
			SearchColumnRefs: []buncolgen.Column{cols.Code, cols.Name},
			EntityName:       "RateAgreement",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.Status.Eq(), rateagreement.StatusActive)
			},
		},
	)
}
