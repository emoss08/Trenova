package shipmentbriefingservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/shipmentboardbrief"
	"github.com/emoss08/trenova/internal/modules/agent/app/shipmentnarration"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/shipmentbrief"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/shipmentbriefports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentboardcache"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentsuggestionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	maxOutputTokens      = 600
	maxNarrationChars    = 500
	maxSegments          = shipmentbrief.MaxSegments
	maxGenerationsPerDay = 6
	schemaName           = "shipment_board_briefing"
	liveFactsVariant     = "live-facts"
)

var factFilters = [...]shipment.QuickFilter{
	shipment.QuickFilterDeliveringToday,
	shipment.QuickFilterMoving,
	shipment.QuickFilterLate,
	shipment.QuickFilterUncovered,
	shipment.QuickFilterDetention,
	shipment.QuickFilterReadyToBill,
}

type Params struct {
	fx.In

	Board         shipmentports.ShipmentBoardRepository
	Briefing      shipmentports.ShipmentBriefingRepository
	Briefs        shipmentbriefports.ShipmentBriefRepository
	QuickFilters  shipmentports.ShipmentQuickFilterBasisResolver
	Organizations tenantports.OrganizationRepository
	Suggestions   shipmentbriefports.ShipmentSuggestionCandidates
	Completion    agentports.CompletionService
	Cache         shipmentports.ShipmentBoardCache
	Logger        *zap.Logger
}

type Dependencies struct {
	Board         shipmentports.ShipmentBoardRepository
	Briefing      shipmentports.ShipmentBriefingRepository
	Briefs        shipmentbriefports.ShipmentBriefRepository
	QuickFilters  shipmentports.ShipmentQuickFilterBasisResolver
	Organizations tenantports.OrganizationRepository
	Suggestions   shipmentbriefports.ShipmentSuggestionCandidates
	Completion    agentports.CompletionService
	Cache         shipmentports.ShipmentBoardCache
	Logger        *zap.Logger
	Now           func() time.Time
}

type Service struct {
	board         shipmentports.ShipmentBoardRepository
	briefing      shipmentports.ShipmentBriefingRepository
	briefs        shipmentbriefports.ShipmentBriefRepository
	quickFilters  shipmentports.ShipmentQuickFilterBasisResolver
	organizations tenantports.OrganizationRepository
	suggestions   shipmentbriefports.ShipmentSuggestionCandidates
	completion    agentports.CompletionService
	cache         shipmentports.ShipmentBoardCache
	l             *zap.Logger
	now           func() time.Time
}

var (
	_ shipmentbriefports.ShipmentBriefingReader = (*Service)(nil)
	_ shipmentbriefports.ShipmentBriefWriter    = (*Service)(nil)
)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return NewWithDependencies(&Dependencies{
		Board:         p.Board,
		Briefing:      p.Briefing,
		Briefs:        p.Briefs,
		QuickFilters:  p.QuickFilters,
		Organizations: p.Organizations,
		Suggestions:   p.Suggestions,
		Completion:    p.Completion,
		Cache:         p.Cache,
		Logger:        p.Logger,
		Now:           time.Now,
	})
}

func NewWithDependencies(d *Dependencies) *Service {
	return &Service{
		board:         d.Board,
		briefing:      d.Briefing,
		briefs:        d.Briefs,
		quickFilters:  d.QuickFilters,
		organizations: d.Organizations,
		suggestions:   d.Suggestions,
		completion:    d.Completion,
		cache:         d.Cache,
		l:             d.Logger.Named("service.shipment-briefing"),
		now:           d.Now,
	}
}

func NewReader(s *Service) shipmentbriefports.ShipmentBriefingReader { return s }

func NewWriter(s *Service) shipmentbriefports.ShipmentBriefWriter { return s }

func (s *Service) Briefing(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	_ string,
) (*shipmentbriefports.ShipmentBriefing, error) {
	day, err := shipmentboardbrief.Today(ctx, s.organizations, tenantInfo)
	if err != nil {
		return nil, err
	}
	brief, err := shipmentboardbrief.Latest(ctx, s.briefs, tenantInfo, day)
	if err != nil {
		return nil, err
	}

	switch {
	case brief == nil:
		brief, err = s.write(ctx, tenantInfo, day, shipmentbrief.TriggerOnDemand, nil)
	case s.cleared(ctx, tenantInfo, day, brief):
		brief, err = s.write(ctx, tenantInfo, day, shipmentbrief.TriggerCleared, brief)
	}
	if err != nil {
		return nil, err
	}

	return toBriefing(brief), nil
}

func (s *Service) WriteBrief(
	ctx context.Context,
	req *shipmentbriefports.WriteShipmentBriefRequest,
) (*shipmentbrief.Brief, error) {
	day, err := shipmentboardbrief.Today(ctx, s.organizations, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	previous, err := shipmentboardbrief.Latest(ctx, s.briefs, req.TenantInfo, day)
	if err != nil {
		return nil, err
	}
	if previous != nil && req.Trigger == shipmentbrief.TriggerScheduled &&
		previous.Trigger == shipmentbrief.TriggerScheduled {
		return previous, nil
	}

	return s.write(ctx, req.TenantInfo, day, req.Trigger, previous)
}

// cleared reports whether a brief written while issues were open now faces a
// board with none, so the day earns a fresh brief. It is bounded per day, so a
// board that keeps flipping between clear and busy costs a handful of model
// calls rather than one per visit.
func (s *Service) cleared(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	day shipmentboardbrief.Day,
	brief *shipmentbrief.Brief,
) bool {
	if brief.OpenIssues == 0 || brief.Generation >= maxGenerationsPerDay {
		return false
	}

	live, err := s.liveFacts(ctx, tenantInfo, day.Timezone)
	if err != nil {
		s.l.Warn("could not read the board's live figures", zap.Error(err))
		return false
	}

	return live.OpenIssues() == 0
}

func (s *Service) liveFacts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timezone string,
) (*Facts, error) {
	return shipmentboardcache.GetOrCompute(
		ctx,
		s.cache,
		s.l,
		&shipmentboardcache.Request{
			Section:    shipmentports.ShipmentBoardSectionBriefing,
			TenantInfo: tenantInfo,
			Timezone:   timezone,
			Variant:    liveFactsVariant,
		},
		func(ctx context.Context) (*Facts, error) {
			facts, _, err := s.snapshot(ctx, tenantInfo, timezone)
			return facts, err
		},
	)
}

func (s *Service) write(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	day shipmentboardbrief.Day,
	trigger shipmentbrief.Trigger,
	previous *shipmentbrief.Brief,
) (*shipmentbrief.Brief, error) {
	facts, candidates, err := s.snapshot(ctx, tenantInfo, day.Timezone)
	if err != nil {
		return nil, err
	}

	segments := Deterministic(*facts)
	narrated, model := false, ""
	if worded, wordedBy, ok := s.narrate(ctx, tenantInfo, facts, segments); ok {
		segments, narrated, model = worded, true, wordedBy
	}

	generation := 1
	if previous != nil {
		generation = previous.Generation + 1
	}

	brief := &shipmentbrief.Brief{
		OrganizationID:  tenantInfo.OrgID,
		BusinessUnitID:  tenantInfo.BuID,
		BriefDate:       day.Date,
		Generation:      generation,
		Trigger:         trigger,
		Segments:        toStored(segments),
		Wording:         s.wordSuggestions(ctx, tenantInfo, candidates),
		Facts:           facts.Facts,
		Narrated:        narrated,
		ModelIdentifier: model,
		GeneratedAt:     s.now().Unix(),
	}
	brief.Normalize()

	multiErr := errortypes.NewMultiError()
	brief.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.briefs.Insert(ctx, brief)
}

func (s *Service) snapshot(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timezone string,
) (*Facts, []*shipmentsuggestionports.ShipmentSuggestion, error) {
	var (
		facts      *Facts
		candidates []*shipmentsuggestionports.ShipmentSuggestion
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		facts, err = s.Facts(gctx, tenantInfo, timezone)
		return err
	})
	g.Go(func() error {
		var err error
		candidates, err = s.suggestions.Candidates(gctx, tenantInfo, timezone)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	facts.OpenSuggestions = len(candidates)
	facts.Timezone = timezone
	return facts, candidates, nil
}

func toStored(segments []shipmentbriefports.ShipmentBriefingSegment) []shipmentbrief.Segment {
	out := make([]shipmentbrief.Segment, 0, len(segments))
	for _, segment := range segments {
		out = append(out, shipmentbrief.Segment{Text: segment.Text, Filter: segment.Filter})
	}
	return out
}

func toBriefing(brief *shipmentbrief.Brief) *shipmentbriefports.ShipmentBriefing {
	segments := make([]shipmentbriefports.ShipmentBriefingSegment, 0, len(brief.Segments))
	for _, segment := range brief.Segments {
		segments = append(segments, shipmentbriefports.ShipmentBriefingSegment{
			Text:   segment.Text,
			Filter: segment.Filter,
		})
	}

	return &shipmentbriefports.ShipmentBriefing{
		Segments:    segments,
		Narrated:    brief.Narrated,
		GeneratedAt: brief.GeneratedAt,
		Generation:  brief.Generation,
	}
}

func (s *Service) Facts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timezone string,
) (*Facts, error) {
	specs := make([]shipment.QuickFilterSpec, 0, len(factFilters))
	for _, filter := range factFilters {
		specs = append(specs, shipment.Quick(filter))
	}

	margin, detention := shipment.QuickFiltersNeed(specs)
	basis, err := s.quickFilters.Resolve(ctx, &shipmentports.ResolveShipmentQuickFilterBasisRequest{
		TenantInfo: tenantInfo,
		Timezone:   timezone,
		Margin:     margin,
		Detention:  detention,
	})
	if err != nil {
		return nil, err
	}

	facts := new(Facts)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		totals, tErr := s.board.QuickFilterTotals(
			gctx,
			&shipmentports.CountShipmentQuickFiltersRequest{
				Scope: &shipmentports.ShipmentBoardScope{
					Filter:  &pagination.QueryOptions{TenantInfo: tenantInfo},
					Options: shipmentports.ShipmentOptions{QuickFilterBasis: basis},
				},
				Filters: specs,
			},
		)
		if tErr != nil {
			return tErr
		}
		facts.DeliveringToday = totals[0].Count
		facts.Moving = totals[1].Count
		facts.Late = totals[2].Count
		facts.Uncovered = totals[3].Count
		facts.Detention = totals[4].Count
		facts.ReadyToBill = totals[5].Count
		return nil
	})
	g.Go(func() error {
		reason, rErr := s.briefing.LeadingLateReason(gctx, tenantInfo)
		if rErr != nil {
			return rErr
		}
		if reason != nil {
			facts.LateReason = reason.Label
		}
		return nil
	})
	g.Go(func() error {
		caps, cErr := s.organizations.GetCapabilities(
			gctx,
			tenantports.GetOrganizationCapabilitiesRequest{TenantInfo: tenantInfo},
		)
		if cErr != nil {
			return cErr
		}
		facts.OperationType = tenant.OperationTypeOf(
			caps.BrokerageEnabled,
			caps.AssetOperationsEnabled,
		)
		return nil
	})
	if err = g.Wait(); err != nil {
		return nil, err
	}

	if facts.Late == 0 {
		facts.LateReason = ""
	}

	return facts, nil
}

type narrationDraft struct {
	Segments []struct {
		Text   string `json:"text"`
		Filter string `json:"filter"`
	} `json:"segments"`
}

func (s *Service) narrate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	facts *Facts,
	plain []shipmentbriefports.ShipmentBriefingSegment,
) ([]shipmentbriefports.ShipmentBriefingSegment, string, bool) {
	allowed := facts.Filters()
	narration, ok := shipmentnarration.Narrate[narrationDraft](
		ctx,
		s.completion,
		s.l,
		&agentports.StructuredCompletionRequest{
			TenantInfo:   tenantInfo,
			Task:         aiprovider.TaskDailyBriefing,
			System:       systemPrompt,
			Context:      buildContext(facts, plain),
			OutputSchema: outputSchema(allowed),
			SchemaName:   schemaName,
			MaxTokens:    maxOutputTokens,
			Attribution:  aiusageports.AIUsageAttribution{UserID: tenantInfo.UserID},
		},
	)
	if !ok {
		return nil, "", false
	}

	segments, accepted := AcceptNarration(narration.Draft.toSegments(), facts, allowed)
	if !accepted {
		return nil, "", false
	}
	return segments, narration.Model, true
}

func (d *narrationDraft) toSegments() []shipmentbriefports.ShipmentBriefingSegment {
	out := make([]shipmentbriefports.ShipmentBriefingSegment, 0, len(d.Segments))
	for _, segment := range d.Segments {
		item := shipmentbriefports.ShipmentBriefingSegment{Text: segment.Text}
		if segment.Filter != "" {
			filter := shipment.QuickFilter(segment.Filter)
			item.Filter = &filter
		}
		out = append(out, item)
	}
	return out
}

func AcceptNarration(
	segments []shipmentbriefports.ShipmentBriefingSegment,
	facts *Facts,
	allowed []shipment.QuickFilter,
) ([]shipmentbriefports.ShipmentBriefingSegment, bool) {
	if len(segments) == 0 || len(segments) > maxSegments {
		return nil, false
	}

	var builder strings.Builder
	for _, segment := range segments {
		if segment.Filter != nil && !containsFilter(allowed, *segment.Filter) {
			return nil, false
		}
		builder.WriteString(segment.Text)
	}

	prose := strings.TrimSpace(builder.String())
	if prose == "" || len(prose) > maxNarrationChars {
		return nil, false
	}
	if !shipmentnarration.Supported(prose, shipmentnarration.IntValues(facts.Counts()...)) {
		return nil, false
	}

	return segments, true
}

func containsFilter(filters []shipment.QuickFilter, filter shipment.QuickFilter) bool {
	for _, candidate := range filters {
		if candidate == filter {
			return true
		}
	}
	return false
}
