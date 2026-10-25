package permitrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/places/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/modules/places/domain/jurisdictionrule/jurisdictionruledb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason/holdreasondb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/permit"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/permit/permitdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	permitErrors      = dberror.Mapping{Entity: "Permit"}
	requirementErrors = dberror.Mapping{Entity: "PermitRequirement"}
	holdReasonErrors  = dberror.Mapping{Entity: "HoldReason"}
	ruleErrors        = dberror.Mapping{Entity: "JurisdictionRule"}
	overrideErrors    = dberror.Mapping{Entity: "JurisdictionOverride"}
)

type locationState struct {
	ID      pulid.ID
	StateID pulid.ID
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) permitports.PermitRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.permit-repository"),
	}
}

func (r *repository) ListByShipment(
	ctx context.Context,
	req *permitports.ListPermitsRequest,
) ([]*permit.Permit, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*permit.Permit, error) {
		p := permitdb.Permits
		entities, err := p.Where(
			p.ShipmentID.Eq(req.ShipmentID),
			keel.When(req.Status != "", p.Status.Eq(permit.Status(req.Status))),
		).
			Load(p.State()).
			Order(p.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list permits", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req *permitports.GetPermitByIDRequest,
) (*permit.Permit, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*permit.Permit, error) {
		p := permitdb.Permits
		entity, err := p.Where(p.ID.Eq(req.PermitID)).Load(p.State()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get permit", zap.Error(err))
			return nil, permitErrors.Map(err, req.PermitID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *permit.Permit,
) (*permit.Permit, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*permit.Permit, error) {
		if err := permitdb.Permits.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create permit", zap.Error(err))
			return nil, permitErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *permit.Permit,
) (*permit.Permit, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*permit.Permit, error) {
		if err := permitdb.Permits.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update permit", zap.Error(err))
			return nil, permitErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListRequirements(
	ctx context.Context,
	req *permitports.ListRequirementsRequest,
) ([]*permit.Requirement, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*permit.Requirement, error) {
		rq := permitdb.Requirements
		entities, err := rq.Where(
			rq.ShipmentID.Eq(req.ShipmentID),
			keel.When(req.Status != "", rq.Status.Eq(permit.RequirementStatus(req.Status))),
			keel.When(req.ExcludeSuperseded, rq.Status.Ne(permit.RequirementSuperseded)),
		).
			Load(rq.State()).
			Order(rq.RouteSequence.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list permit requirements", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

// ReplaceForShipment supersedes the prior derivation and inserts the new one in
// one transaction, so a shipment is never observed carrying a mix of two.
//
// Every live row is superseded, not only the open ones. Leaving waived and
// satisfied rows in place while inserting a fresh row for the same jurisdiction
// would accumulate a duplicate per save and resurrect a requirement an operator
// had already waived. The caller carries a waiver forward onto the replacement
// row instead.
func (r *repository) ReplaceForShipment(
	ctx context.Context,
	req *permitports.ReplaceRequirementsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := r.replace(ctx, req); err != nil {
			r.l.Error("failed to replace permit requirements", zap.Error(err))
			return dberror.MapRetryableTransactionError(
				requirementErrors.Map(err, ""),
				"Permit requirements are busy. Retry the request.",
			)
		}

		return nil
	})
}

func (r *repository) replace(
	ctx context.Context,
	req *permitports.ReplaceRequirementsRequest,
) error {
	db := r.db.Keel(ctx)
	rq := permitdb.Requirements
	_, err := rq.Where(
		rq.ShipmentID.Eq(req.ShipmentID),
		rq.Status.Ne(permit.RequirementSuperseded),
	).Update(ctx, db, rq.Status.To(permit.RequirementSuperseded))
	if err != nil {
		return err
	}

	if len(req.Requirements) == 0 {
		return nil
	}

	for _, requirement := range req.Requirements {
		requirement.ShipmentID = req.ShipmentID
		requirement.OrganizationID = req.TenantInfo.OrgID
		requirement.BusinessUnitID = req.TenantInfo.BuID
	}

	return rq.CreateMany(ctx, db, req.Requirements)
}

func (r *repository) ResolveRouteStates(
	ctx context.Context,
	req *permitports.ResolveRouteStatesRequest,
) (map[pulid.ID]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]pulid.ID, error) {
		if len(req.LocationIDs) == 0 {
			return map[pulid.ID]pulid.ID{}, nil
		}

		l := locationdb.Locations
		rows, err := keel.Into[locationState](ctx, r.db.Keel(ctx), l.
			Pick(l.ID, l.StateID).
			Where(l.ID.In(req.LocationIDs...)).
			Limit(len(req.LocationIDs)))
		if err != nil {
			r.l.Error("failed to resolve route states", zap.Error(err))
			return nil, err
		}

		states := make(map[pulid.ID]pulid.ID, len(rows))
		for _, row := range rows {
			if row.StateID.IsNotNil() {
				states[row.ID] = row.StateID
			}
		}

		return states, nil
	})
}

func (r *repository) WaiveRequirement(
	ctx context.Context,
	req *permitports.WaiveRequirementRepoRequest,
) (*permit.Requirement, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*permit.Requirement, error) {
		rq := permitdb.Requirements
		rows, err := rq.Where(rq.ID.Eq(req.RequirementID)).
			UpdateReturning(ctx, r.db.Keel(ctx), rq.Status.To(permit.RequirementWaived), rq.WaivedByID.To(req.WaivedByID), rq.WaiverReason.To(req.Reason))
		if err != nil {
			r.l.Error("failed to waive permit requirement", zap.Error(err))
			return nil, requirementErrors.Map(err, req.RequirementID.String())
		}
		if len(rows) == 0 {
			return nil, requirementErrors.Map(keel.ErrNotFound, req.RequirementID.String())
		}

		return rows[0], nil
	})
}

func (r *repository) FindHoldReasonByCode(
	ctx context.Context,
	_ pagination.TenantInfo,
	code string,
) (*holdreason.HoldReason, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*holdreason.HoldReason, error) {
		hr := holdreasondb.HoldReasons
		entity, err := hr.Where(hr.Code.Eq(code), hr.Active.IsTrue()).
			Order(hr.CreatedAt.Asc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, holdReasonErrors.Map(err, code)
		}

		return entity, nil
	})
}

type JurisdictionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type jurisdictionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewJurisdictionRuleRepository(
	p JurisdictionParams,
) permitports.JurisdictionRuleRepository {
	return &jurisdictionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.jurisdiction-rule-repository"),
	}
}

func (r *jurisdictionRepository) GetActiveByStateIDs(
	ctx context.Context,
	req *permitports.GetJurisdictionRulesRequest,
) ([]*jurisdictionrule.JurisdictionRule, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*jurisdictionrule.JurisdictionRule, error) {
			if len(req.StateIDs) == 0 {
				return nil, nil
			}

			jr := jurisdictionruledb.JurisdictionRules
			entities, err := jr.Where(
				jr.Status.Eq(jurisdictionrule.StatusActive),
				jr.StateID.In(req.StateIDs...),
			).
				Load(jr.State(), jr.EscortThresholds()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to load jurisdiction rules", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *jurisdictionRepository) GetOverrides(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*jurisdictionrule.Override, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*jurisdictionrule.Override, error) {
		o := jurisdictionruledb.Overrides
		entities, err := o.Query().Order(o.StateID.Asc()).All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to load jurisdiction overrides", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *jurisdictionRepository) List(
	ctx context.Context,
	req *permitports.ListJurisdictionRulesRequest,
) (*pagination.ListResult[*jurisdictionrule.JurisdictionRule], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*jurisdictionrule.JurisdictionRule], error) {
			jr := jurisdictionruledb.JurisdictionRules
			q := jr.Where(
				keel.When(
					req.VerificationState != "",
					jr.VerificationState.Eq(req.VerificationState),
				),
				keel.When(req.Status != "", jr.Status.Eq(req.Status)),
			)

			result, err := listquery.List(ctx, r.db.Keel(ctx), q,
				&listquery.Request[jurisdictionrule.JurisdictionRule]{
					Filter: req.Filter,
					Shape: func(
						q *keel.Query[*jurisdictionrule.JurisdictionRule],
					) *keel.Query[*jurisdictionrule.JurisdictionRule] {
						return q.Load(jr.State())
					},
					Then: []keel.Order{jr.StateID.Asc()},
				})
			if err != nil {
				r.l.Error("failed to list jurisdiction rules", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *jurisdictionRepository) GetByID(
	ctx context.Context,
	req *permitports.GetJurisdictionRuleByIDRequest,
) (*jurisdictionrule.JurisdictionRule, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*jurisdictionrule.JurisdictionRule, error) {
			jr := jurisdictionruledb.JurisdictionRules
			entity, err := jr.Where(jr.ID.Eq(req.RuleID)).
				Load(jr.State()).
				Load(listquery.When(req.ExpandThresholds, jr.EscortThresholds())...).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get jurisdiction rule", zap.Error(err))
				return nil, ruleErrors.Map(err, req.RuleID.String())
			}

			return entity, nil
		},
	)
}

func (r *jurisdictionRepository) Create(
	ctx context.Context,
	entity *jurisdictionrule.JurisdictionRule,
) (*jurisdictionrule.JurisdictionRule, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*jurisdictionrule.JurisdictionRule, error) {
			err := jurisdictionruledb.JurisdictionRules.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create jurisdiction rule", zap.Error(err))
				return nil, ruleErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *jurisdictionRepository) Update(
	ctx context.Context,
	entity *jurisdictionrule.JurisdictionRule,
) (*jurisdictionrule.JurisdictionRule, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*jurisdictionrule.JurisdictionRule, error) {
			err := jurisdictionruledb.JurisdictionRules.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update jurisdiction rule", zap.Error(err))
				return nil, ruleErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

// Verify writes only the verification columns.
//
// Deliberately not a full Update: confirming a row against the statute is an
// assertion about the numbers already there, so it must not be able to change
// them in the same operation. An operator who wants different limits edits the
// row, which resets it to Unverified.
func (r *jurisdictionRepository) Verify(
	ctx context.Context,
	req *permitports.VerifyJurisdictionRuleRequest,
) (*jurisdictionrule.JurisdictionRule, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*jurisdictionrule.JurisdictionRule, error) {
			jr := jurisdictionruledb.JurisdictionRules
			sets := []keel.Assignment{
				jr.VerificationState.To(req.State),
				jr.VerifiedAt.To(req.VerifiedAt),
			}
			if req.SourceNote != "" {
				sets = append(sets, jr.SourceNote.To(req.SourceNote))
			}
			if req.SourceURL != "" {
				sets = append(sets, jr.SourceURL.To(req.SourceURL))
			}

			n, err := jr.Where(jr.ID.Eq(req.RuleID)).Update(ctx, r.db.Keel(ctx), sets...)
			if err != nil {
				r.l.Error("failed to verify jurisdiction rule", zap.Error(err))
				return nil, err
			}
			if err = dberror.CheckAffected(n, "JurisdictionRule", req.RuleID.String()); err != nil {
				return nil, err
			}

			return r.GetByID(ctx, &permitports.GetJurisdictionRuleByIDRequest{RuleID: req.RuleID})
		},
	)
}

// ListConnection backs the admin table.
//
// No tenant scoping and no tenant filter in the cursor: this table is global
// reference data, so every organization's admin table shows the same rows.
func (r *jurisdictionRepository) ListConnection(
	ctx context.Context,
	req *permitports.ListJurisdictionRuleConnectionRequest,
) (*pagination.CursorListResult[*jurisdictionrule.JurisdictionRule], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*jurisdictionrule.JurisdictionRule], error) {
			jr := jurisdictionruledb.JurisdictionRules
			result, err := listquery.Connection(ctx, r.db.Keel(ctx), jr.Query(),
				&listquery.Request[jurisdictionrule.JurisdictionRule]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*jurisdictionrule.JurisdictionRule],
					) *keel.Query[*jurisdictionrule.JurisdictionRule] {
						return q.Load(jr.State())
					},
				})
			if err != nil {
				r.l.Error("failed to list jurisdiction rules", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *jurisdictionRepository) ListOverridesConnection(
	ctx context.Context,
	req *permitports.ListOverridesRequest,
	_ pagination.TenantInfo,
) (*pagination.CursorListResult[*jurisdictionrule.Override], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*jurisdictionrule.Override], error) {
			o := jurisdictionruledb.Overrides
			result, err := listquery.Connection(ctx, r.db.Keel(ctx), o.Query(),
				&listquery.Request[jurisdictionrule.Override]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*jurisdictionrule.Override],
					) *keel.Query[*jurisdictionrule.Override] {
						return q.Load(o.State())
					},
				})
			if err != nil {
				r.l.Error("failed to list jurisdiction overrides", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *jurisdictionRepository) GetOverrideByID(
	ctx context.Context,
	req *permitports.GetOverrideByIDRequest,
) (*jurisdictionrule.Override, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*jurisdictionrule.Override, error) {
		o := jurisdictionruledb.Overrides
		entity, err := o.Where(o.ID.Eq(req.OverrideID)).Load(o.State()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get jurisdiction override", zap.Error(err))
			return nil, overrideErrors.Map(err, req.OverrideID.String())
		}

		return entity, nil
	})
}

func (r *jurisdictionRepository) CreateOverride(
	ctx context.Context,
	entity *jurisdictionrule.Override,
) (*jurisdictionrule.Override, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*jurisdictionrule.Override, error) {
		if err := jurisdictionruledb.Overrides.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create jurisdiction override", zap.Error(err))
			return nil, overrideErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *jurisdictionRepository) UpdateOverride(
	ctx context.Context,
	entity *jurisdictionrule.Override,
) (*jurisdictionrule.Override, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*jurisdictionrule.Override, error) {
		if err := jurisdictionruledb.Overrides.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update jurisdiction override", zap.Error(err))
			return nil, overrideErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *jurisdictionRepository) DeleteOverride(
	ctx context.Context,
	req *permitports.DeleteOverrideRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := jurisdictionruledb.Overrides.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&jurisdictionrule.Override{ID: req.OverrideID},
		)
		if err != nil {
			r.l.Error("failed to delete jurisdiction override", zap.Error(err))
			return overrideErrors.Map(err, req.OverrideID.String())
		}

		return nil
	})
}
