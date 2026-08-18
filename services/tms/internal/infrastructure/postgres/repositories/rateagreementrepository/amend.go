package rateagreementrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// AmendRules closes out the rules a change replaces and inserts their
// successors, in one transaction.
//
// This is the only way rules ever change, and it is deliberately the same
// operation whether the change came from a person editing a lane, a general
// rate increase, a rate sheet import, or a simulation being made real. Nothing
// is updated in place and nothing is deleted: the superseded rows keep their
// history and simply stop being effective at the moment their replacements
// begin.
//
// The two halves have to happen together. Closing without inserting would leave
// a lane unpriced; inserting without closing would leave two rules effective at
// once, and which one won would depend on the tiebreak rather than on what
// anybody intended.
func (r *repository) AmendRules(
	ctx context.Context,
	req *repositories.AmendRateAgreementRulesRequest,
) error {
	log := r.l.With(
		zap.String("operation", "AmendRules"),
		zap.String("agreementId", req.RateAgreementID.String()),
		zap.Int("superseded", len(req.SupersededIDs)),
		zap.Int("inserted", len(req.Rules)),
	)

	if len(req.SupersededIDs) == 0 && len(req.Rules) == 0 {
		return nil
	}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if cErr := r.closeOutRules(c, req); cErr != nil {
			return cErr
		}

		return r.insertRules(c, req)
	})
	if err != nil {
		log.Error("failed to amend rate agreement rules", zap.Error(err))
		return dberror.MapRetryableTransactionError(
			err,
			"Rate agreement is busy. Retry the request.",
		)
	}

	return nil
}

// closeOutRules ends the superseded rules at the instant their replacements
// begin. Windows are half open, so the two never overlap and never leave a gap.
//
// A rule that already ends earlier is left alone: an amendment should not
// extend the life of something already retired.
func (r *repository) closeOutRules(
	ctx context.Context,
	req *repositories.AmendRateAgreementRulesRequest,
) error {
	if len(req.SupersededIDs) == 0 {
		return nil
	}

	cols := buncolgen.RateAgreementRuleColumns

	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*rateagreement.RateAgreementRule)(nil)).
		Set(cols.EffectiveTo.Set(), req.EffectiveFrom).
		Set(cols.UpdatedAt.SetExpr(r.db.NowEpoch())).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.RateAgreementRuleScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.RateAgreementID.Eq(), req.RateAgreementID).
				Where(cols.ID.In(), bun.List(req.SupersededIDs)).
				WhereGroup(" AND ", func(wq *bun.UpdateQuery) *bun.UpdateQuery {
					return wq.
						Where(cols.EffectiveTo.IsNull()).
						WhereOr(cols.EffectiveTo.Gt(), req.EffectiveFrom)
				})
		}).
		Exec(ctx)

	return err
}

func (r *repository) insertRules(
	ctx context.Context,
	req *repositories.AmendRateAgreementRulesRequest,
) error {
	if len(req.Rules) == 0 {
		return nil
	}

	for _, rule := range req.Rules {
		if rule == nil {
			continue
		}

		rule.ID = pulid.Nil
		rule.RateAgreementID = req.RateAgreementID
		rule.OrganizationID = req.TenantInfo.OrgID
		rule.BusinessUnitID = req.TenantInfo.BuID
		if rule.EffectiveFrom == 0 {
			rule.EffectiveFrom = req.EffectiveFrom
		}
	}

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&req.Rules).
		Returning("*").
		Exec(ctx); err != nil {
		return err
	}

	return r.insertRuleBreaks(ctx, req.Rules)
}

func (r *repository) insertRuleBreaks(
	ctx context.Context,
	rules []*rateagreement.RateAgreementRule,
) error {
	breaks := make([]*rateagreement.RateAgreementRuleBreak, 0)

	for _, rule := range rules {
		if rule == nil {
			continue
		}

		for _, ruleBreak := range rule.Breaks {
			if ruleBreak == nil {
				continue
			}

			ruleBreak.ID = pulid.Nil
			ruleBreak.RateAgreementRuleID = rule.ID
			ruleBreak.OrganizationID = rule.OrganizationID
			ruleBreak.BusinessUnitID = rule.BusinessUnitID
			breaks = append(breaks, ruleBreak)
		}
	}

	if len(breaks) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&breaks).
		Returning("*").
		Exec(ctx)

	return err
}

func (r *repository) ListVersions(
	ctx context.Context,
	req *repositories.ListRateAgreementVersionsRequest,
) (*pagination.ListResult[*rateagreement.RateAgreementVersion], error) {
	log := r.l.With(zap.String("operation", "ListVersions"))

	cols := buncolgen.RateAgreementVersionColumns
	entities := make([]*rateagreement.RateAgreementVersion, 0)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.Rel(buncolgen.RateAgreementVersionRelations.CreatedBy)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateAgreementVersionScopeTenant(sq, req.TenantInfo).
				Where(cols.RateAgreementID.Eq(), req.RateAgreementID)
		}).
		Order(cols.VersionNumber.OrderDesc())

	if req.Limit > 0 {
		q = q.Limit(req.Limit).Offset(req.Offset)
	}

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to list rate agreement versions", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*rateagreement.RateAgreementVersion]{
		Items: entities,
		Total: total,
	}, nil
}

// GetEffectiveVersion returns the header terms that governed at a moment.
//
// A missing version is not an error. Agreements written before anybody amended
// them have no version rows at all, and the caller falls back to the live
// header, which is by definition what was in force.
func (r *repository) GetEffectiveVersion(
	ctx context.Context,
	req *repositories.GetEffectiveAgreementVersionRequest,
) (*rateagreement.RateAgreementVersion, error) {
	log := r.l.With(zap.String("operation", "GetEffectiveVersion"))

	cols := buncolgen.RateAgreementVersionColumns
	entity := new(rateagreement.RateAgreementVersion)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateAgreementVersionScopeTenant(sq, req.TenantInfo).
				Where(cols.RateAgreementID.Eq(), req.RateAgreementID).
				Where(cols.EffectiveFrom.Lte(), req.AsOf).
				WhereGroup(" AND ", func(wq *bun.SelectQuery) *bun.SelectQuery {
					return wq.
						Where(cols.EffectiveTo.IsNull()).
						WhereOr(cols.EffectiveTo.Gt(), req.AsOf)
				})
		}).
		Order(cols.VersionNumber.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // an agreement never amended has no version rows
		}
		log.Error("failed to get effective agreement version", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// CreateVersion closes the version it supersedes and writes the new one, so the
// history is a contiguous chain with exactly one version live at any moment.
func (r *repository) CreateVersion(
	ctx context.Context,
	version *rateagreement.RateAgreementVersion,
) (*rateagreement.RateAgreementVersion, error) {
	log := r.l.With(
		zap.String("operation", "CreateVersion"),
		zap.String("agreementId", version.RateAgreementID.String()),
	)

	cols := buncolgen.RateAgreementVersionColumns
	tenantInfo := pagination.TenantInfo{
		OrgID: version.OrganizationID,
		BuID:  version.BusinessUnitID,
	}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model((*rateagreement.RateAgreementVersion)(nil)).
			Set(cols.EffectiveTo.Set(), version.EffectiveFrom).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.RateAgreementVersionScopeTenantUpdate(uq, tenantInfo).
					Where(cols.RateAgreementID.Eq(), version.RateAgreementID).
					Where(cols.EffectiveTo.IsNull())
			}).
			Exec(c); uErr != nil {
			return uErr
		}

		_, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(version).
			Returning("*").
			Exec(c)

		return iErr
	})
	if err != nil {
		log.Error("failed to create rate agreement version", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Rate agreement is busy. Retry the request.",
		)
	}

	return version, nil
}
