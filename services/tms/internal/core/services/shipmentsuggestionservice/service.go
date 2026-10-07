package shipmentsuggestionservice

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipmentbrief"
	"github.com/emoss08/trenova/internal/core/domain/shipmentsuggestion"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentboardbrief"
	"github.com/emoss08/trenova/internal/core/services/shipmentboardcache"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	shiftWindow       = 12 * time.Hour
	decisionRetention = 7 * 24 * time.Hour
	maxQueueItems     = 20
)

type detentionDesk interface {
	ListDesk(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*detentionservice.DeskEntry, error)
}

type Params struct {
	fx.In

	Console         services.DispatchConsoleService
	Capabilities    services.ShipmentBoardCapabilitiesReader
	Watchlist       services.ShipmentWatchlistReader
	Comments        services.ShipmentCommentService
	Detention       *detentionservice.Service
	Decisions       repositories.ShipmentSuggestionDecisionRepository
	AssignmentReads repositories.AssignmentRepository
	Tenders         repositories.TenderRepository
	TenderGuard     services.TenderGuard
	Assignments     services.AssignmentService
	Cache           repositories.ShipmentBoardCache `optional:"true"`
	Briefs          repositories.ShipmentBriefRepository
	Organizations   repositories.OrganizationCacheRepository
	Logger          *zap.Logger
}

type Dependencies struct {
	Console         services.DispatchConsoleService
	Capabilities    services.ShipmentBoardCapabilitiesReader
	Watchlist       services.ShipmentWatchlistReader
	Comments        services.ShipmentCommentService
	Detention       detentionDesk
	Decisions       repositories.ShipmentSuggestionDecisionRepository
	AssignmentReads repositories.AssignmentRepository
	Tenders         repositories.TenderRepository
	TenderGuard     services.TenderGuard
	Assignments     services.AssignmentService
	Cache           repositories.ShipmentBoardCache
	Briefs          repositories.ShipmentBriefRepository
	Organizations   repositories.OrganizationCacheRepository
	Logger          *zap.Logger
	Now             func() time.Time
}

type Service struct {
	console         services.DispatchConsoleService
	capabilities    services.ShipmentBoardCapabilitiesReader
	watchlist       services.ShipmentWatchlistReader
	comments        services.ShipmentCommentService
	detention       detentionDesk
	decisions       repositories.ShipmentSuggestionDecisionRepository
	assignmentReads repositories.AssignmentRepository
	tenders         repositories.TenderRepository
	tenderGuard     services.TenderGuard
	assignments     services.AssignmentService
	cache           repositories.ShipmentBoardCache
	briefs          repositories.ShipmentBriefRepository
	organizations   repositories.OrganizationCacheRepository
	rules           []rule
	l               *zap.Logger
	now             func() time.Time
}

var (
	_ services.ShipmentSuggestionReader     = (*Service)(nil)
	_ services.ShipmentSuggestionDecider    = (*Service)(nil)
	_ services.ShipmentSuggestionCandidates = (*Service)(nil)
)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return NewWithDependencies(&Dependencies{
		Console:         p.Console,
		Capabilities:    p.Capabilities,
		Watchlist:       p.Watchlist,
		Comments:        p.Comments,
		Detention:       p.Detention,
		Decisions:       p.Decisions,
		AssignmentReads: p.AssignmentReads,
		Tenders:         p.Tenders,
		TenderGuard:     p.TenderGuard,
		Assignments:     p.Assignments,
		Cache:           p.Cache,
		Briefs:          p.Briefs,
		Organizations:   p.Organizations,
		Logger:          p.Logger,
	})
}

func NewWithDependencies(d *Dependencies) *Service {
	now := d.Now
	if now == nil {
		now = time.Now
	}
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{
		console:         d.Console,
		capabilities:    d.Capabilities,
		watchlist:       d.Watchlist,
		comments:        d.Comments,
		detention:       d.Detention,
		decisions:       d.Decisions,
		assignmentReads: d.AssignmentReads,
		tenders:         d.Tenders,
		tenderGuard:     d.TenderGuard,
		assignments:     d.Assignments,
		cache:           d.Cache,
		briefs:          d.Briefs,
		organizations:   d.Organizations,
		rules:           defaultRules(),
		l:               logger.Named("service.shipment-suggestions"),
		now:             now,
	}
}

func NewReader(s *Service) services.ShipmentSuggestionReader { return s }

func NewDecider(s *Service) services.ShipmentSuggestionDecider { return s }

func NewCandidates(s *Service) services.ShipmentSuggestionCandidates { return s }

type queue struct {
	Items []*services.ShipmentSuggestion `json:"items"`
}

func (s *Service) Suggestions(
	ctx context.Context,
	req *services.SuggestionRequest,
) (*services.ShipmentSuggestionQueue, error) {
	base, err := s.queue(ctx, req.TenantInfo, req.Timezone)
	if err != nil {
		return nil, err
	}

	items, narrated := s.applyBriefWording(ctx, req.TenantInfo, req.Timezone, base.Items)

	now := s.now()
	decided, err := s.decisions.ListSince(ctx, &repositories.ListSuggestionDecisionsRequest{
		TenantInfo: req.TenantInfo,
		UserID:     req.UserID,
		Since:      now.Add(-shiftWindow).Unix(),
	})
	if err != nil {
		return nil, err
	}

	out := applyDecisions(items, decided)
	out.Narrated = narrated

	return out, nil
}

func (s *Service) Candidates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timezone string,
) ([]*services.ShipmentSuggestion, error) {
	base, err := s.queue(ctx, tenantInfo, timezone)
	if err != nil {
		return nil, err
	}

	return copyItems(base.Items), nil
}

func (s *Service) queue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timezone string,
) (*queue, error) {
	if !timeutils.IsValidLocation(timezone) {
		return nil, errortypes.NewValidationError(
			"timezone",
			errortypes.ErrInvalid,
			"A valid IANA time zone is required",
		)
	}
	loc := timeutils.LoadLocation(timezone)

	return shipmentboardcache.GetOrCompute(
		ctx,
		s.cache,
		s.l,
		&shipmentboardcache.Request{
			Section:    repositories.ShipmentBoardSectionSuggestions,
			TenantInfo: tenantInfo,
			Timezone:   timezone,
		},
		func(ctx context.Context) (*queue, error) {
			return s.compute(ctx, tenantInfo, timezone, loc)
		},
	)
}

func (s *Service) applyBriefWording(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timezone string,
	items []*services.ShipmentSuggestion,
) ([]*services.ShipmentSuggestion, bool) {
	copied := copyItems(items)
	if s.briefs == nil || s.organizations == nil || len(copied) == 0 {
		return copied, false
	}

	day, err := shipmentboardbrief.Today(ctx, s.organizations, tenantInfo)
	if err != nil {
		s.l.Warn("could not resolve the brief's day", zap.Error(err))
		return copied, false
	}
	brief, err := shipmentboardbrief.Latest(ctx, s.briefs, tenantInfo, day)
	if err != nil {
		s.l.Warn("could not read the shipment brief", zap.Error(err))
		return copied, false
	}
	if brief == nil || brief.Facts.Timezone != timezone {
		return copied, false
	}

	return copied, ApplyWording(copied, brief.Wording)
}

func ApplyWording(
	items []*services.ShipmentSuggestion,
	wording map[string]shipmentbrief.Wording,
) bool {
	applied := false
	for _, item := range items {
		worded, ok := wording[item.Key]
		if !ok || !worded.Fits(item.Title, item.Reason, item.Impact) {
			continue
		}
		item.Title = worded.Title
		item.Reason = worded.Reason
		applied = true
	}

	return applied
}

func copyItems(items []*services.ShipmentSuggestion) []*services.ShipmentSuggestion {
	out := make([]*services.ShipmentSuggestion, 0, len(items))
	for _, item := range items {
		copied := *item
		out = append(out, &copied)
	}

	return out
}

func applyDecisions(
	items []*services.ShipmentSuggestion,
	decided []*shipmentsuggestion.DecisionRecord,
) *services.ShipmentSuggestionQueue {
	byKey := make(map[string]shipmentsuggestion.Decision, len(decided))
	handled := 0
	for _, record := range decided {
		byKey[record.SuggestionKey] = record.Decision
		if record.Decision == shipmentsuggestion.DecisionDone {
			handled++
		}
	}

	open := make([]*services.ShipmentSuggestion, 0, len(items))
	for _, item := range items {
		decision, ok := byKey[item.Key]
		if ok && decision == shipmentsuggestion.DecisionDone {
			continue
		}
		item.Deferred = ok && decision == shipmentsuggestion.DecisionLater
		open = append(open, item)
	}
	slices.SortStableFunc(open, func(a, b *services.ShipmentSuggestion) int {
		return cmp.Compare(boolRank(a.Deferred), boolRank(b.Deferred))
	})

	return &services.ShipmentSuggestionQueue{
		Items:            open,
		HandledThisShift: handled,
	}
}

func boolRank(value bool) int {
	if value {
		return 1
	}

	return 0
}

func (s *Service) Decide(ctx context.Context, req *services.DecideSuggestionRequest) error {
	if _, err := shipmentsuggestion.ParseKey(req.Key); err != nil {
		return errortypes.NewValidationError("key", errortypes.ErrInvalid, err.Error())
	}
	if !req.Decision.IsValid() {
		return errortypes.NewValidationError(
			"decision",
			errortypes.ErrInvalid,
			"Decision must be Done or Later",
		)
	}

	now := s.now()
	if _, err := s.decisions.Upsert(ctx, &shipmentsuggestion.DecisionRecord{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		UserID:         req.UserID,
		SuggestionKey:  req.Key,
		Decision:       req.Decision,
		DecidedAt:      now.Unix(),
	}); err != nil {
		return err
	}

	if err := s.decisions.Prune(ctx, &repositories.PruneSuggestionDecisionsRequest{
		TenantInfo: req.TenantInfo,
		UserID:     req.UserID,
		Before:     now.Add(-decisionRetention).Unix(),
	}); err != nil {
		s.l.Warn("failed to prune old suggestion decisions", zap.Error(err))
	}

	return nil
}

func (s *Service) Undo(ctx context.Context, req *services.UndoSuggestionRequest) error {
	key, err := shipmentsuggestion.ParseKey(req.Key)
	if err != nil {
		return errortypes.NewValidationError("key", errortypes.ErrInvalid, err.Error())
	}

	removed, err := s.decisions.Delete(ctx, &repositories.DeleteSuggestionDecisionRequest{
		TenantInfo: req.TenantInfo,
		UserID:     req.UserID,
		Key:        req.Key,
	})
	if err != nil || removed == nil {
		return err
	}
	if removed.Decision != shipmentsuggestion.DecisionDone {
		return nil
	}

	return s.revert(ctx, req.TenantInfo, key, removed.DecidedAt)
}
