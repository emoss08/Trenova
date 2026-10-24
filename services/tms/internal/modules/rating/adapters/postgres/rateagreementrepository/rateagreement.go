package rateagreementrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) rateagreementports.RateAgreementRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.rateagreement-repository"),
	}
}

func listFilters(req *rateagreementports.ListRateAgreementRequest) []keel.Pred {
	ra := rateagreementdb.RateAgreements
	preds := []keel.Pred{
		keel.When(req.PartyType != "", ra.PartyType.Eq(req.PartyType)),
		keel.When(req.Status != "", ra.Status.Eq(req.Status)),
	}
	if req.CustomerID != nil {
		preds = append(preds, ra.CustomerID.Eq(*req.CustomerID))
	}
	if req.CarrierID != nil {
		preds = append(preds, ra.CarrierID.Eq(*req.CarrierID))
	}

	return preds
}

func (r *repository) List(
	ctx context.Context,
	req *rateagreementports.ListRateAgreementRequest,
) (*pagination.ListResult[*rateagreement.RateAgreement], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*rateagreement.RateAgreement], error) {
			ra := rateagreementdb.RateAgreements
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				ra.Where(listFilters(req)...),
				&listquery.Request[rateagreement.RateAgreement]{
					Filter: req.Filter,
					Then:   []keel.Order{ra.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list rate agreements", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *rateagreementports.ListRateAgreementConnectionRequest,
) (*pagination.CursorListResult[*rateagreement.RateAgreement], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*rateagreement.RateAgreement], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				rateagreementdb.RateAgreements.Query(),
				&listquery.Request[rateagreement.RateAgreement]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.RateAgreementColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list rate agreements", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func agreementQuery(
	req *rateagreementports.GetRateAgreementByIDRequest,
	now int64,
) *keel.Query[rateagreement.RateAgreement] {
	ra := rateagreementdb.RateAgreements
	q := ra.Where(ra.ID.Eq(req.RateAgreementID))

	if req.IncludeChildren && !req.OmitRules {
		rules := ra.Rules()
		window := ruleWindowColumns{from: rules.EffectiveFrom, to: rules.EffectiveTo}
		scope := openAt(rules.EffectiveTo, now)
		if req.AsOf > 0 {
			scope = effectiveAt(window, req.AsOf)
		}
		q = q.Load(rules.Where(scope).Order(rules.SpecificityScore.Desc()))
	}

	if req.IncludeChildren {
		q = q.Load(ra.Accessorials(), ra.FuelBinding())
	}

	return q
}

func (r *repository) GetByID(
	ctx context.Context,
	req *rateagreementports.GetRateAgreementByIDRequest,
) (*rateagreement.RateAgreement, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*rateagreement.RateAgreement, error) {
		entity, err := agreementQuery(req, timeutils.NowUnix()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get rate agreement",
				zap.String("id", req.RateAgreementID.String()),
				zap.Error(err),
			)
			return nil, agreementErrors.Map(err, req.RateAgreementID.String())
		}

		return entity, nil
	})
}

func rulesQuery(
	req *rateagreementports.ListRateAgreementRulesRequest,
) *keel.Query[rateagreement.RateAgreementRule] {
	rr := rateagreementdb.RateAgreementRules
	q := rr.Where(
		rr.RateAgreementID.Eq(req.RateAgreementID),
		keel.When(!req.IncludeInactive, rr.Status.Eq(rateagreement.RuleStatusActive)),
		keel.When(req.LaneKey != "", rr.LaneKey.Eq(req.LaneKey)),
	)

	if req.IncludeSuperseded {
		return q.Order(rr.EffectiveFrom.Desc())
	}

	return q.Where(ruleWindowAt(ruleWindow(), req.AsOf)).
		Order(rr.SpecificityScore.Desc(), rr.ID.Asc())
}

func (r *repository) ListRules(
	ctx context.Context,
	req *rateagreementports.ListRateAgreementRulesRequest,
) ([]*rateagreement.RateAgreementRule, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*rateagreement.RateAgreementRule, error) {
			entities, err := rulesQuery(req).All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list rate agreement rules", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) GetRuleByID(
	ctx context.Context,
	req *rateagreementports.GetRateAgreementRuleByIDRequest,
) (*rateagreement.RateAgreementRule, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*rateagreement.RateAgreementRule, error) {
			rr := rateagreementdb.RateAgreementRules
			entity, err := rr.Where(rr.ID.Eq(req.RuleID)).
				Load(rr.Agreement()).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get rate agreement rule",
					zap.String("id", req.RuleID.String()),
					zap.Error(err),
				)
				return nil, ruleErrors.Map(err, req.RuleID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
) (*rateagreement.RateAgreement, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*rateagreement.RateAgreement, error) {
		err := rateagreementdb.RateAgreements.Create(ctx, r.db.Keel(ctx), entity)
		if err == nil {
			err = r.writeChildren(ctx, entity, false)
		}
		if err == nil && len(entity.Rules) > 0 {
			err = r.insertRules(ctx, &rateagreementports.AmendRateAgreementRulesRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
				RateAgreementID: entity.ID,
				EffectiveFrom:   entity.EffectiveFrom,
				Rules:           entity.Rules,
			})
		}
		if err != nil {
			r.l.Error("failed to create rate agreement",
				zap.String("code", entity.Code),
				zap.Error(err),
			)
			return nil, mapAgreementError(err, entity.ID.String())
		}

		return entity, nil
	})
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
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*rateagreement.RateAgreement, error) {
		err := rateagreementdb.RateAgreements.Save(ctx, r.db.Keel(ctx), entity)
		if err == nil {
			err = r.writeChildren(ctx, entity, true)
		}
		if err != nil {
			r.l.Error("failed to update rate agreement",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, mapAgreementError(err, entity.ID.String())
		}

		return entity, nil
	})
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
	db := r.db.Keel(ctx)
	if replace {
		if err := deleteChildren(ctx, db, entity.ID); err != nil {
			return err
		}
	}

	stampAccessorials(entity, replace)

	accessorials := make([]*rateagreement.RateAgreementAccessorial, 0, len(entity.Accessorials))
	for _, accessorial := range entity.Accessorials {
		if accessorial != nil {
			accessorials = append(accessorials, accessorial)
		}
	}
	if len(accessorials) > 0 {
		if err := rateagreementdb.RateAgreementAccessorials.CreateMany(ctx, db, accessorials); err != nil {
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

	return rateagreementdb.RateAgreementFuelBindings.Create(ctx, db, entity.FuelBinding)
}

func deleteChildren(ctx context.Context, db *keel.DB, agreementID pulid.ID) error {
	ac := rateagreementdb.RateAgreementAccessorials
	if _, err := ac.Where(ac.RateAgreementID.Eq(agreementID)).Delete(ctx, db); err != nil {
		return err
	}

	fb := rateagreementdb.RateAgreementFuelBindings
	_, err := fb.Where(fb.RateAgreementID.Eq(agreementID)).Delete(ctx, db)

	return err
}

func selectOptionsQuery() *keel.Query[rateagreement.RateAgreement] {
	ra := rateagreementdb.RateAgreements

	return ra.Pick(ra.ID, ra.Code, ra.Name, ra.PartyType, ra.Status, ra.Currency).
		Where(ra.Status.Eq(rateagreement.StatusActive))
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*rateagreement.RateAgreement], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*rateagreement.RateAgreement], error) {
			ra := rateagreementdb.RateAgreements

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				selectOptionsQuery(),
				req,
				func(s string) keel.Pred {
					return keel.Or(ra.Code.ContainsFold(s), ra.Name.ContainsFold(s))
				},
			)
		},
	)
}
