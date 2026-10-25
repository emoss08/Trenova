package rateagreementrepository

import (
	"context"
	"sort"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/postgis"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// DefaultCandidateLimit bounds how many rules one lane may return.
//
// A tenant whose lane genuinely matches more rules than this has a data quality
// problem — overlapping contracts, or lanes written far broader than intended —
// and the quote says so rather than quietly scanning through them.
const DefaultCandidateLimit = 200

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
	req *rateagreementports.ResolveRateRulesRequest,
) (*rateagreementports.ResolveRateRulesResult, error) {
	log := r.l.With(
		zap.String("operation", "ResolveRules"),
		zap.String("partyType", string(req.PartyType)),
		zap.Int("laneKeys", len(req.LaneKeys)),
	)

	limit := req.Limit
	if limit <= 0 {
		limit = DefaultCandidateLimit
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*rateagreementports.ResolveRateRulesResult, error) {
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
			if err = r.attachAgreements(ctx, merged); err != nil {
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

			return &rateagreementports.ResolveRateRulesResult{
				Rules:  merged,
				Total:  total,
				Capped: capped,
			}, nil
		},
	)
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
func candidateOrder(
	q *keel.Query[rateagreement.RateAgreementRule],
) *keel.Query[rateagreement.RateAgreementRule] {
	rr := rateagreementdb.RateAgreementRules
	ra := rateagreementdb.RateAgreements

	return q.Order(
		rr.Priority.Desc(),
		ra.Priority.Desc(),
		rr.SpecificityScore.Desc(),
		rr.EffectiveFrom.Desc(),
		rr.ID.Asc(),
	)
}

func baseCandidateQuery(
	req *rateagreementports.ResolveRateRulesRequest,
) *keel.Query[rateagreement.RateAgreementRule] {
	rr := rateagreementdb.RateAgreementRules
	ra := rateagreementdb.RateAgreements

	// A rule cannot outlive the contract it belongs to. Suspending or
	// expiring an agreement has to stop it pricing even where its rules
	// are still individually within their own windows, which is how a
	// contract is taken out of service without editing every lane.
	//
	// One agreement can be exempted, for a simulation: a draft contract
	// is by definition not Active, and the whole point is to see what it
	// would do. Relaxing the real query rather than writing a second one
	// is what makes a simulated rate the same rate the contract will
	// produce once it is signed.
	inService := keel.And(
		rr.Status.Eq(rateagreement.RuleStatusActive),
		ra.Status.Eq(rateagreement.StatusActive),
		effectiveAt(ruleWindowColumns{from: ra.EffectiveFrom, to: ra.EffectiveTo}, req.AsOf),
	)
	if req.SimulateAgreementID != nil && !req.SimulateAgreementID.IsNil() {
		inService = keel.Or(inService, rr.RateAgreementID.Eq(*req.SimulateAgreementID))
	}

	return rr.Join(ra, ra.ID.EqCol(rr.RateAgreementID)).Where(
		rr.PartyType.Eq(req.PartyType),
		effectiveAt(ruleWindow(), req.AsOf),
		keel.When(len(req.PartyIDs) > 0, rr.PartyID.In(req.PartyIDs...)),
		inService,
	)
}

func keyedRulesQuery(
	req *rateagreementports.ResolveRateRulesRequest,
	limit int,
) *keel.Query[rateagreement.RateAgreementRule] {
	rr := rateagreementdb.RateAgreementRules

	return candidateOrder(baseCandidateQuery(req).Where(rr.LaneKey.In(req.LaneKeys...))).
		Limit(limit + 1)
}

func (r *repository) resolveKeyedRules(
	ctx context.Context,
	req *rateagreementports.ResolveRateRulesRequest,
	limit int,
) ([]*rateagreement.RateAgreementRule, error) {
	if len(req.LaneKeys) == 0 {
		return nil, nil
	}

	// One row over the limit, so the caller can tell a full page from a capped
	// one without a second count.
	return keyedRulesQuery(req, limit).List(ctx, r.db.Keel(ctx))
}

// resolveRadiusRules finds the lanes described by a circle around a point.
//
// These cannot be reduced to a key, so they are matched with a geospatial
// predicate against the partial GiST indexes instead. There are only ever a
// handful per organization, which is why a second query is cheaper than trying
// to force both shapes through one.
func radiusMatch(scope, center, radius keel.Expr, lon, lat float64) keel.Pred {
	return keel.Or(
		keel.DynOf(scope).Ne(keel.DynConst(string(rategeo.ScopeTypeRadius))),
		postgis.DWithin(center, postgis.Pt(lon, lat), radius),
	)
}

func radiusRulesQuery(
	req *rateagreementports.ResolveRateRulesRequest,
	limit int,
) *keel.Query[rateagreement.RateAgreementRule] {
	rr := rateagreementdb.RateAgreementRules

	return candidateOrder(baseCandidateQuery(req).Where(
		rr.LaneKey.Eq(rateagreement.RadiusLaneKey),
		radiusMatch(rr.OriginScopeType, rr.OriginCenter, rr.OriginRadiusMeters,
			*req.OriginLongitude, *req.OriginLatitude),
		radiusMatch(rr.DestinationScopeType, rr.DestinationCenter, rr.DestinationRadiusMeters,
			*req.DestinationLongitude, *req.DestinationLatitude),
	)).Limit(limit + 1)
}

func (r *repository) resolveRadiusRules(
	ctx context.Context,
	req *rateagreementports.ResolveRateRulesRequest,
	limit int,
) ([]*rateagreement.RateAgreementRule, error) {
	if req.OriginLatitude == nil || req.OriginLongitude == nil ||
		req.DestinationLatitude == nil || req.DestinationLongitude == nil {
		return nil, nil
	}

	return radiusRulesQuery(req, limit).List(ctx, r.db.Keel(ctx))
}

// attachAgreements loads the handful of distinct agreements behind a candidate
// set and hangs each one off its rules.
//
// The engine needs the agreement for currency, rounding, guardrail defaults and
// the name that goes in the trace, and the ordering needs its priority.
func (r *repository) attachAgreements(
	ctx context.Context,
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

	ra := rateagreementdb.RateAgreements
	agreements, err := ra.Where(ra.ID.In(ids...)).Limit(len(ids)).List(ctx, r.db.Keel(ctx))
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
