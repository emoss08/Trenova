package watchtowerservice

import (
	"context"
	"fmt"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
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

	Logger        *zap.Logger
	Repo          repositories.WatchtowerRepository
	Permissions   services.PermissionEngine
	Definitions   repositories.AgentDefinitionRepository
	Roles         repositories.RoleRepository  `optional:"true"`
	Runs          services.AgentRunService     `optional:"true"`
	Events        services.AgentEventPublisher `optional:"true"`
	Realtime      services.RealtimeService     `optional:"true"`
	Notifications *notificationservice.Service `optional:"true"`
	Sources       []services.WatchtowerSource  `group:"watchtower_sources"`
}

// Service keeps the watchtower: what the sources project into it, what a
// reader may see of it, and how an item is handed to an agent.
type Service struct {
	l             *zap.Logger
	repo          repositories.WatchtowerRepository
	permissions   services.PermissionEngine
	definitions   repositories.AgentDefinitionRepository
	roles         repositories.RoleRepository
	runs          services.AgentRunService
	events        services.AgentEventPublisher
	realtime      services.RealtimeService
	notifications *notificationservice.Service
	sources       map[watchtower.SourceKind]services.WatchtowerSource
	now           func() int64
}

func New(p Params) *Service {
	sources := make(map[watchtower.SourceKind]services.WatchtowerSource, len(p.Sources))
	for _, source := range p.Sources {
		if source != nil {
			sources[source.Kind()] = source
		}
	}

	return &Service{
		l:             p.Logger.Named("service.watchtower"),
		repo:          p.Repo,
		permissions:   p.Permissions,
		definitions:   p.Definitions,
		roles:         p.Roles,
		runs:          p.Runs,
		events:        p.Events,
		realtime:      p.Realtime,
		notifications: p.Notifications,
		sources:       sources,
		now:           timeutils.NowUnix,
	}
}

// AsProjector and AsService are the two faces fx hands out: sources hold the
// first, the API the second.
func AsProjector(s *Service) services.WatchtowerProjector { return s }

func AsService(s *Service) services.WatchtowerService { return s }

// itemFromInput builds the stored shape from what a source said.
func itemFromInput(input services.WatchtowerItemInput) *watchtower.Item {
	occurredAt := input.OccurredAt
	if occurredAt <= 0 {
		occurredAt = timeutils.NowUnix()
	}
	item := &watchtower.Item{
		OrganizationID: input.TenantInfo.OrgID,
		BusinessUnitID: input.TenantInfo.BuID,
		SourceKind:     input.SourceKind,
		SourceID:       input.SourceID,
		Severity:       input.Severity,
		Title:          input.Title,
		Summary:        input.Summary,
		SubjectType:    input.SubjectType,
		SubjectID:      input.SubjectID,
		EventKind:      input.EventKind,
		Path:           input.Path,
		OccurredAt:     occurredAt,
	}
	item.Normalize()

	return item
}

// Upsert projects one source record. Nothing here reaches the caller: a
// projection that fails is logged, and the nightly reconcile catches it.
func (s *Service) Upsert(ctx context.Context, input services.WatchtowerItemInput) {
	if _, err := s.upsert(ctx, input); err != nil {
		s.l.Warn("watchtower projection lost",
			zap.String("kind", string(input.SourceKind)),
			zap.String("source", input.SourceID),
			zap.Error(err),
		)
	}
}

func (s *Service) upsert(ctx context.Context, input services.WatchtowerItemInput) (*watchtower.Item, error) {
	item := itemFromInput(input)
	multiErr := errortypes.NewMultiError()
	item.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, inserted, err := s.repo.Upsert(ctx, item)
	if err != nil {
		return nil, err
	}

	action := services.ActivityUpdated
	if inserted {
		action = services.ActivityCreated
	}
	s.publish(ctx, saved, action)
	if inserted && saved.Severity == watchtower.SeverityCritical {
		s.notifyCritical(ctx, saved)
	}

	return saved, nil
}

// Resolve closes the item for a source record that is no longer open.
func (s *Service) Resolve(
	ctx context.Context,
	tenant pagination.TenantInfo,
	kind watchtower.SourceKind,
	sourceID string,
) {
	item, err := s.repo.Resolve(ctx, repositories.ResolveWatchtowerItemRequest{
		TenantInfo: tenant,
		SourceKind: kind,
		SourceID:   sourceID,
		ResolvedAt: s.now(),
	})
	if err != nil {
		s.l.Warn("watchtower resolution lost",
			zap.String("kind", string(kind)),
			zap.String("source", sourceID),
			zap.Error(err),
		)

		return
	}
	if item != nil {
		s.publish(ctx, item, "resolved")
	}
}

func (s *Service) publish(ctx context.Context, item *watchtower.Item, action string) {
	if s.realtime == nil || item == nil {
		return
	}

	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: item.OrganizationID,
		BusinessUnitID: item.BusinessUnitID,
		ActorType:      services.PrincipalTypeSystem,
		Resource:       RealtimeResource,
		Action:         action,
		RecordID:       item.ID,
		Entity:         item,
	}); err != nil {
		s.l.Warn("watchtower invalidation lost", zap.String("item", item.ID.String()), zap.Error(err))
	}
}

// notifyCritical tells the people who could open the record that something
// critical appeared. Recipients are whoever may read the source's resource,
// bounded, so a critical weather warning reaches dispatch and not the whole
// company.
func (s *Service) notifyCritical(ctx context.Context, item *watchtower.Item) {
	if s.notifications == nil || s.roles == nil {
		return
	}

	tenant := pagination.TenantInfo{OrgID: item.OrganizationID, BuID: item.BusinessUnitID}
	recipients, err := s.roles.ListUsersWithPermission(ctx, repositories.ListUsersWithPermissionRequest{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Resource:       item.SourceKind.ReadResource(),
		Operation:      permission.OpRead,
		Now:            s.now(),
	})
	if err != nil {
		s.l.Warn("could not find who to tell about a critical item",
			zap.String("item", item.ID.String()), zap.Error(err))

		return
	}
	if len(recipients) > maxCriticalRecipients {
		recipients = recipients[:maxCriticalRecipients]
	}

	correlation := item.ID.String()
	link := item.Path
	if link == "" {
		link = FeedPath + "?item=" + item.ID.String()
	}
	for _, recipient := range recipients {
		buID := tenant.BuID
		userID := recipient.UserID
		if _, err = s.notifications.Create(ctx, &notification.Notification{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: &buID,
			TargetUserID:   &userID,
			Channel:        notification.ChannelUser,
			EventType:      EventCritical,
			Priority:       notification.PriorityHigh,
			Title:          item.Title,
			Message:        item.Summary,
			Source:         notificationSource,
			CorrelationID:  &correlation,
			Data: map[string]any{
				"link":       link,
				"itemId":     item.ID.String(),
				"sourceKind": string(item.SourceKind),
				"sourceId":   item.SourceID,
				"severity":   string(item.Severity),
			},
			RelatedEntities: map[string]any{"watchtowerItemId": item.ID.String()},
		}); err != nil {
			s.l.Warn("critical watchtower notification lost",
				zap.String("item", item.ID.String()),
				zap.String("userId", recipient.UserID.String()),
				zap.Error(err),
			)
		}
	}
}

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
			return nil, errortypes.NewValidationError("kinds", errortypes.ErrInvalid, "Unknown watchtower kind")
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
			return nil, errortypes.NewValidationError("severities", errortypes.ErrInvalid, "Unknown severity")
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
			return nil, errortypes.NewValidationError("after", errortypes.ErrInvalid, "Cursor is invalid")
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
	item, err := s.repo.GetByID(ctx, repositories.GetWatchtowerItemRequest{ID: id, TenantInfo: tenant})
	if err != nil {
		return nil, err
	}
	if err = s.assertMaySee(ctx, actor, item); err != nil {
		return nil, err
	}
	if item.IsResolved() {
		return item, nil
	}

	resolved, err := s.repo.ResolveByID(ctx, repositories.GetWatchtowerItemRequest{ID: id, TenantInfo: tenant}, s.now())
	if err != nil {
		return nil, err
	}
	s.publish(ctx, resolved, "resolved")

	return resolved, nil
}

func (s *Service) assertMaySee(ctx context.Context, actor *services.RequestActor, item *watchtower.Item) error {
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
	item, err := s.repo.GetByID(ctx, repositories.GetWatchtowerItemRequest{ID: req.ItemID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	if err = s.assertMaySee(ctx, actor, item); err != nil {
		return nil, err
	}
	if item.SubjectType == "" || item.SubjectID.IsNil() {
		return nil, errortypes.NewBusinessError("This item is not about a record an agent can work on")
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
		subscribers, lErr := s.definitions.ListEnabledByTrigger(ctx, repositories.ListAgentDefinitionsByTriggerRequest{
			TenantInfo: req.TenantInfo,
			Mode:       agentdefinition.TriggerEvent,
			EventKind:  item.EventKind,
		})
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
