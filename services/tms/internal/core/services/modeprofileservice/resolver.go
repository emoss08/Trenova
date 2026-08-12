package modeprofileservice

import (
	"context"
	"slices"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	rejectionNotEffective  = "Profile is outside its effective date window"
	rejectionCustomer      = "Profile targets a different customer"
	rejectionShipmentType  = "Profile targets a different shipment type"
	rejectionServiceType   = "Profile targets a different service type"
	rejectionEquipmentType = "Profile targets a different equipment type"
	rejectionOutranked     = "A higher priority profile matched first"
)

type matchResult struct {
	matched         bool
	matchedOn       []string
	rejectionReason string
}

func evaluateMatch(
	profile *modeprofile.Profile,
	req *services.ResolveModeProfileRequest,
	at int64,
) matchResult {
	if !profile.IsEffectiveAt(at) {
		return matchResult{rejectionReason: rejectionNotEffective}
	}

	matchedOn := make([]string, 0, 4)

	if profile.IsOrgDefault {
		matchedOn = append(matchedOn, modeprofile.MatchOrgDefault)
	}

	if profile.CustomerID != nil && !profile.CustomerID.IsNil() {
		if *profile.CustomerID != req.CustomerID {
			return matchResult{rejectionReason: rejectionCustomer}
		}
		matchedOn = append(matchedOn, modeprofile.MatchCustomer)
	}

	if len(profile.ShipmentTypeIDs) > 0 {
		if !slices.Contains(profile.ShipmentTypeIDs, req.ShipmentTypeID) {
			return matchResult{rejectionReason: rejectionShipmentType}
		}
		matchedOn = append(matchedOn, modeprofile.MatchShipmentType)
	}

	if len(profile.ServiceTypeIDs) > 0 {
		if !slices.Contains(profile.ServiceTypeIDs, req.ServiceTypeID) {
			return matchResult{rejectionReason: rejectionServiceType}
		}
		matchedOn = append(matchedOn, modeprofile.MatchServiceType)
	}

	if len(profile.EquipmentTypeIDs) > 0 {
		if !matchesEquipment(profile.EquipmentTypeIDs, req) {
			return matchResult{rejectionReason: rejectionEquipmentType}
		}
		matchedOn = append(matchedOn, modeprofile.MatchEquipmentTyp)
	}

	return matchResult{matched: true, matchedOn: matchedOn}
}

func matchesEquipment(
	equipmentTypeIDs []pulid.ID,
	req *services.ResolveModeProfileRequest,
) bool {
	if req.TrailerTypeID.IsNotNil() && slices.Contains(equipmentTypeIDs, req.TrailerTypeID) {
		return true
	}
	if req.TractorTypeID.IsNotNil() && slices.Contains(equipmentTypeIDs, req.TractorTypeID) {
		return true
	}
	return false
}

func (s *service) resolveFromCandidates(
	candidates []*modeprofile.Profile,
	req *services.ResolveModeProfileRequest,
	at int64,
) *modeprofile.ResolvedPolicy {
	explained := make([]modeprofile.ProfileCandidate, 0, len(candidates))

	var (
		selected      *modeprofile.Profile
		selectedMatch matchResult
	)

	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}

		result := evaluateMatch(candidate, req, at)

		entry := modeprofile.ProfileCandidate{
			ProfileID:        candidate.ID,
			ProfileCode:      candidate.Code,
			ProfileName:      candidate.Name,
			Priority:         candidate.Priority,
			SpecificityScore: candidate.SpecificityScore,
			MatchedOn:        result.matchedOn,
		}

		switch {
		case !result.matched:
			entry.RejectionReason = result.rejectionReason
		case selected == nil:
			entry.Selected = true
			selected = candidate
			selectedMatch = result
		default:
			entry.RejectionReason = rejectionOutranked
		}

		explained = append(explained, entry)
	}

	if selected == nil {
		return nil
	}

	return buildResolvedPolicy(selected, selectedMatch, explained, at)
}

func buildResolvedPolicy(
	profile *modeprofile.Profile,
	match matchResult,
	candidates []modeprofile.ProfileCandidate,
	at int64,
) *modeprofile.ResolvedPolicy {
	configured := make(map[modeprofile.RuleKey]*modeprofile.CapabilityRule, len(profile.Rules))
	for _, rule := range profile.Rules {
		if rule != nil {
			configured[rule.RuleKey] = rule
		}
	}

	rules := make(map[modeprofile.RuleKey]modeprofile.ResolvedRule, len(configured))

	defs := modeprofile.AllRuleDefinitions()
	for i := range defs {
		def := &defs[i]

		if !profile.HasCapability(def.Capability) {
			continue
		}

		resolved := modeprofile.ResolvedRule{
			Key:         def.Key,
			Capability:  def.Capability,
			Label:       def.Label,
			Enforcement: def.DefaultEnforcement,
			Enabled:     true,
			Fields:      def.Fields,
			Parameters:  defaultParameters(def),
		}
		resolved.RequiredFields = requiredFieldsFor(def, resolved.Parameters)

		provenance := modeprofile.Provenance{
			ProfileID:          profile.ID,
			ProfileCode:        profile.Code,
			ProfileName:        profile.Name,
			IsOrgDefault:       profile.IsOrgDefault,
			Priority:           profile.Priority,
			SpecificityScore:   profile.SpecificityScore,
			MatchedOn:          match.matchedOn,
			RuleKey:            def.Key,
			Capability:         def.Capability,
			CapabilityLabel:    def.Capability.Label(),
			Enforcement:        def.DefaultEnforcement,
			DefaultEnforcement: def.DefaultEnforcement,
			Rationale:          def.Rationale,
		}

		if rule, ok := configured[def.Key]; ok {
			resolved.Enforcement = rule.Enforcement
			resolved.Enabled = rule.Enabled
			resolved.Parameters = mergeParameters(def, rule.Parameters)
			resolved.RequiredFields = requiredFieldsFor(def, resolved.Parameters)

			provenance.Enforcement = rule.Enforcement
			provenance.Overridden = rule.Enforcement != def.DefaultEnforcement
			provenance.OverrideReason = rule.OverrideReason
		}

		resolved.Provenance = provenance
		rules[def.Key] = resolved
	}

	return &modeprofile.ResolvedPolicy{
		ProfileID:      profile.ID,
		ProfileCode:    profile.Code,
		ProfileName:    profile.Name,
		ServiceModel:   profile.ServiceModel,
		EquipmentClass: profile.EquipmentClass,
		ExecutionParty: profile.ExecutionParty,
		Capabilities:   profile.Capabilities,
		Rules:          rules,
		Candidates:     candidates,
		ResolvedAt:     at,
	}
}

// requiredFieldsFor narrows a definition's RequiredFields using the parameters
// the profile actually resolved to.
//
// The narrowing exists because a requirement can be switched off by a parameter
// without disabling the rule. Temperature range always demands a minimum, but
// only demands a maximum when requireBothBounds is on — and validateTemperature
// enforces exactly that. Reporting both unconditionally would mark a field
// required in the form that the server would happily accept as empty.
func requiredFieldsFor(
	def *modeprofile.RuleDefinition,
	parameters map[string]any,
) []string {
	if len(def.RequiredFields) == 0 {
		return nil
	}

	if def.Key != modeprofile.RuleKeyTemperatureRange {
		return def.RequiredFields
	}

	requireBoth, _ := parameters[modeprofile.ParamRequireBothBounds].(bool)
	if requireBoth {
		return def.RequiredFields
	}

	return slices.DeleteFunc(
		slices.Clone(def.RequiredFields),
		func(field string) bool { return field == "temperatureMax" },
	)
}

func defaultParameters(def *modeprofile.RuleDefinition) map[string]any {
	if len(def.Parameters) == 0 {
		return nil
	}

	params := make(map[string]any, len(def.Parameters))
	for _, param := range def.Parameters {
		if param.Default != nil {
			params[param.Name] = param.Default
		}
	}

	if len(params) == 0 {
		return nil
	}

	return params
}

func mergeParameters(
	def *modeprofile.RuleDefinition,
	configured map[string]any,
) map[string]any {
	merged := defaultParameters(def)
	if len(configured) == 0 {
		return merged
	}

	if merged == nil {
		merged = make(map[string]any, len(configured))
	}

	for name, value := range configured {
		if value != nil {
			merged[name] = value
		}
	}

	return merged
}

func sortCandidates(candidates []*modeprofile.Profile) {
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]

		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}
		if left.SpecificityScore != right.SpecificityScore {
			return left.SpecificityScore > right.SpecificityScore
		}

		return left.CreatedAt < right.CreatedAt
	})
}

func (s *service) loadActiveProfiles(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*modeprofile.Profile, error) {
	if s.cacheRepo != nil {
		cached, err := s.cacheRepo.GetActiveProfiles(ctx, tenantInfo)
		if err == nil {
			return cached, nil
		}
	}

	profiles, err := s.profileRepo.GetActiveProfiles(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	if s.cacheRepo != nil {
		if cacheErr := s.cacheRepo.SetActiveProfiles(
			ctx, tenantInfo, profiles,
		); cacheErr != nil {
			s.l.Warn("failed to populate mode profile cache", zap.Error(cacheErr))
		}
	}

	return profiles, nil
}

func (s *service) Resolve(
	ctx context.Context,
	req *services.ResolveModeProfileRequest,
) (*modeprofile.ResolvedPolicy, error) {
	at := req.At
	if at == 0 {
		at = timeutils.NowUnix()
	}

	candidates, err := s.loadActiveProfiles(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return nil, services.ErrNoModeProfileConfigured
	}

	sortCandidates(candidates)

	return s.resolveFromCandidates(candidates, req, at), nil
}

func (s *service) InvalidateCache(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	if s.cacheRepo == nil {
		return nil
	}

	return s.cacheRepo.Invalidate(ctx, tenantInfo)
}
