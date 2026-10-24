package rateagreementrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
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
	req *rateagreementports.AmendRateAgreementRulesRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(req.SupersededIDs) == 0 && len(req.Rules) == 0 {
			return nil
		}

		err := r.closeOutRules(ctx, req)
		if err == nil {
			err = r.insertRules(ctx, req)
		}
		if err != nil {
			r.l.Error("failed to amend rate agreement rules",
				zap.String("agreementId", req.RateAgreementID.String()),
				zap.Int("superseded", len(req.SupersededIDs)),
				zap.Int("inserted", len(req.Rules)),
				zap.Error(err),
			)
			return mapAgreementError(err, req.RateAgreementID.String())
		}

		return nil
	})
}

// closeOutRules ends the superseded rules at the instant their replacements
// begin. Windows are half open, so the two never overlap and never leave a gap.
//
// A rule that already ends earlier is left alone: an amendment should not
// extend the life of something already retired.
func (r *repository) closeOutRules(
	ctx context.Context,
	req *rateagreementports.AmendRateAgreementRulesRequest,
) error {
	if len(req.SupersededIDs) == 0 {
		return nil
	}

	rr := rateagreementdb.RateAgreementRules
	_, err := rr.Where(
		rr.RateAgreementID.Eq(req.RateAgreementID),
		rr.ID.In(req.SupersededIDs...),
		openAt(rr.EffectiveTo, req.EffectiveFrom),
	).Update(ctx, r.db.Keel(ctx), rr.EffectiveTo.To(req.EffectiveFrom))

	return err
}

func (r *repository) insertRules(
	ctx context.Context,
	req *rateagreementports.AmendRateAgreementRulesRequest,
) error {
	rules := make([]*rateagreement.RateAgreementRule, 0, len(req.Rules))
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
		rules = append(rules, rule)
	}
	if len(rules) == 0 {
		return nil
	}

	return rateagreementdb.RateAgreementRules.CreateMany(ctx, r.db.Keel(ctx), rules)
}

func versionsQuery(
	req *rateagreementports.ListRateAgreementVersionsRequest,
) *keel.Query[rateagreement.RateAgreementVersion] {
	v := rateagreementdb.RateAgreementVersions

	return v.Where(v.RateAgreementID.Eq(req.RateAgreementID))
}

func (r *repository) ListVersions(
	ctx context.Context,
	req *rateagreementports.ListRateAgreementVersionsRequest,
) (*pagination.ListResult[*rateagreement.RateAgreementVersion], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*rateagreement.RateAgreementVersion], error) {
			v := rateagreementdb.RateAgreementVersions
			rows := versionsQuery(req).Load(v.CreatedBy()).Order(v.VersionNumber.Desc())

			var entities []*rateagreement.RateAgreementVersion
			var total int64
			read := keel.AllInto(rows, &entities)
			if req.Limit > 0 {
				read = keel.ListInto(rows.Limit(req.Limit).Offset(req.Offset), &entities)
			}
			if err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				read,
				keel.CountInto(versionsQuery(req), &total),
			); err != nil {
				r.l.Error("failed to list rate agreement versions", zap.Error(err))
				return nil, err
			}

			if err := r.resolveAccessorialNames(ctx, entities); err != nil {
				r.l.Error("failed to resolve accessorial names for versions", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*rateagreement.RateAgreementVersion]{
				Items: entities,
				Total: int(total),
			}, nil
		},
	)
}

// resolveAccessorialNames patches each version's id → code map for every
// accessorial charge its snapshot or change summary mentions. Names are looked
// up at read time rather than stored, so a charge renamed after the fact still
// reads under its current code — the terms themselves stay immutable, keyed by
// id.
func (r *repository) resolveAccessorialNames(
	ctx context.Context,
	versions []*rateagreement.RateAgreementVersion,
) error {
	chargeIDs := collectAccessorialChargeIDs(versions)
	if len(chargeIDs) == 0 {
		return nil
	}

	ac := accessorialchargedb.AccessorialCharges
	charges, err := ac.Pick(ac.ID, ac.Code).
		Where(ac.ID.In(chargeIDs...)).
		Limit(len(chargeIDs)).
		List(ctx, r.db.Keel(ctx))
	if err != nil {
		return err
	}

	names := make(map[string]string, len(charges))
	for _, charge := range charges {
		names[charge.ID.String()] = charge.Code
	}

	for _, version := range versions {
		versionNames := make(map[string]string)
		for id := range version.AccessorialTerms {
			if code, ok := names[id]; ok {
				versionNames[id] = code
			}
		}
		for _, id := range summaryAccessorialChargeIDs(version.ChangeSummary) {
			if code, ok := names[id]; ok {
				versionNames[id] = code
			}
		}
		if len(versionNames) > 0 {
			version.AccessorialNames = versionNames
		}
	}

	return nil
}

func collectAccessorialChargeIDs(
	versions []*rateagreement.RateAgreementVersion,
) []pulid.ID {
	seen := make(map[string]struct{})
	for _, version := range versions {
		for id := range version.AccessorialTerms {
			seen[id] = struct{}{}
		}
		for _, id := range summaryAccessorialChargeIDs(version.ChangeSummary) {
			seen[id] = struct{}{}
		}
	}

	ids := make([]pulid.ID, 0, len(seen))
	for id := range seen {
		ids = append(ids, pulid.ID(id))
	}

	return ids
}

const accessorialTermsPathPrefix = "accessorialTerms."

// summaryAccessorialChargeIDs reads the charge ids out of a change summary's
// paths — "accessorialTerms.<id>" or "accessorialTerms.<id>.<field>". A
// removed accessorial appears only here: its id is gone from the snapshot but
// its removal is still a change somebody will want named.
func summaryAccessorialChargeIDs(
	summary map[string]jsonutils.FieldChange,
) []string {
	ids := make([]string, 0, len(summary))
	for path := range summary {
		rest, ok := strings.CutPrefix(path, accessorialTermsPathPrefix)
		if !ok {
			continue
		}
		if id, _, _ := strings.Cut(rest, "."); id != "" {
			ids = append(ids, id)
		}
	}

	return ids
}

// GetEffectiveVersion returns the header terms that governed at a moment.
//
// A missing version is not an error. Agreements written before anybody amended
// them have no version rows at all, and the caller falls back to the live
// header, which is by definition what was in force.
func (r *repository) GetEffectiveVersion(
	ctx context.Context,
	req *rateagreementports.GetEffectiveAgreementVersionRequest,
) (*rateagreement.RateAgreementVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*rateagreement.RateAgreementVersion, error) {
			v := rateagreementdb.RateAgreementVersions
			entity, err := v.Where(
				v.RateAgreementID.Eq(req.RateAgreementID),
				effectiveAt(ruleWindowColumns{from: v.EffectiveFrom, to: v.EffectiveTo}, req.AsOf),
			).Order(v.VersionNumber.Desc()).First(ctx, r.db.Keel(ctx))
			if err != nil {
				if dberror.IsNotFoundError(err) {
					return nil, nil //nolint:nilnil // an agreement never amended has no version rows
				}
				r.l.Error("failed to get effective agreement version", zap.Error(err))
				return nil, err
			}

			return entity, nil
		},
	)
}

// CreateVersion closes the version it supersedes and writes the new one, so the
// history is a contiguous chain with exactly one version live at any moment.
func (r *repository) CreateVersion(
	ctx context.Context,
	version *rateagreement.RateAgreementVersion,
) (*rateagreement.RateAgreementVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*rateagreement.RateAgreementVersion, error) {
			db := r.db.Keel(ctx)
			v := rateagreementdb.RateAgreementVersions
			_, err := v.Where(v.RateAgreementID.Eq(version.RateAgreementID), v.EffectiveTo.IsNull()).
				Update(ctx, db, v.EffectiveTo.To(version.EffectiveFrom))
			if err == nil {
				err = v.Create(ctx, db, version)
			}
			if err != nil {
				r.l.Error("failed to create rate agreement version",
					zap.String("agreementId", version.RateAgreementID.String()),
					zap.Error(err),
				)
				return nil, dberror.MapRetryableTransactionError(
					versionErrors.Map(err, version.ID.String()),
					busyMessage,
				)
			}

			return version, nil
		},
	)
}
