package carrierintelservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	defaultSourcingLimit = 25
	maxSourcingLimit     = 100
	minAutocompleteChars = 2
	maxAutocompleteLimit = 15
)

type SourcingQuery struct {
	TenantInfo             pagination.TenantInfo
	Text                   string
	State                  string
	OriginState            string
	DestinationState       string
	MinPowerUnits          *int
	MaxPowerUnits          *int
	MinAuthorityAgeDays    *int
	HazmatOnly             bool
	ExcludeBlocking        bool
	ExcludeExistingCarrier bool
	Limit                  int
	Offset                 int
}

type SourcingResult struct {
	Profile           *carrierintel.Profile
	ProviderRef       string
	DOTNumber         string
	LegalName         string
	ExistingCarrierID pulid.ID
	LaneMatches       int
	Findings          []carrierintel.Finding
	RiskLevel         carrierintel.RiskLevel
	Score             float64
}

type SourcingPage struct {
	Items    []*SourcingResult
	Total    int
	Provider string
}

func (s *Service) searchCapable(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*boundProvider, services.CarrierIntelSearcher, *carrierintel.CarrierIntelControl, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, nil, nil, err
	}
	bound, err := s.resolvePrimary(ctx, tenantInfo, control)
	if err != nil {
		return nil, nil, nil, err
	}
	searcher, ok := bound.client.(services.CarrierIntelSearcher)
	if !ok || !bound.capabilities().Has(carrierintel.CapabilitySearch) {
		return nil, nil, nil, errortypes.NewBusinessError(
			"{0} does not support carrier search", bound.provider.String(),
		)
	}
	return bound, searcher, control, nil
}

func (s *Service) SearchCarriers(ctx context.Context, query *SourcingQuery) (*SourcingPage, error) {
	text := strings.TrimSpace(query.Text)
	if text == "" && query.State == "" && query.OriginState == "" && query.DestinationState == "" {
		return nil, errortypes.NewValidationError("text", errortypes.ErrRequired,
			"Enter a name, EIN, VIN or choose a state to search")
	}
	limit := query.Limit
	if limit <= 0 {
		limit = defaultSourcingLimit
	}
	limit = min(limit, maxSourcingLimit)

	ctx = carrierintel.WithPurpose(ctx, carrierintel.PurposeSourcing)
	bound, searcher, control, err := s.searchCapable(ctx, query.TenantInfo)
	if err != nil {
		return nil, err
	}
	if err = s.guardBudget(ctx, &budgetCheck{
		tenant:   query.TenantInfo,
		control:  control,
		bound:    bound,
		endpoint: carrierintel.EndpointSearch,
	}); err != nil {
		return nil, err
	}

	request := &services.CarrierIntelSearchRequest{
		State:  firstNonEmpty(query.State, query.OriginState),
		Limit:  limit,
		Offset: max(query.Offset, 0),
	}
	switch {
	case isVINLike(text):
		request.VIN = strings.ToUpper(text)
	case isEINLike(text):
		request.EIN = stringutils.DigitsOnly(text)
	default:
		request.Query = text
	}

	var result *services.CarrierIntelSearchResult
	err = s.withRateLimitWait(ctx, func() error {
		var searchErr error
		result, searchErr = searcher.Search(ctx, request)
		return searchErr
	})
	if err != nil {
		return nil, toBusinessError(bound.provider, err)
	}

	dots := make([]string, 0, len(result.Items))
	for _, hit := range result.Items {
		if dot := hit.Profile.DOTNumber(); dot != "" {
			dots = append(dots, dot)
		}
	}
	existing, err := s.subjectRepo.ListExistingCarrierDOTs(ctx, query.TenantInfo, dots)
	if err != nil {
		return nil, err
	}

	now := s.now()
	items := make([]*SourcingResult, 0, len(result.Items))
	for _, hit := range result.Items {
		profile := hit.Profile
		if profile == nil {
			continue
		}
		profile.NormalizeCoverage()
		dot := profile.DOTNumber()
		entry := &SourcingResult{
			Profile:           profile,
			ProviderRef:       hit.ProviderRef,
			DOTNumber:         dot,
			ExistingCarrierID: existing[dot],
		}
		if profile.Identity != nil {
			entry.LegalName = profile.Identity.LegalName
		}
		if query.ExcludeExistingCarrier && entry.ExistingCarrierID.IsNotNil() {
			continue
		}
		if !matchesSourcingFilters(profile, query) {
			continue
		}
		entry.Findings = carrierintel.EvaluateFindings(&carrierintel.EvaluateInput{
			Profile:  profile,
			Now:      now,
			Subject:  carrierintel.SubjectTypeProspect,
			Settings: control.Rules,
		})
		if query.ExcludeBlocking && len(carrierintel.BlockingCodes(entry.Findings)) > 0 {
			continue
		}
		entry.RiskLevel = profile.DeriveRiskLevel(entry.Findings)
		entry.LaneMatches = laneMatches(profile, query.OriginState, query.DestinationState)
		entry.Score = sourcingScore(entry)
		items = append(items, entry)
	}

	slices.SortStableFunc(items, func(a, b *SourcingResult) int {
		switch {
		case a.Score > b.Score:
			return -1
		case a.Score < b.Score:
			return 1
		default:
			return 0
		}
	})

	return &SourcingPage{Items: items, Total: result.Total, Provider: bound.provider.String()}, nil
}

func isVINLike(text string) bool {
	return len(text) == 17 && strings.IndexFunc(text, func(r rune) bool {
		return (r < '0' || r > '9') && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z')
	}) == -1
}

func isEINLike(text string) bool {
	digits := stringutils.DigitsOnly(text)
	return len(digits) == 9 && len(strings.TrimSpace(text)) <= 10
}

func matchesSourcingFilters(profile *carrierintel.Profile, query *SourcingQuery) bool {
	if query.MinPowerUnits != nil || query.MaxPowerUnits != nil {
		if profile.Fleet == nil || profile.Fleet.PowerUnits == nil {
			return false
		}
		units := *profile.Fleet.PowerUnits
		if query.MinPowerUnits != nil && units < *query.MinPowerUnits {
			return false
		}
		if query.MaxPowerUnits != nil && units > *query.MaxPowerUnits {
			return false
		}
	}
	if query.MinAuthorityAgeDays != nil {
		age := profile.Authority.OldestActiveAgeDays()
		if age == nil || *age < *query.MinAuthorityAgeDays {
			return false
		}
	}
	if query.HazmatOnly {
		if profile.Operations == nil || profile.Operations.HazmatCarrier == nil ||
			!*profile.Operations.HazmatCarrier {
			return false
		}
	}
	return true
}

func laneMatches(profile *carrierintel.Profile, origin, destination string) int {
	if profile.Lanes == nil || (origin == "" && destination == "") {
		return 0
	}
	matches := 0
	for _, lane := range profile.Lanes.Preferred {
		if origin != "" && !strings.EqualFold(lane.OriginState, origin) {
			continue
		}
		if destination != "" && !strings.EqualFold(lane.DestinationState, destination) {
			continue
		}
		weight := 1
		if lane.Loads != nil && *lane.Loads > 0 {
			weight = *lane.Loads
		}
		matches += weight
	}
	return matches
}

func sourcingScore(entry *SourcingResult) float64 {
	score := 100.0
	score -= float64(entry.RiskLevel.Rank()) * 10
	for idx := range entry.Findings {
		finding := entry.Findings[idx]
		if finding.Unverifiable {
			continue
		}
		switch finding.Action {
		case carrierintel.RuleActionBlock:
			score -= 40
		case carrierintel.RuleActionWarn:
			score -= 8
		}
	}
	score += float64(min(entry.LaneMatches, 50))
	if entry.ExistingCarrierID.IsNotNil() {
		score += 5
	}
	return score
}

func (s *Service) AutocompleteCarriers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query string,
	limit int,
) ([]services.CarrierIntelSuggestion, error) {
	query = strings.TrimSpace(query)
	if len(query) < minAutocompleteChars {
		return []services.CarrierIntelSuggestion{}, nil
	}
	if limit <= 0 || limit > maxAutocompleteLimit {
		limit = maxAutocompleteLimit
	}
	ctx = carrierintel.WithPurpose(ctx, carrierintel.PurposeSourcing)
	bound, searcher, control, err := s.searchCapable(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !bound.capabilities().Has(carrierintel.CapabilityAutocomplete) {
		return []services.CarrierIntelSuggestion{}, nil
	}
	if err = s.guardBudget(ctx, &budgetCheck{
		tenant:   tenantInfo,
		control:  control,
		bound:    bound,
		endpoint: carrierintel.EndpointAutocomplete,
	}); err != nil {
		return nil, err
	}
	suggestions, err := searcher.Autocomplete(ctx, query, limit)
	if err != nil {
		return nil, toBusinessError(bound.provider, err)
	}
	return suggestions, nil
}

type ImportProspectRequest struct {
	TenantInfo       pagination.TenantInfo
	DOTNumber        string
	Code             string
	EnrollMonitoring bool
}

func (s *Service) ImportProspect(
	ctx context.Context,
	req *ImportProspectRequest,
) (*carrier.Carrier, error) {
	dot := stringutils.DigitsOnly(req.DOTNumber)
	if dot == "" {
		return nil, errortypes.NewValidationError("dotNumber", errortypes.ErrRequired,
			"A DOT number is required to import a carrier")
	}
	existing, err := s.subjectRepo.ListExistingCarrierDOTs(ctx, req.TenantInfo, []string{dot})
	if err != nil {
		return nil, err
	}
	if id, found := existing[dot]; found {
		return nil, errortypes.NewBusinessError(
			"A carrier with DOT {0} already exists", dot,
		).WithParam("carrierId", id.String())
	}

	fetched, _, err := s.LookupProspect(ctx, &LookupProspectRequest{
		TenantInfo: req.TenantInfo,
		DOTNumber:  dot,
		Depth:      carrierintel.LookupDepthFMCSA,
	})
	if err != nil {
		return nil, err
	}
	snapshot := fetched.Snapshot
	if snapshot.NotFound || snapshot.Profile == nil || snapshot.Profile.Identity == nil {
		return nil, errortypes.NewBusinessError(
			"The provider has no record for DOT {0}", dot,
		)
	}

	entity := buildProspectCarrier(snapshot.Profile, req.Code)
	entity.OrganizationID = req.TenantInfo.OrgID
	entity.BusinessUnitID = req.TenantInfo.BuID

	created, err := s.carrierService.Create(ctx, entity, &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    req.TenantInfo.UserID,
		UserID:         req.TenantInfo.UserID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	})
	if err != nil {
		return nil, err
	}

	subject := carrierSubject(created)
	if _, fetchErr := s.Fetch(ctx, &FetchRequest{
		TenantInfo: req.TenantInfo,
		Subject:    subject,
		MinDepth:   carrierintel.LookupDepthFMCSA,
		Purpose:    carrierintel.PurposeVet,
		Source:     carrierintel.EventSourceSnapshotDiff,
	}); fetchErr != nil {
		s.l.Warn("failed to attach intelligence to imported carrier", zap.Error(fetchErr))
	}

	if req.EnrollMonitoring {
		if _, enrollErr := s.SetManualEnrollment(ctx, &EnrollSubjectsRequest{
			TenantInfo: req.TenantInfo,
			CarrierIDs: []pulid.ID{created.ID},
			Enroll:     true,
		}); enrollErr != nil {
			s.l.Warn("failed to enroll imported carrier", zap.Error(enrollErr))
		}
	}
	return created, nil
}

func buildProspectCarrier(profile *carrierintel.Profile, code string) *carrier.Carrier {
	identity := profile.Identity
	name := stringutils.TruncateRunes(strings.TrimSpace(identity.LegalName), 255)
	entity := &carrier.Carrier{
		Status:           carrier.StatusActive,
		ComplianceStatus: carrier.ComplianceStatusPending,
		CarrierType:      carrier.TypeCommon,
		Name:             name,
		DBAName:          stringutils.TruncateRunes(strings.TrimSpace(identity.DBAName), 255),
		DOTNumber:        stringutils.DigitsOnly(identity.DOTNumber),
		Code:             prospectCode(code, name, identity.DOTNumber),
		SafetyRating:     carrier.SafetyRatingNotRated,
		PaymentMethod:    carrier.PaymentMethodCheck,
		PaymentTermDays:  30,
	}
	if docket := stringutils.DigitsOnly(identity.DocketNumber); docket != "" &&
		(identity.DocketPrefix == "" || strings.EqualFold(identity.DocketPrefix, "MC")) {
		entity.MCNumber = stringutils.TruncateRunes(docket, 12)
	}
	if profile.Authority != nil && profile.Authority.Broker.IsActive() &&
		!profile.Authority.Common.IsActive() && !profile.Authority.Contract.IsActive() {
		entity.CarrierType = carrier.TypeBroker
	}
	if identity.PhysicalAddress != nil {
		entity.AddressLine1 = stringutils.TruncateRunes(identity.PhysicalAddress.Line1, 150)
		entity.City = stringutils.TruncateRunes(identity.PhysicalAddress.City, 100)
		entity.PostalCode = stringutils.TruncateRunes(identity.PhysicalAddress.PostalCode, 10)
	}
	if profile.Contacts != nil {
		entity.Phone = stringutils.TruncateRunes(stringutils.DigitsOnly(profile.Contacts.Phone), 20)
		email := strings.ToLower(strings.TrimSpace(profile.Contacts.Email))
		if strings.Contains(email, "@") {
			entity.Email = stringutils.TruncateRunes(email, 255)
		}
	}
	if identity.EIN != "" {
		ein := stringutils.DigitsOnly(identity.EIN)
		if len(ein) == 9 {
			einType := carrier.TaxIDTypeEIN
			entity.TaxID = ein
			entity.TaxIDType = &einType
		}
	}
	if profile.Safety != nil {
		entity.SafetyRating = mapCarrierSafetyRating(profile.Safety.Rating)
	}
	entity.Notes = "Imported from carrier intelligence sourcing"
	return entity
}

func mapCarrierSafetyRating(rating carrierintel.SafetyRating) carrier.SafetyRating {
	switch rating {
	case carrierintel.SafetyRatingSatisfactory:
		return carrier.SafetyRatingSatisfactory
	case carrierintel.SafetyRatingConditional:
		return carrier.SafetyRatingConditional
	case carrierintel.SafetyRatingUnsatisfactory:
		return carrier.SafetyRatingUnsatisfactory
	default:
		return carrier.SafetyRatingNotRated
	}
}

func prospectCode(requested, name, dot string) string {
	code := strings.ToUpper(strings.TrimSpace(requested))
	if code != "" {
		return stringutils.TruncateRunes(code, 10)
	}
	letters := make([]rune, 0, 4)
	for _, r := range strings.ToUpper(name) {
		if r >= 'A' && r <= 'Z' {
			letters = append(letters, r)
			if len(letters) == 4 {
				break
			}
		}
	}
	digits := stringutils.DigitsOnly(dot)
	if len(digits) > 6 {
		digits = digits[len(digits)-6:]
	}
	return stringutils.TruncateRunes(string(letters)+digits, 10)
}
