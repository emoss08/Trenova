package rateagreementrepository

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// DefaultCandidateLimit bounds how many rules one lane may return.
//
// A tenant whose lane genuinely matches more rules than this has a data quality
// problem — overlapping contracts, or lanes written far broader than intended —
// and the quote says so rather than quietly scanning through them.
const DefaultCandidateLimit = 200

// agreementJoin joins the owning agreement under the alias buncolgen generates
// for it, so every reference to an agreement column in this file is the same
// generated constant used everywhere else.
//
// It is written out rather than declared with bun's Relation because a relation
// join is aliased by the relation's name, and ordering by a column under an
// alias the ORM chose would silently stop working if that field were ever
// renamed. Which rule wins is not something to leave resting on that.
func agreementJoin() string {
	agreementCols := buncolgen.RateAgreementColumns
	ruleCols := buncolgen.RateAgreementRuleColumns

	return "JOIN " + buncolgen.RateAgreementTable.As(buncolgen.RateAgreementTable.Alias) +
		" ON " + agreementCols.ID.Qualified() + " = " + ruleCols.RateAgreementID.Qualified() +
		" AND " + agreementCols.OrganizationID.Qualified() + " = " +
		ruleCols.OrganizationID.Qualified() +
		" AND " + agreementCols.BusinessUnitID.Qualified() + " = " +
		ruleCols.BusinessUnitID.Qualified()
}

// ResolveRules fetches every rule that could price one lane.
//
// The work splits in two because the two ways of naming a place cannot share an
// index. Keyed lanes — everything from a postal code to a country — are found
// by matching the rule's stored lane key against the handful a shipment could
// produce, which the planner answers as a bitmap of selective index probes.
// Radius lanes have no key and are found geospatially instead. The two result
// sets are merged here, deduplicated, and put in the one total ordering the
// engine relies on.
//
// The owning agreements are loaded in a second query rather than joined into
// the model. There are only ever a few distinct agreements behind a candidate
// set, and keeping them out of the candidate query leaves it reading nothing
// but the covering index.
func (r *repository) ResolveRules(
	ctx context.Context,
	req *repositories.ResolveRateRulesRequest,
) (*repositories.ResolveRateRulesResult, error) {
	log := r.l.With(
		zap.String("operation", "ResolveRules"),
		zap.String("partyType", string(req.PartyType)),
		zap.Int("laneKeys", len(req.LaneKeys)),
	)

	limit := req.Limit
	if limit <= 0 {
		limit = DefaultCandidateLimit
	}

	keyed, err := r.resolveKeyedRules(ctx, req, limit)
	if err != nil {
		log.Error("failed to resolve keyed rate rules", zap.Error(err))
		return nil, err
	}

	radius, err := r.resolveRadiusRules(ctx, req, limit)
	if err != nil {
		log.Error("failed to resolve radius rate rules", zap.Error(err))
		return nil, err
	}

	merged := mergeCandidates(keyed, radius)

	if err = r.attachAgreements(ctx, req.TenantInfo, merged); err != nil {
		log.Error("failed to load agreements for candidates", zap.Error(err))
		return nil, err
	}

	// Agreement priority only enters the ordering once the agreements are
	// loaded, so the final sort happens after they are attached.
	sortCandidates(merged)

	total := len(merged)
	capped := total > limit
	if capped {
		merged = merged[:limit]
	}

	return &repositories.ResolveRateRulesResult{
		Rules:  merged,
		Total:  total,
		Capped: capped,
	}, nil
}

// candidateOrder is the ordering the database applies. It stops short of
// agreement priority, which is settled in Go once the agreements are attached;
// everything below that point is here so the limit cuts the right rows.
//
// The last term is not decoration. PULIDs sort by creation, so two rules that
// are indistinguishable on every other term always resolve to the older one —
// on every server, on every run. Without it the winner would follow whatever
// order the database happened to return, and the same shipment could rate two
// ways.
func candidateOrder(q *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.RateAgreementRuleColumns
	agreementCols := buncolgen.RateAgreementColumns

	return q.
		OrderExpr(cols.Priority.Qualified() + " DESC").
		OrderExpr(agreementCols.Priority.Qualified() + " DESC").
		OrderExpr(cols.SpecificityScore.Qualified() + " DESC").
		OrderExpr(cols.EffectiveFrom.Qualified() + " DESC").
		OrderExpr(cols.ID.Qualified() + " ASC")
}

func (r *repository) baseCandidateQuery(
	ctx context.Context,
	req *repositories.ResolveRateRulesRequest,
	entities *[]*rateagreement.RateAgreementRule,
) *bun.SelectQuery {
	cols := buncolgen.RateAgreementRuleColumns
	agreementCols := buncolgen.RateAgreementColumns

	return r.db.DBForContext(ctx).
		NewSelect().
		Model(entities).
		ColumnExpr(buncolgen.RateAgreementRuleTable.All()).
		Join(agreementJoin()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.RateAgreementRuleScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.Eq(), rateagreement.RuleStatusActive).
				Where(cols.PartyType.Eq(), req.PartyType).
				Where(cols.EffectiveFrom.Lte(), req.AsOf).
				WhereGroup(" AND ", func(wq *bun.SelectQuery) *bun.SelectQuery {
					return wq.
						Where(cols.EffectiveTo.IsNull()).
						WhereOr(cols.EffectiveTo.Gt(), req.AsOf)
				})

			if len(req.PartyIDs) > 0 {
				sq = sq.Where(cols.PartyID.In(), bun.In(req.PartyIDs))
			}

			// A rule cannot outlive the contract it belongs to. Suspending or
			// expiring an agreement has to stop it pricing even where its rules
			// are still individually within their own windows, which is how a
			// contract is taken out of service without editing every lane.
			return sq.
				Where(agreementCols.Status.Eq(), rateagreement.StatusActive).
				Where(agreementCols.EffectiveFrom.Lte(), req.AsOf).
				WhereGroup(" AND ", func(wq *bun.SelectQuery) *bun.SelectQuery {
					return wq.
						Where(agreementCols.EffectiveTo.IsNull()).
						WhereOr(agreementCols.EffectiveTo.Gt(), req.AsOf)
				})
		})
}

func (r *repository) resolveKeyedRules(
	ctx context.Context,
	req *repositories.ResolveRateRulesRequest,
	limit int,
) ([]*rateagreement.RateAgreementRule, error) {
	if len(req.LaneKeys) == 0 {
		return nil, nil
	}

	cols := buncolgen.RateAgreementRuleColumns
	entities := make([]*rateagreement.RateAgreementRule, 0, limit)

	// One row over the limit, so the caller can tell a full page from a capped
	// one without a second count.
	err := r.baseCandidateQuery(ctx, req, &entities).
		Where(cols.LaneKey.In(), bun.In(req.LaneKeys)).
		Apply(candidateOrder).
		Limit(limit + 1).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

// resolveRadiusRules finds the lanes described by a circle around a point.
//
// These cannot be reduced to a key, so they are matched with a geospatial
// predicate against the partial GiST indexes instead. There are only ever a
// handful per organization, which is why a second query is cheaper than trying
// to force both shapes through one.
func (r *repository) resolveRadiusRules(
	ctx context.Context,
	req *repositories.ResolveRateRulesRequest,
	limit int,
) ([]*rateagreement.RateAgreementRule, error) {
	if req.OriginLatitude == nil || req.OriginLongitude == nil ||
		req.DestinationLatitude == nil || req.DestinationLongitude == nil {
		return nil, nil
	}

	cols := buncolgen.RateAgreementRuleColumns
	entities := make([]*rateagreement.RateAgreementRule, 0, limit)

	err := r.baseCandidateQuery(ctx, req, &entities).
		Where(cols.LaneKey.Eq(), rateagreement.RadiusLaneKey).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(
					cols.OriginScopeType.Qualified()+" <> 'Radius'"+
						" OR ST_DWithin("+cols.OriginCenter.Qualified()+
						", ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, "+
						cols.OriginRadiusMeters.Qualified()+")",
					*req.OriginLongitude, *req.OriginLatitude,
				).
				Where(
					cols.DestinationScopeType.Qualified()+" <> 'Radius'"+
						" OR ST_DWithin("+cols.DestinationCenter.Qualified()+
						", ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, "+
						cols.DestinationRadiusMeters.Qualified()+")",
					*req.DestinationLongitude, *req.DestinationLatitude,
				)
		}).
		Apply(candidateOrder).
		Limit(limit + 1).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

// attachAgreements loads the handful of distinct agreements behind a candidate
// set and hangs each one off its rules.
//
// The engine needs the agreement for currency, rounding, guardrail defaults and
// the name that goes in the trace, and the ordering needs its priority.
func (r *repository) attachAgreements(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rules []*rateagreement.RateAgreementRule,
) error {
	if len(rules) == 0 {
		return nil
	}

	ids := make([]pulid.ID, 0, len(rules))
	seen := make(map[pulid.ID]struct{}, len(rules))
	for _, rule := range rules {
		if _, dup := seen[rule.RateAgreementID]; dup {
			continue
		}
		seen[rule.RateAgreementID] = struct{}{}
		ids = append(ids, rule.RateAgreementID)
	}

	cols := buncolgen.RateAgreementColumns
	agreements := make([]*rateagreement.RateAgreement, 0, len(ids))

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&agreements).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateAgreementScopeTenant(sq, tenantInfo).
				Where(cols.ID.In(), bun.In(ids))
		}).
		Scan(ctx)
	if err != nil {
		return err
	}

	byID := make(map[pulid.ID]*rateagreement.RateAgreement, len(agreements))
	for _, agreement := range agreements {
		byID[agreement.ID] = agreement
	}

	for _, rule := range rules {
		rule.Agreement = byID[rule.RateAgreementID]
	}

	return nil
}

// mergeCandidates combines the two result sets, dropping duplicates.
//
// Deduplication is by rule id: a rule with a radius on one end and a keyed
// scope on the other is deliberately reachable from both queries.
func mergeCandidates(
	sets ...[]*rateagreement.RateAgreementRule,
) []*rateagreement.RateAgreementRule {
	size := 0
	for _, set := range sets {
		size += len(set)
	}

	merged := make([]*rateagreement.RateAgreementRule, 0, size)
	seen := make(map[pulid.ID]struct{}, size)

	for _, set := range sets {
		for _, rule := range set {
			if rule == nil {
				continue
			}
			if _, dup := seen[rule.ID]; dup {
				continue
			}
			seen[rule.ID] = struct{}{}
			merged = append(merged, rule)
		}
	}

	return merged
}

func sortCandidates(rules []*rateagreement.RateAgreementRule) {
	sort.SliceStable(rules, func(a, b int) bool {
		return lessCandidate(rules[a], rules[b])
	})
}

// lessCandidate mirrors candidateOrder, plus the agreement priority the
// database could not order by before the agreements were attached. The two have
// to agree wherever they overlap, because a disagreement would change which
// rule wins depending on whether the set came back from one query or two.
func lessCandidate(a, b *rateagreement.RateAgreementRule) bool {
	if a.Priority != b.Priority {
		return a.Priority > b.Priority
	}

	if agreementPriority(a) != agreementPriority(b) {
		return agreementPriority(a) > agreementPriority(b)
	}

	if a.SpecificityScore != b.SpecificityScore {
		return a.SpecificityScore > b.SpecificityScore
	}

	if a.EffectiveFrom != b.EffectiveFrom {
		return a.EffectiveFrom > b.EffectiveFrom
	}

	return a.ID.String() < b.ID.String()
}

func agreementPriority(rule *rateagreement.RateAgreementRule) int16 {
	if rule.Agreement == nil {
		return 0
	}

	return rule.Agreement.Priority
}
