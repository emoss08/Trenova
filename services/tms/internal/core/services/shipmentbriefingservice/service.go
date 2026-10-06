package shipmentbriefingservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmentboardcache"
	"github.com/emoss08/trenova/internal/core/services/shipmentnarration"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	maxOutputTokens   = 400
	maxNarrationChars = 400
	maxSegments       = 12
	schemaName        = "shipment_board_briefing"
)

var factFilters = [...]shipment.QuickFilter{
	shipment.QuickFilterDeliveringToday,
	shipment.QuickFilterMoving,
	shipment.QuickFilterLate,
	shipment.QuickFilterUncovered,
}

type Params struct {
	fx.In

	Board         repositories.ShipmentBoardRepository
	Briefing      repositories.ShipmentBriefingRepository
	QuickFilters  services.ShipmentQuickFilterBasisResolver
	Organizations repositories.OrganizationRepository
	Completion    services.CompletionService
	Cache         repositories.ShipmentBoardCache
	Logger        *zap.Logger
}

type Dependencies struct {
	Board         repositories.ShipmentBoardRepository
	Briefing      repositories.ShipmentBriefingRepository
	QuickFilters  services.ShipmentQuickFilterBasisResolver
	Organizations repositories.OrganizationRepository
	Completion    services.CompletionService
	Cache         repositories.ShipmentBoardCache
	Logger        *zap.Logger
	Now           func() time.Time
}

type Service struct {
	board         repositories.ShipmentBoardRepository
	briefing      repositories.ShipmentBriefingRepository
	quickFilters  services.ShipmentQuickFilterBasisResolver
	organizations repositories.OrganizationRepository
	completion    services.CompletionService
	cache         repositories.ShipmentBoardCache
	l             *zap.Logger
	now           func() time.Time
}

var _ services.ShipmentBriefingReader = (*Service)(nil)

func New(p Params) services.ShipmentBriefingReader {
	return NewWithDependencies(&Dependencies{
		Board:         p.Board,
		Briefing:      p.Briefing,
		QuickFilters:  p.QuickFilters,
		Organizations: p.Organizations,
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
		quickFilters:  d.QuickFilters,
		organizations: d.Organizations,
		completion:    d.Completion,
		cache:         d.Cache,
		l:             d.Logger.Named("service.shipment-briefing"),
		now:           d.Now,
	}
}

func (s *Service) Briefing(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timezone string,
) (*services.ShipmentBriefing, error) {
	return shipmentboardcache.GetOrCompute(
		ctx,
		s.cache,
		s.l,
		shipmentboardcache.Request{
			Section:    repositories.ShipmentBoardSectionBriefing,
			TenantInfo: tenantInfo,
			Timezone:   timezone,
		},
		func(ctx context.Context) (*services.ShipmentBriefing, error) {
			facts, err := s.Facts(ctx, tenantInfo, timezone)
			if err != nil {
				return nil, err
			}

			briefing := &services.ShipmentBriefing{
				Segments:    Deterministic(*facts),
				GeneratedAt: s.now().Unix(),
			}
			if narrated, ok := s.narrate(ctx, tenantInfo, facts, briefing.Segments); ok {
				briefing.Segments = narrated
				briefing.Narrated = true
			}

			return briefing, nil
		},
	)
}

func (s *Service) Facts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timezone string,
) (*Facts, error) {
	basis, err := s.quickFilters.Resolve(ctx, &services.ResolveShipmentQuickFilterBasisRequest{
		TenantInfo: tenantInfo,
		Timezone:   timezone,
	})
	if err != nil {
		return nil, err
	}

	specs := make([]shipment.QuickFilterSpec, 0, len(factFilters))
	for _, filter := range factFilters {
		specs = append(specs, shipment.Quick(filter))
	}

	facts := new(Facts)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		totals, tErr := s.board.QuickFilterTotals(
			gctx,
			&repositories.CountShipmentQuickFiltersRequest{
				Scope: &repositories.ShipmentBoardScope{
					Filter:  &pagination.QueryOptions{TenantInfo: tenantInfo},
					Options: repositories.ShipmentOptions{QuickFilterBasis: basis},
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
			repositories.GetOrganizationCapabilitiesRequest{TenantInfo: tenantInfo},
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
	plain []services.ShipmentBriefingSegment,
) ([]services.ShipmentBriefingSegment, bool) {
	allowed := facts.Filters()
	draft, ok := shipmentnarration.Narrate[narrationDraft](
		ctx,
		s.completion,
		s.l,
		&services.StructuredCompletionRequest{
			TenantInfo:   tenantInfo,
			Task:         aiprovider.TaskDailyBriefing,
			System:       systemPrompt,
			Context:      buildContext(facts, plain),
			OutputSchema: outputSchema(allowed),
			SchemaName:   schemaName,
			MaxTokens:    maxOutputTokens,
			Attribution:  services.AIUsageAttribution{UserID: tenantInfo.UserID},
		},
	)
	if !ok {
		return nil, false
	}

	return AcceptNarration(draft.toSegments(), facts, allowed)
}

func (d *narrationDraft) toSegments() []services.ShipmentBriefingSegment {
	out := make([]services.ShipmentBriefingSegment, 0, len(d.Segments))
	for _, segment := range d.Segments {
		item := services.ShipmentBriefingSegment{Text: segment.Text}
		if segment.Filter != "" {
			filter := shipment.QuickFilter(segment.Filter)
			item.Filter = &filter
		}
		out = append(out, item)
	}
	return out
}

func AcceptNarration(
	segments []services.ShipmentBriefingSegment,
	facts *Facts,
	allowed []shipment.QuickFilter,
) ([]services.ShipmentBriefingSegment, bool) {
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
