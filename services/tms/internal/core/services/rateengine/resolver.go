package rateengine

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Resolution is the outcome of choosing a rule, before it is priced.
type Resolution struct {
	// Winner is the rule that will price the shipment, or nil when nothing
	// covered the lane.
	Winner *rateagreement.RateAgreementRule
	// Candidates is every rule that was considered, winner included, in the
	// order they were ranked.
	Candidates []ratetypes.Candidate
	// TieBreak names the ordering step that actually separated the winner from
	// the runner up, which is the question a pricing analyst asks first when a
	// rate surprises them.
	TieBreak string
	// LaneKeysTried is what the lane was looked up under. When nothing matched
	// this is the answer to why: no rule is keyed to any of them.
	LaneKeysTried  []string
	CandidateCount int32
	Capped         bool
}

// resolve picks the rule that prices a shipment.
//
// The database has already narrowed to the rules whose lane, party, status and
// effective window fit. What remains are the conditions an index cannot serve —
// weight and distance bands, commodity and class sets, equipment, day of week —
// which are checked here over the handful of rows that came back. Every
// rejection is recorded with a reason, because "no rate applied" and "a rate
// applied but not the one you expected" are different problems and only the
// full list tells them apart.
func (s *Service) resolve(
	ctx context.Context,
	rateCtx *RateContext,
) (*Resolution, error) {
	log := s.l.With(
		zap.String("operation", "resolve"),
		zap.String("partyType", string(rateCtx.PartyType)),
	)

	laneKeys := rateCtx.LaneKeys()

	// No party means no contract, and querying for one would be a scan across
	// every rule in the tenant to find nothing.
	if rateCtx.PartyID.IsNil() {
		return &Resolution{LaneKeysTried: laneKeys}, nil
	}

	fetched, err := s.agreementRepo.ResolveRules(ctx, &repositories.ResolveRateRulesRequest{
		TenantInfo:           rateCtx.TenantInfo,
		PartyType:            rateCtx.PartyType,
		PartyIDs:             []pulid.ID{rateCtx.PartyID},
		LaneKeys:             laneKeys,
		AsOf:                 rateCtx.AsOf,
		OriginLatitude:       rateCtx.OriginLatitude,
		OriginLongitude:      rateCtx.OriginLongitude,
		DestinationLatitude:  rateCtx.DestinationLatitude,
		DestinationLongitude: rateCtx.DestinationLongitude,
		SimulateAgreementID:  rateCtx.SimulateAgreementID,
	})
	if err != nil {
		log.Error("failed to fetch rate rule candidates", zap.Error(err))
		return nil, err
	}

	resolution := &Resolution{
		LaneKeysTried:  laneKeys,
		CandidateCount: int32(fetched.Total), //nolint:gosec // bounded by the resolve query limit
		Capped:         fetched.Capped,
		Candidates:     make([]ratetypes.Candidate, 0, len(fetched.Rules)),
	}

	facts := rateCtx.MatchFacts()
	rank := int16(0)

	for _, rule := range fetched.Rules {
		matched, reason := rule.Matches(facts)

		candidate := describeCandidate(rule, reason)

		if !matched {
			resolution.Candidates = append(resolution.Candidates, candidate)
			continue
		}

		if resolution.Winner == nil {
			resolution.Winner = rule
			candidate.Won = true
			candidate.RejectReason = ratetypes.RejectReasonNone
			candidate.MatchedOn = matchedOn(rule)
		} else {
			// Everything that matched but arrived later lost, and the ordering
			// term that put the winner first is the reason. The sentence has to
			// move with it: a rule that lost on specificity saying "Applied" is
			// the opposite of what happened.
			candidate.RejectReason = lossReason(resolution.Winner, rule)
			candidate.RejectDetail = candidate.RejectReason.Explanation()
		}

		candidate.Rank = rank
		rank++
		resolution.Candidates = append(resolution.Candidates, candidate)
	}

	if resolution.Winner != nil {
		resolution.TieBreak = tieBreak(
			resolution.Winner,
			runnerUp(fetched.Rules, facts, resolution.Winner),
		)
	}

	if fetched.Capped {
		log.Warn("rate rule candidates were capped",
			zap.Int("total", fetched.Total),
			zap.String("partyId", rateCtx.PartyID.String()),
		)
	}

	return resolution, nil
}

func describeCandidate(
	rule *rateagreement.RateAgreementRule,
	reason ratetypes.RejectReason,
) ratetypes.Candidate {
	candidate := ratetypes.Candidate{
		AgreementID:      rule.RateAgreementID.String(),
		RuleID:           rule.ID.String(),
		RuleLabel:        rule.Label,
		LaneKey:          rule.LaneKey,
		SpecificityScore: rule.SpecificityScore,
		RulePriority:     rule.Priority,
		EffectiveFrom:    rule.EffectiveFrom,
		EffectiveTo:      rule.EffectiveTo,
		RejectReason:     reason,
		RejectDetail:     reason.Explanation(),
	}

	if rule.Agreement != nil {
		candidate.AgreementCode = rule.Agreement.Code
		candidate.AgreementName = rule.Agreement.Name
		candidate.AgreementPriority = rule.Agreement.Priority
	}

	return candidate
}

// matchedOn lists what the winning rule narrowed on, in the same vocabulary
// mode profile provenance uses, so the two read alike in the interface.
func matchedOn(rule *rateagreement.RateAgreementRule) []string {
	matched := make([]string, 0, 8)

	matched = append(matched,
		"origin:"+rule.OriginScopeType.String(),
		"destination:"+rule.DestinationScopeType.String(),
	)

	if len(rule.CommodityIDs) > 0 {
		matched = append(matched, "commodity")
	}
	if len(rule.FreightClasses) > 0 {
		matched = append(matched, "freightClass")
	}
	if len(rule.TractorTypeIDs) > 0 || len(rule.TrailerTypeIDs) > 0 ||
		len(rule.EquipmentClasses) > 0 {
		matched = append(matched, "equipment")
	}
	if len(rule.ServiceTypeIDs) > 0 {
		matched = append(matched, "serviceType")
	}
	if len(rule.ShipmentTypeIDs) > 0 {
		matched = append(matched, "shipmentType")
	}
	if rule.MinWeight.Valid || rule.MaxWeight.Valid {
		matched = append(matched, "weight")
	}
	if rule.MinDistance.Valid || rule.MaxDistance.Valid {
		matched = append(matched, "distance")
	}
	if len(rule.ServiceModels) > 0 {
		matched = append(matched, "mode")
	}
	if rule.HazmatOnly {
		matched = append(matched, "hazmat")
	}
	if rule.TempControlOnly {
		matched = append(matched, "temperatureControl")
	}

	return matched
}

// lossReason names the ordering term that put the winner ahead of this rule.
//
// It is deliberately specific. "Another rate was written more specifically" and
// "another rate took effect more recently" send a pricing analyst to completely
// different places, and a generic "lost" would send them to neither.
func lossReason(
	winner, loser *rateagreement.RateAgreementRule,
) ratetypes.RejectReason {
	switch {
	case winner.Priority != loser.Priority:
		return ratetypes.RejectReasonLostOnPriority
	case agreementPriorityOf(winner) != agreementPriorityOf(loser):
		return ratetypes.RejectReasonLostOnPriority
	case winner.SpecificityScore != loser.SpecificityScore:
		return ratetypes.RejectReasonLostOnSpecificity
	case winner.EffectiveFrom != loser.EffectiveFrom:
		return ratetypes.RejectReasonLostOnEffectiveDate
	default:
		return ratetypes.RejectReasonLostOnTiebreak
	}
}

// tieBreak describes, for the trace, how the winner beat the closest rule that
// also applied.
func tieBreak(winner, runnerUp *rateagreement.RateAgreementRule) string {
	if runnerUp == nil {
		return "only matching rate"
	}

	switch {
	case winner.Priority != runnerUp.Priority:
		return "rule priority " + strconv.Itoa(int(winner.Priority))
	case agreementPriorityOf(winner) != agreementPriorityOf(runnerUp):
		return "agreement priority " + strconv.Itoa(int(agreementPriorityOf(winner)))
	case winner.SpecificityScore != runnerUp.SpecificityScore:
		return "specificity " + strconv.Itoa(int(winner.SpecificityScore))
	case winner.EffectiveFrom != runnerUp.EffectiveFrom:
		return "most recently effective"
	default:
		return "created first"
	}
}

// runnerUp is the next rule that also applied, which is what the winner's
// margin is described against.
func runnerUp(
	rules []*rateagreement.RateAgreementRule,
	facts *rateagreement.MatchFacts,
	winner *rateagreement.RateAgreementRule,
) *rateagreement.RateAgreementRule {
	seenWinner := false

	for _, rule := range rules {
		if rule == winner {
			seenWinner = true
			continue
		}
		if !seenWinner {
			continue
		}
		if matched, _ := rule.Matches(facts); matched {
			return rule
		}
	}

	return nil
}

func agreementPriorityOf(rule *rateagreement.RateAgreementRule) int16 {
	if rule == nil || rule.Agreement == nil {
		return 0
	}

	return rule.Agreement.Priority
}

// resolveZones fills in the zones each end of the lane belongs to.
//
// Zone membership is the one part of the lane a shipment cannot work out for
// itself, and it has to be settled before the candidate query runs because the
// zone ids become lane keys.
func (s *Service) resolveZones(ctx context.Context, rateCtx *RateContext) error {
	origin, err := s.zoneRepo.ResolveMembership(
		ctx,
		&repositories.ResolveZoneMembershipRequest{
			TenantInfo: rateCtx.TenantInfo,
			MatchKeys:  rateCtx.Origin.MatchKeys(),
		},
	)
	if err != nil {
		return err
	}

	destination, err := s.zoneRepo.ResolveMembership(
		ctx,
		&repositories.ResolveZoneMembershipRequest{
			TenantInfo: rateCtx.TenantInfo,
			MatchKeys:  rateCtx.Destination.MatchKeys(),
		},
	)
	if err != nil {
		return err
	}

	rateCtx.Origin.ZoneIDs = origin
	rateCtx.Destination.ZoneIDs = destination

	return nil
}

// totalsFor splits a linehaul into the shape an invoice reads. Fuel and
// accessorials are added by the caller that owns them; the engine only ever
// produces the linehaul.
func totalsFor(linehaul decimal.Decimal) ratetypes.Totals {
	return ratetypes.Totals{
		Linehaul: linehaul,
		Total:    linehaul,
	}
}
