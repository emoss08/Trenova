package shipmentsuggestionservice

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipmentsuggestion"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
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
	maxQueueItems     = 12
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
	Completion      services.CompletionService      `optional:"true"`
	Cache           repositories.ShipmentBoardCache `optional:"true"`
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
	Completion      services.CompletionService
	Cache           repositories.ShipmentBoardCache
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
	completion      services.CompletionService
	cache           repositories.ShipmentBoardCache
	rules           []rule
	l               *zap.Logger
	now             func() time.Time
}

var (
	_ services.ShipmentSuggestionReader  = (*Service)(nil)
	_ services.ShipmentSuggestionDecider = (*Service)(nil)
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
		Completion:      p.Completion,
		Cache:           p.Cache,
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
		completion:      d.Completion,
		cache:           d.Cache,
		rules:           defaultRules(),
		l:               logger.Named("service.shipment-suggestions"),
		now:             now,
	}
}

func NewReader(s *Service) services.ShipmentSuggestionReader { return s }

func NewDecider(s *Service) services.ShipmentSuggestionDecider { return s }

type queue struct {
	Items    []*services.ShipmentSuggestion `json:"items"`
	Narrated bool                           `json:"narrated"`
}

func (s *Service) Suggestions(
	ctx context.Context,
	req *services.SuggestionRequest,
) (*services.ShipmentSuggestionQueue, error) {
	if !timeutils.IsValidLocation(req.Timezone) {
		return nil, errortypes.NewValidationError(
			"timezone",
			errortypes.ErrInvalid,
			"A valid IANA time zone is required",
		)
	}
	loc := timeutils.LoadLocation(req.Timezone)

	base, err := shipmentboardcache.GetOrCompute(
		ctx,
		s.cache,
		s.l,
		&shipmentboardcache.Request{
			Section:    repositories.ShipmentBoardSectionSuggestions,
			TenantInfo: req.TenantInfo,
			Timezone:   req.Timezone,
		},
		func(ctx context.Context) (*queue, error) {
			return s.compute(ctx, req.TenantInfo, req.Timezone, loc)
		},
	)
	if err != nil {
		return nil, err
	}

	now := s.now()
	decided, err := s.decisions.ListSince(ctx, &repositories.ListSuggestionDecisionsRequest{
		TenantInfo: req.TenantInfo,
		UserID:     req.UserID,
		Since:      now.Add(-shiftWindow).Unix(),
	})
	if err != nil {
		return nil, err
	}

	return applyDecisions(base, decided), nil
}

func applyDecisions(
	base *queue,
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

	items := make([]*services.ShipmentSuggestion, 0, len(base.Items))
	for _, item := range base.Items {
		decision, ok := byKey[item.Key]
		if ok && decision == shipmentsuggestion.DecisionDone {
			continue
		}
		copied := *item
		copied.Deferred = ok && decision == shipmentsuggestion.DecisionLater
		items = append(items, &copied)
	}
	slices.SortStableFunc(items, func(a, b *services.ShipmentSuggestion) int {
		return cmp.Compare(boolRank(a.Deferred), boolRank(b.Deferred))
	})

	return &services.ShipmentSuggestionQueue{
		Items:            items,
		HandledThisShift: handled,
		Narrated:         base.Narrated,
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
