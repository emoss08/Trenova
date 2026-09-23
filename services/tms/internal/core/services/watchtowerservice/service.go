package watchtowerservice

import (
	"context"
	"fmt"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// RealtimeResource is what connected clients listen on for the feed.
	RealtimeResource = "watchtower"
	// FeedPath is where the feed lives; a notification's link opens it on
	// the item that raised the notification.
	FeedPath = "/desk/watchtower"

	defaultPageSize = 50
	maxPageSize     = 200
	// maxCriticalRecipients bounds who is told about a critical item. An
	// organization where hundreds may read shipments does not want hundreds
	// of notifications per weather warning.
	maxCriticalRecipients = 50
	notificationSource    = "watchtower"
	// EventCritical is the notification a critical item raises when it
	// first appears.
	EventCritical = "watchtower.critical"
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	Repo        repositories.WatchtowerRepository
	Projector   *Projector
	Permissions services.PermissionEngine
	Definitions repositories.AgentDefinitionRepository
	Runs        services.AgentRunService     `optional:"true"`
	Events      services.AgentEventPublisher `optional:"true"`
	Sources     []services.WatchtowerSource  `                group:"watchtower_sources"`
}

// Service is the watchtower's read face: what a reader may see of it, where
// their unseen line sits, and how an item is handed to an agent. It also
// owns the sweeps, which is why it knows the sources.
//
// The write face lives in Projector and is injected rather than embedded.
// See the note there for why the two cannot be one constructor.
type Service struct {
	l           *zap.Logger
	repo        repositories.WatchtowerRepository
	projector   *Projector
	permissions services.PermissionEngine
	definitions repositories.AgentDefinitionRepository
	runs        services.AgentRunService
	events      services.AgentEventPublisher
	sources     map[watchtower.SourceKind]services.WatchtowerSource
	now         func() int64
}

func New(p Params) *Service {
	sources := make(map[watchtower.SourceKind]services.WatchtowerSource, len(p.Sources))
	for _, source := range p.Sources {
		if source != nil {
			sources[source.Kind()] = source
		}
	}

	return &Service{
		l:           p.Logger.Named("service.watchtower"),
		repo:        p.Repo,
		projector:   p.Projector,
		permissions: p.Permissions,
		definitions: p.Definitions,
		runs:        p.Runs,
		events:      p.Events,
		sources:     sources,
		now:         timeutils.NowUnix,
	}
}

func AsService(s *Service) services.WatchtowerService { return s }

// visibleKinds is the kinds this reader may be shown: those whose source
// resource they may read. A request naming kinds is narrowed to the ones
// they may see; naming none means all of them.
func (s *Service) visibleKinds(
	ctx context.Context,
	actor *services.RequestActor,
	requested []watchtower.SourceKind,
) ([]watchtower.SourceKind, error) {
	candidates := requested
	if len(candidates) == 0 {
		candidates = watchtower.AllSourceKinds()
	}

	allowed := make(map[permission.Resource]bool, len(candidates))
	kinds := make([]watchtower.SourceKind, 0, len(candidates))
	for _, kind := range candidates {
		if !kind.IsValid() {
			return nil, errortypes.NewValidationError(
				"kinds",
				errortypes.ErrInvalid,
				"Unknown watchtower kind",
			)
		}
		resource := kind.ReadResource()
		ok, seen := allowed[resource]
		if !seen {
			result, err := s.permissions.Check(ctx, &services.PermissionCheckRequest{
				PrincipalType:  actor.PrincipalType,
				PrincipalID:    actor.PrincipalID,
				UserID:         actor.UserID,
				APIKeyID:       actor.APIKeyID,
				BusinessUnitID: actor.BusinessUnitID,
				OrganizationID: actor.OrganizationID,
				Resource:       resource.String(),
				Operation:      permission.OpRead,
			})
			if err != nil {
				return nil, err
			}
			ok = result != nil && result.Allowed
			allowed[resource] = ok
		}
		if ok {
			kinds = append(kinds, kind)
		}
	}

	return kinds, nil
}

func (s *Service) List(
	ctx context.Context,
	req services.ListWatchtowerItemsRequest,
	actor *services.RequestActor,
) (*services.WatchtowerPage, error) {
	kinds, err := s.visibleKinds(ctx, actor, req.Kinds)
	if err != nil {
		return nil, err
	}
	for _, severity := range req.Severities {
		if !severity.IsValid() {
			return nil, errortypes.NewValidationError(
				"severities",
				errortypes.ErrInvalid,
				"Unknown severity",
			)
		}
	}

	cursor, err := s.repo.GetCursor(ctx, repositories.GetWatchtowerCursorRequest{
		UserID:     actor.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	page := &services.WatchtowerPage{Items: []*watchtower.Item{}, SeenAt: cursor.SeenAt}
	if len(kinds) == 0 {
		return page, nil
	}

	first := req.First
	if first <= 0 {
		first = defaultPageSize
	}
	if first > maxPageSize {
		first = maxPageSize
	}

	listReq := repositories.ListWatchtowerItemsRequest{
		TenantInfo:     req.TenantInfo,
		Kinds:          kinds,
		Severities:     req.Severities,
		UnresolvedOnly: req.UnresolvedOnly,
		Since:          req.Since,
		Limit:          first + 1,
	}
	if req.After != "" {
		decoded, dErr := pagination.DecodeCursor(req.After)
		if dErr != nil {
			return nil, errortypes.NewValidationError(
				"after",
				errortypes.ErrInvalid,
				"Cursor is invalid",
			)
		}
		listReq.BeforeOccurredAt = decoded.CreatedAt
		listReq.BeforeID = decoded.ID
	}

	items, err := s.repo.List(ctx, listReq)
	if err != nil {
		return nil, err
	}
	if len(items) > first {
		items = items[:first]
		page.HasNextPage = true
	}
	for _, item := range items {
		item.Seen = item.OccurredAt <= cursor.SeenAt
	}
	page.Items = items
	if last := len(items); last > 0 {
		encoded, eErr := pagination.EncodeCursor(pagination.Cursor{
			CreatedAt: items[last-1].OccurredAt,
			ID:        items[last-1].ID,
		})
		if eErr != nil {
			return nil, fmt.Errorf("encode watchtower cursor: %w", eErr)
		}
		page.EndCursor = encoded
	}

	return page, nil
}

func (s *Service) Counts(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor *services.RequestActor,
) (*services.WatchtowerCounts, error) {
	kinds, err := s.visibleKinds(ctx, actor, nil)
	if err != nil {
		return nil, err
	}
	cursor, err := s.repo.GetCursor(ctx, repositories.GetWatchtowerCursorRequest{
		UserID:     actor.UserID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	counts := &services.WatchtowerCounts{
		ByKind: map[watchtower.SourceKind]int{},
		SeenAt: cursor.SeenAt,
	}
	if len(kinds) == 0 {
		return counts, nil
	}

	stored, err := s.repo.Counts(ctx, repositories.CountWatchtowerItemsRequest{
		TenantInfo: tenant,
		Kinds:      kinds,
		SeenAt:     cursor.SeenAt,
	})
	if err != nil {
		return nil, err
	}
	counts.Unresolved = stored.Unresolved
	counts.Critical = stored.Critical
	counts.UnseenCritical = stored.UnseenCritical
	counts.Unseen = stored.Unseen
	for _, row := range stored.ByKind {
		counts.ByKind[row.SourceKind] = row.Count
	}

	return counts, nil
}

// MarkSeen moves the reader's cursor to seenAt, or to now when zero, and
// returns the counts as they stand afterwards.
func (s *Service) MarkSeen(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor *services.RequestActor,
	seenAt int64,
) (*services.WatchtowerCounts, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewBusinessError("Only a person has a place in the watchtower")
	}
	if seenAt <= 0 {
		seenAt = s.now()
	}

	if _, err := s.repo.SetCursor(ctx, &watchtower.Cursor{
		UserID:         actor.UserID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		SeenAt:         seenAt,
	}); err != nil {
		return nil, err
	}

	return s.Counts(ctx, tenant, actor)
}

// Dismiss resolves an item by hand. The source is untouched: a dismissed
// failed run is still a failed run, it is just no longer on the tower.
func (s *Service) Dismiss(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	actor *services.RequestActor,
) (*watchtower.Item, error) {
	item, err := s.repo.GetByID(
		ctx,
		repositories.GetWatchtowerItemRequest{ID: id, TenantInfo: tenant},
	)
	if err != nil {
		return nil, err
	}
	if err = s.assertMaySee(ctx, actor, item); err != nil {
		return nil, err
	}
	if item.IsResolved() {
		return item, nil
	}

	resolved, err := s.repo.ResolveByID(
		ctx,
		repositories.GetWatchtowerItemRequest{ID: id, TenantInfo: tenant},
		s.now(),
	)
	if err != nil {
		return nil, err
	}
	s.projector.publish(ctx, resolved, "resolved")

	return resolved, nil
}

func (s *Service) assertMaySee(
	ctx context.Context,
	actor *services.RequestActor,
	item *watchtower.Item,
) error {
	kinds, err := s.visibleKinds(ctx, actor, []watchtower.SourceKind{item.SourceKind})
	if err != nil {
		return err
	}
	if len(kinds) == 0 {
		return errortypes.NewAuthorizationError("You cannot see this watchtower item")
	}

	return nil
}

// HandOff gives an item to an agent. A named agent runs on the item's
// subject at once. Otherwise the item's event is published to whoever
// subscribes to it; with no subscriber, the answer is who could take it,
// so the person can pick rather than be told nothing happened.
func (s *Service) HandOff(
	ctx context.Context,
	req services.HandOffWatchtowerItemRequest,
	actor *services.RequestActor,
) (*services.HandOffWatchtowerItemResult, error) {
	item, err := s.repo.GetByID(
		ctx,
		repositories.GetWatchtowerItemRequest{ID: req.ItemID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if err = s.assertMaySee(ctx, actor, item); err != nil {
		return nil, err
	}
	if item.SubjectType == "" || item.SubjectID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"This item is not about a record an agent can work on",
		)
	}

	if req.AgentDefinitionID.IsNotNil() {
		if s.runs == nil {
			return nil, errortypes.NewBusinessError("Agents cannot be started on this deployment")
		}
		run, rErr := s.runs.StartForDefinition(ctx, &services.StartAgentRunForDefinitionRequest{
			DefinitionID: req.AgentDefinitionID,
			SubjectType:  item.SubjectType,
			SubjectID:    item.SubjectID,
			Trigger:      agent.RunTriggerManual,
			EventKind:    item.EventKind,
			TenantInfo:   req.TenantInfo,
		}, actor)
		if rErr != nil {
			return nil, rErr
		}

		return &services.HandOffWatchtowerItemResult{Item: item, Run: run}, nil
	}

	result := &services.HandOffWatchtowerItemResult{Item: item}
	if item.EventKind != "" {
		subscribers, lErr := s.definitions.ListEnabledByTrigger(
			ctx,
			repositories.ListAgentDefinitionsByTriggerRequest{
				TenantInfo: req.TenantInfo,
				Mode:       agentdefinition.TriggerEvent,
				EventKind:  item.EventKind,
			},
		)
		if lErr != nil {
			return nil, lErr
		}
		if len(subscribers) > 0 && s.events != nil {
			services.PublishAgentEvent(ctx, s.events, services.AgentEvent{
				Kind:       item.EventKind,
				SubjectID:  item.SubjectID,
				TenantInfo: req.TenantInfo,
			})
			result.Subscribers = subscribers

			return result, nil
		}
		for _, template := range agentdefinition.AllTemplates() {
			for _, kind := range template.StarterEvents() {
				if kind == item.EventKind {
					result.Templates = append(result.Templates, template)
				}
			}
		}
	}

	candidates, err := s.definitions.List(ctx, &repositories.ListAgentDefinitionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: req.TenantInfo,
			Pagination: pagination.Info{Limit: 100},
		},
		EnabledOnly: true,
	})
	if err != nil {
		return nil, err
	}
	result.Candidates = make([]*agentdefinition.Definition, 0, len(candidates.Items))
	for _, definition := range candidates.Items {
		if definition != nil && definition.TriggerMode != agentdefinition.TriggerChat {
			result.Candidates = append(result.Candidates, definition)
		}
	}
	sort.Slice(result.Candidates, func(i, j int) bool {
		return result.Candidates[i].Name < result.Candidates[j].Name
	})

	return result, nil
}
