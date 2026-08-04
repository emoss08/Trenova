package permitrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.PermitRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.permit-repository"),
	}
}

func (r *repository) ListByShipment(
	ctx context.Context,
	req *repositories.ListPermitsRequest,
) ([]*permit.Permit, error) {
	cols := buncolgen.PermitColumns
	entities := make([]*permit.Permit, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.PermitRelations.State).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PermitScopeTenant(sq, req.TenantInfo).
				Where(cols.ShipmentID.Eq(), req.ShipmentID)
		})

	if req.Status != "" {
		q = q.Where(cols.Status.Eq(), req.Status)
	}

	if err := q.Order(cols.CreatedAt.OrderAsc()).Scan(ctx); err != nil {
		r.l.Error("failed to list permits", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetPermitByIDRequest,
) (*permit.Permit, error) {
	entity := new(permit.Permit)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.PermitRelations.State).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PermitScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.PermitColumns.ID.Eq(), req.PermitID)
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get permit", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *permit.Permit,
) (*permit.Permit, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create permit", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *permit.Permit,
) (*permit.Permit, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update permit", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Permit", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListRequirements(
	ctx context.Context,
	req *repositories.ListRequirementsRequest,
) ([]*permit.Requirement, error) {
	cols := buncolgen.RequirementColumns
	entities := make([]*permit.Requirement, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.RequirementRelations.State).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RequirementScopeTenant(sq, req.TenantInfo).
				Where(cols.ShipmentID.Eq(), req.ShipmentID)
		})

	if req.Status != "" {
		q = q.Where(cols.Status.Eq(), req.Status)
	}

	if err := q.Order(cols.RouteSequence.OrderAsc()).Scan(ctx); err != nil {
		r.l.Error("failed to list permit requirements", zap.Error(err))
		return nil, err
	}

	return entities, nil
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
	req *repositories.ReplaceRequirementsRequest,
) error {
	cols := buncolgen.RequirementColumns

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, dErr := r.db.DBForContext(c).
			NewUpdate().
			Model((*permit.Requirement)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.RequirementScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ShipmentID.Eq(), req.ShipmentID).
					Where(cols.Status.NotEq(), permit.RequirementSuperseded)
			}).
			Set(cols.Status.Set(), permit.RequirementSuperseded).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Exec(c); dErr != nil {
			return dErr
		}

		if len(req.Requirements) == 0 {
			return nil
		}

		for _, requirement := range req.Requirements {
			requirement.ShipmentID = req.ShipmentID
			requirement.OrganizationID = req.TenantInfo.OrgID
			requirement.BusinessUnitID = req.TenantInfo.BuID
		}

		_, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(&req.Requirements).
			Returning("*").
			Exec(c)

		return iErr
	})
	if err != nil {
		r.l.Error("failed to replace permit requirements", zap.Error(err))
		return dberror.MapRetryableTransactionError(
			err,
			"Permit requirements are busy. Retry the request.",
		)
	}

	return nil
}

func (r *repository) ResolveRouteStates(
	ctx context.Context,
	req *repositories.ResolveRouteStatesRequest,
) (map[pulid.ID]pulid.ID, error) {
	if len(req.LocationIDs) == 0 {
		return map[pulid.ID]pulid.ID{}, nil
	}

	cols := buncolgen.LocationColumns
	rows := make([]*location.Location, 0, len(req.LocationIDs))

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&rows).
		Column(cols.ID.Bare(), cols.StateID.Bare()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.LocationScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.In(), bun.In(req.LocationIDs))
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to resolve route states", zap.Error(err))
		return nil, err
	}

	states := make(map[pulid.ID]pulid.ID, len(rows))
	for _, row := range rows {
		if row != nil && row.StateID.IsNotNil() {
			states[row.ID] = row.StateID
		}
	}

	return states, nil
}

func (r *repository) WaiveRequirement(
	ctx context.Context,
	req *repositories.WaiveRequirementRepoRequest,
) (*permit.Requirement, error) {
	cols := buncolgen.RequirementColumns
	now := timeutils.NowUnix()
	waivedBy := req.WaivedByID

	entity := new(permit.Requirement)

	err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.RequirementScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RequirementID)
		}).
		Set(cols.Status.Set(), permit.RequirementWaived).
		Set(cols.WaivedByID.Set(), waivedBy).
		Set(cols.WaiverReason.Set(), req.Reason).
		Set(cols.UpdatedAt.Set(), now).
		Returning("*").
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to waive permit requirement", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) FindHoldReasonByCode(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	code string,
) (*holdreason.HoldReason, error) {
	entity := new(holdreason.HoldReason)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.HoldReasonScopeTenant(sq, tenantInfo).
				Where(buncolgen.HoldReasonColumns.Code.Eq(), code).
				Where(buncolgen.HoldReasonColumns.Active.IsTrue())
		}).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entity, nil
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
) repositories.JurisdictionRuleRepository {
	return &jurisdictionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.jurisdiction-rule-repository"),
	}
}

func (r *jurisdictionRepository) GetActiveByStateIDs(
	ctx context.Context,
	req *repositories.GetJurisdictionRulesRequest,
) ([]*jurisdictionrule.JurisdictionRule, error) {
	if len(req.StateIDs) == 0 {
		return nil, nil
	}

	cols := buncolgen.JurisdictionRuleColumns
	entities := make([]*jurisdictionrule.JurisdictionRule, 0, len(req.StateIDs))

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.JurisdictionRuleRelations.State).
		Relation(buncolgen.JurisdictionRuleRelations.EscortThresholds).
		Where(cols.Status.Eq(), jurisdictionrule.StatusActive).
		Where(cols.StateID.In(), bun.In(req.StateIDs)).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to load jurisdiction rules", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *jurisdictionRepository) GetOverrides(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*jurisdictionrule.Override, error) {
	entities := make([]*jurisdictionrule.Override, 0, 8)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.OverrideScopeTenant(sq, tenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to load jurisdiction overrides", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *jurisdictionRepository) List(
	ctx context.Context,
	req *repositories.ListJurisdictionRulesRequest,
) (*pagination.ListResult[*jurisdictionrule.JurisdictionRule], error) {
	cols := buncolgen.JurisdictionRuleColumns
	entities := make([]*jurisdictionrule.JurisdictionRule, 0, req.Filter.Pagination.SafeLimit())

	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Relation(buncolgen.JurisdictionRuleRelations.State).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			if req.VerificationState != "" {
				sq = sq.Where(cols.VerificationState.Eq(), req.VerificationState)
			}
			if req.Status != "" {
				sq = sq.Where(cols.Status.Eq(), req.Status)
			}
			return sq.
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset())
		}).
		ScanAndCount(ctx)
	if err != nil {
		r.l.Error("failed to list jurisdiction rules", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*jurisdictionrule.JurisdictionRule]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *jurisdictionRepository) GetByID(
	ctx context.Context,
	req *repositories.GetJurisdictionRuleByIDRequest,
) (*jurisdictionrule.JurisdictionRule, error) {
	entity := new(jurisdictionrule.JurisdictionRule)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.JurisdictionRuleRelations.State).
		Where(buncolgen.JurisdictionRuleColumns.ID.Eq(), req.RuleID)

	if req.ExpandThresholds {
		q = q.Relation(buncolgen.JurisdictionRuleRelations.EscortThresholds)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to get jurisdiction rule", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *jurisdictionRepository) Create(
	ctx context.Context,
	entity *jurisdictionrule.JurisdictionRule,
) (*jurisdictionrule.JurisdictionRule, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create jurisdiction rule", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *jurisdictionRepository) Update(
	ctx context.Context,
	entity *jurisdictionrule.JurisdictionRule,
) (*jurisdictionrule.JurisdictionRule, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update jurisdiction rule", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(
		results, "JurisdictionRule", entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

// Verify writes only the verification columns.
//
// Deliberately not a full Update: confirming a row against the statute is an
// assertion about the numbers already there, so it must not be able to change
// them in the same operation. An operator who wants different limits edits the
// row, which resets it to Unverified.
func (r *jurisdictionRepository) Verify(
	ctx context.Context,
	req *repositories.VerifyJurisdictionRuleRequest,
) (*jurisdictionrule.JurisdictionRule, error) {
	cols := buncolgen.JurisdictionRuleColumns

	q := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*jurisdictionrule.JurisdictionRule)(nil)).
		Set("verification_state = ?", req.State).
		Set("verified_at = ?", req.VerifiedAt).
		Set("version = version + 1").
		Set("updated_at = ?", timeutils.NowUnix()).
		Where(cols.ID.Eq(), req.RuleID)

	if req.SourceNote != "" {
		q = q.Set("source_note = ?", req.SourceNote)
	}
	if req.SourceURL != "" {
		q = q.Set("source_url = ?", req.SourceURL)
	}

	results, err := q.Exec(ctx)
	if err != nil {
		r.l.Error("failed to verify jurisdiction rule", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(
		results, "JurisdictionRule", req.RuleID.String(),
	); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetJurisdictionRuleByIDRequest{RuleID: req.RuleID})
}

// ListConnection backs the admin table.
//
// No tenant scoping and no tenant filter in the cursor: this table is global
// reference data, so every organization's admin table shows the same rows.
func (r *jurisdictionRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListJurisdictionRuleConnectionRequest,
) (*pagination.CursorListResult[*jurisdictionrule.JurisdictionRule], error) {
	dba := r.db.DBForContext(ctx)

	total, err := dba.
		NewSelect().
		Model((*jurisdictionrule.JurisdictionRule)(nil)).
		Count(ctx)
	if err != nil {
		r.l.Error("failed to count jurisdiction rules", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*jurisdictionrule.JurisdictionRule]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*jurisdictionrule.JurisdictionRule) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Relation(buncolgen.JurisdictionRuleRelations.State)
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return sq, nil
			},
		})
	if err != nil {
		r.l.Error("failed to scan jurisdiction rules", zap.Error(err))
		return nil, err
	}

	return result, nil
}
