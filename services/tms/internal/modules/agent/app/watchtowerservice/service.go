package watchtowerservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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

	bulkResolvedAction = "bulk_resolved"
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	Repo        watchtowerports.WatchtowerRepository
	Projector   *Projector
	Permissions permissionports.PermissionEngine
	Definitions agentdefinitionports.AgentDefinitionRepository
	Runs        agentports.AgentRunService         `optional:"true"`
	RunRepo     agentports.AgentRunRepository      `optional:"true"`
	Events      agentports.AgentEventPublisher     `optional:"true"`
	Sources     []watchtowerports.WatchtowerSource `                group:"watchtower_sources"`
	Dismisser   *Dismisser
	Exceptions  *Exceptions
}

// Service is the watchtower's read face: what a reader may see of it, where
// their unseen line sits, and how an item is handed to an agent. It also
// owns the sweeps, which is why it knows the sources.
//
// The write face lives in Projector and is injected rather than embedded.
// See the note there for why the two cannot be one constructor.
type Service struct {
	l           *zap.Logger
	repo        watchtowerports.WatchtowerRepository
	projector   *Projector
	permissions permissionports.PermissionEngine
	access      kindAccess
	dismisser   *Dismisser
	definitions agentdefinitionports.AgentDefinitionRepository
	runs        agentports.AgentRunService
	runRepo     agentports.AgentRunRepository
	exceptions  *Exceptions
	events      agentports.AgentEventPublisher
	sources     map[watchtower.SourceKind]watchtowerports.WatchtowerSource
	now         func() int64
}

func New(p Params) *Service {
	sources := make(map[watchtower.SourceKind]watchtowerports.WatchtowerSource, len(p.Sources))
	for _, source := range p.Sources {
		if source != nil {
			sources[source.Kind()] = source
		}
	}

	access := kindAccess{permissions: p.Permissions}
	svc := &Service{
		l:           p.Logger.Named("service.watchtower"),
		repo:        p.Repo,
		projector:   p.Projector,
		permissions: p.Permissions,
		access:      access,
		definitions: p.Definitions,
		runs:        p.Runs,
		runRepo:     p.RunRepo,
		exceptions:  p.Exceptions,
		events:      p.Events,
		sources:     sources,
		dismisser:   p.Dismisser,
		now:         timeutils.NowUnix,
	}

	return svc
}

func ItemCursor(item *watchtower.Item, orderByDue bool) (string, error) {
	key := item.OccurredAt
	if orderByDue {
		key = item.DueSortKey()
	}
	encoded, err := pagination.EncodePositionCursor(pagination.PositionCursor{At: key, ID: item.ID})
	if err != nil {
		return "", fmt.Errorf("encode watchtower cursor: %w", err)
	}
	return encoded, nil
}

func AsService(s *Service) watchtowerports.WatchtowerService { return s }

func (s *Service) List(
	ctx context.Context,
	req watchtowerports.ListWatchtowerItemsRequest,
	actor *permissionports.RequestActor,
) (*watchtowerports.WatchtowerPage, error) {
	kinds, err := s.access.visibleKinds(ctx, actor, req.Kinds)
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
	for _, status := range req.Statuses {
		if !status.IsValid() {
			return nil, errortypes.NewValidationError(
				"statuses",
				errortypes.ErrInvalid,
				"Unknown status",
			)
		}
	}
	assignees := req.AssigneeIDs
	if req.AssignedToMe {
		if actor.UserID.IsNil() {
			return nil, errortypes.NewBusinessError("Only a person has exceptions of their own")
		}
		assignees = append(slices.Clone(assignees), actor.UserID)
	}

	cursor, err := s.repo.GetCursor(ctx, watchtowerports.GetWatchtowerCursorRequest{
		UserID:     actor.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	page := &watchtowerports.WatchtowerPage{Items: []*watchtower.Item{}, SeenAt: cursor.SeenAt}
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

	listReq := watchtowerports.ListWatchtowerItemsQuery{
		TenantInfo:     req.TenantInfo,
		Kinds:          kinds,
		Severities:     req.Severities,
		UnresolvedOnly: req.UnresolvedOnly,
		Since:          req.Since,
		Limit:          first + 1,
		Statuses:       req.Statuses,
		AssigneeIDs:    assignees,
		Unassigned:     req.Unassigned,
		SubjectID:      req.SubjectID,
		OrderByDue:     req.OrderByDue,
	}
	if req.OverdueOnly {
		listReq.OverdueAt = s.now()
	}
	if req.After != "" {
		decoded, dErr := pagination.DecodePositionCursor(req.After)
		if dErr != nil {
			return nil, errortypes.NewValidationError(
				"after",
				errortypes.ErrInvalid,
				"Cursor is invalid",
			)
		}
		listReq.BeforeID = decoded.ID
		if req.OrderByDue {
			listReq.BeforeDueAt = decoded.At
		} else {
			listReq.BeforeOccurredAt = decoded.At
		}
	}

	items, err := s.repo.List(ctx, listReq)
	if err != nil {
		return nil, err
	}
	if len(items) > first {
		items = items[:first]
		page.HasNextPage = true
	}
	read, err := s.readItemIDs(ctx, req.TenantInfo, actor, items)
	if err != nil {
		return nil, err
	}
	page.Cursors = make([]string, 0, len(items))
	for _, item := range items {
		_, opened := read[item.ID]
		item.Seen = opened || item.OccurredAt <= cursor.SeenAt
		encoded, eErr := ItemCursor(item, req.OrderByDue)
		if eErr != nil {
			return nil, eErr
		}
		page.Cursors = append(page.Cursors, encoded)
	}
	page.Items = items
	if last := len(page.Cursors); last > 0 {
		page.EndCursor = page.Cursors[last-1]
	}

	return page, nil
}

func (s *Service) Counts(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
) (*watchtowerports.WatchtowerCounts, error) {
	kinds, err := s.access.visibleKinds(ctx, actor, nil)
	if err != nil {
		return nil, err
	}
	cursor, err := s.repo.GetCursor(ctx, watchtowerports.GetWatchtowerCursorRequest{
		UserID:     actor.UserID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	counts := &watchtowerports.WatchtowerCounts{
		ByKind: map[watchtower.SourceKind]int{},
		SeenAt: cursor.SeenAt,
	}
	if len(kinds) == 0 {
		return counts, nil
	}

	stored, err := s.repo.Counts(ctx, watchtowerports.CountWatchtowerItemsRequest{
		TenantInfo: tenant,
		Kinds:      kinds,
		SeenAt:     cursor.SeenAt,
		AssigneeID: actor.UserID,
		ReaderID:   actor.UserID,
		Now:        s.now(),
	})
	if err != nil {
		return nil, err
	}
	counts.Unresolved = stored.Unresolved
	counts.Critical = stored.Critical
	counts.Active = stored.Active
	counts.ActiveCritical = stored.ActiveCritical
	counts.UnseenCritical = stored.UnseenCritical
	counts.Unseen = stored.Unseen
	counts.Overdue = stored.Overdue
	counts.Unassigned = stored.Unassigned
	counts.Mine = stored.Mine
	counts.MineOverdue = stored.MineOverdue
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
	actor *permissionports.RequestActor,
	seenAt int64,
) (*watchtowerports.WatchtowerCounts, error) {
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

func (s *Service) Dismiss(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	return s.dismisser.Dismiss(ctx, tenant, id, actor)
}

func (s *Service) PlanDismiss(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	actor *permissionports.RequestActor,
) (*kernelports.RecordChange[watchtower.Item], error) {
	return s.dismisser.PlanDismiss(ctx, tenant, id, actor)
}

// HandOff gives an item to an agent. A named agent runs on the item's
// subject at once. Otherwise the item's event is published to whoever
// subscribes to it; with no subscriber, the answer is who could take it,
// so the person can pick rather than be told nothing happened.
func (s *Service) HandOff(
	ctx context.Context,
	req watchtowerports.HandOffWatchtowerItemRequest,
	actor *permissionports.RequestActor,
) (*watchtowerports.HandOffWatchtowerItemResult, error) {
	item, err := s.repo.GetByID(
		ctx,
		watchtowerports.GetWatchtowerItemRequest{ID: req.ItemID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if err = s.access.assertMaySee(ctx, actor, item); err != nil {
		return nil, err
	}
	if item.SubjectType == "" || item.SubjectID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"This item is not about a record an agent can work on",
		)
	}

	if req.AgentDefinitionID.IsNotNil() {
		return s.handOffToAgent(ctx, req, item, actor)
	}

	result := &watchtowerports.HandOffWatchtowerItemResult{Item: item}
	if item.EventKind != "" {
		subscribers, lErr := s.definitions.ListEnabledByTrigger(
			ctx,
			agentdefinitionports.ListAgentDefinitionsByTriggerRequest{
				TenantInfo: req.TenantInfo,
				Mode:       agentdefinition.TriggerEvent,
				EventKind:  item.EventKind,
			},
		)
		if lErr != nil {
			return nil, lErr
		}
		if len(subscribers) > 0 && s.events != nil {
			agentports.PublishAgentEvent(ctx, s.events, agentports.AgentEvent{
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

	candidates, err := s.handOffCandidates(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	result.Candidates = candidates

	return result, nil
}
