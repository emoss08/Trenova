package watchtowerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ProjectorParams struct {
	fx.In

	Logger        *zap.Logger
	Repo          repositories.WatchtowerRepository
	Roles         repositories.RoleRepository  `optional:"true"`
	Realtime      services.RealtimeService     `optional:"true"`
	Notifications *notificationservice.Service `optional:"true"`
}

// Projector is the watchtower's write face: what the sources put on the
// tower and what they take off it again.
//
// It is deliberately separate from Service, and the separation is load
// bearing rather than tidy. Almost every domain service holds a projector —
// detention, billing, carrier intelligence, service failures — while the
// read face has to know every source in order to reconcile against them,
// and some of those sources sit behind the agent decision queue, which
// reaches the tool registry, which reaches billing, which reaches
// detention. Held together in one constructor that closes a loop in the
// dependency graph and the process will not start. A projector that needs
// nothing but its repository cannot be part of a cycle, so the write face
// is free to be held by anyone.
type Projector struct {
	l             *zap.Logger
	repo          repositories.WatchtowerRepository
	roles         repositories.RoleRepository
	realtime      services.RealtimeService
	notifications *notificationservice.Service
	now           func() int64
}

func NewProjector(p ProjectorParams) *Projector {
	return &Projector{
		l:             p.Logger.Named("service.watchtower.projector"),
		repo:          p.Repo,
		roles:         p.Roles,
		realtime:      p.Realtime,
		notifications: p.Notifications,
		now:           timeutils.NowUnix,
	}
}

// AsProjector is the face fx hands to every source.
func AsProjector(p *Projector) services.WatchtowerProjector { return p }

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
func (p *Projector) Upsert(ctx context.Context, input services.WatchtowerItemInput) {
	if _, err := p.upsert(ctx, input); err != nil {
		p.l.Warn("watchtower projection lost",
			zap.String("kind", string(input.SourceKind)),
			zap.String("source", input.SourceID),
			zap.Error(err),
		)
	}
}

func (p *Projector) upsert(
	ctx context.Context,
	input services.WatchtowerItemInput,
) (*watchtower.Item, error) {
	item := itemFromInput(input)
	multiErr := errortypes.NewMultiError()
	item.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, inserted, err := p.repo.Upsert(ctx, item)
	if err != nil {
		return nil, err
	}

	action := services.ActivityUpdated
	if inserted {
		action = services.ActivityCreated
	}
	p.publish(ctx, saved, action)
	if inserted && saved.Severity == watchtower.SeverityCritical {
		p.notifyCritical(ctx, saved)
	}

	return saved, nil
}

// Resolve closes the item for a source record that is no longer open.
func (p *Projector) Resolve(
	ctx context.Context,
	tenant pagination.TenantInfo,
	kind watchtower.SourceKind,
	sourceID string,
) {
	item, err := p.repo.Resolve(ctx, repositories.ResolveWatchtowerItemRequest{
		TenantInfo: tenant,
		SourceKind: kind,
		SourceID:   sourceID,
		ResolvedAt: p.now(),
	})
	if err != nil {
		p.l.Warn("watchtower resolution lost",
			zap.String("kind", string(kind)),
			zap.String("source", sourceID),
			zap.Error(err),
		)

		return
	}
	if item != nil {
		p.publish(ctx, item, "resolved")
	}
}

func (p *Projector) publish(ctx context.Context, item *watchtower.Item, action string) {
	if p.realtime == nil || item == nil {
		return
	}

	if err := realtimeinvalidation.Publish(ctx, p.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: item.OrganizationID,
		BusinessUnitID: item.BusinessUnitID,
		ActorType:      services.PrincipalTypeSystem,
		Resource:       RealtimeResource,
		Action:         action,
		RecordID:       item.ID,
		Entity:         item,
	}); err != nil {
		p.l.Warn("watchtower invalidation lost",
			zap.String("item", item.ID.String()), zap.Error(err))
	}
}

// notifyCritical tells the people who could open the record that something
// critical appeared. Recipients are whoever may read the source's resource,
// bounded, so a critical weather warning reaches dispatch and not the whole
// company.
func (p *Projector) notifyCritical(ctx context.Context, item *watchtower.Item) {
	if p.notifications == nil || p.roles == nil {
		return
	}

	tenant := pagination.TenantInfo{OrgID: item.OrganizationID, BuID: item.BusinessUnitID}
	recipients, err := p.roles.ListUsersWithPermission(
		ctx,
		repositories.ListUsersWithPermissionRequest{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Resource:       item.SourceKind.ReadResource(),
			Operation:      permission.OpRead,
			Now:            p.now(),
		},
	)
	if err != nil {
		p.l.Warn("could not find who to tell about a critical item",
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
		if _, err = p.notifications.Create(ctx, &notification.Notification{
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
			p.l.Warn("critical watchtower notification lost",
				zap.String("item", item.ID.String()),
				zap.String("userId", recipient.UserID.String()),
				zap.Error(err),
			)
		}
	}
}
