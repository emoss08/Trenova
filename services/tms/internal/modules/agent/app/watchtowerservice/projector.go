package watchtowerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/caseownerports"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/emoss08/trenova/internal/modules/agent/domain/desklens"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentactivityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type ProjectorParams struct {
	fx.In

	Logger        *zap.Logger
	Repo          watchtowerports.WatchtowerRepository
	Router        *Router
	Permissions   permissionports.PermissionEngine
	Realtime      realtimeports.RealtimeService
	Notifications *notificationservice.Service
	Owners        caseownerports.Service
	// Events wakes the agents that subscribe to an item's event when a sweep
	// first raises it.
	Events agenteventports.AgentEventPublisher
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
	repo          watchtowerports.WatchtowerRepository
	router        *Router
	access        kindAccess
	realtime      realtimeports.RealtimeService
	notifications *notificationservice.Service
	owners        caseownerports.Service
	events        agenteventports.AgentEventPublisher
	now           func() int64
}

func NewProjector(p ProjectorParams) *Projector {
	return &Projector{
		l:             p.Logger.Named("service.watchtower.projector"),
		repo:          p.Repo,
		router:        p.Router,
		access:        kindAccess{permissions: p.Permissions, owners: p.Owners},
		realtime:      p.Realtime,
		notifications: p.Notifications,
		owners:        p.Owners,
		events:        p.Events,
		now:           timeutils.NowUnix,
	}
}

// AsProjector is the face fx hands to every source.
func AsProjector(p *Projector) watchtowerports.WatchtowerProjector { return p }

// ItemFromInput builds the stored shape from what a source said.
func ItemFromInput(input watchtowerports.WatchtowerItemInput) *watchtower.Item {
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
		Facts:          input.Facts,
		Impact:         input.Impact,
		Trail:          input.Trail,
		Panel:          input.Panel,
	}
	if input.Record != nil {
		item.RecordKind = input.Record.Kind
		item.RecordLabel = input.Record.Label
	}
	item.SetAbout(pulid.FirstNotNil(input.About, input.SubjectID))
	item.ApplyDeadline(input.Deadline)
	item.Normalize()

	return item
}

// Upsert projects one source record. Nothing here reaches the caller: a
// projection that fails is logged, and the nightly reconcile catches it.
func (p *Projector) Upsert(ctx context.Context, input watchtowerports.WatchtowerItemInput) {
	if err := p.upsert(ctx, input); err != nil {
		p.l.Warn("watchtower projection lost",
			zap.String("kind", string(input.SourceKind)),
			zap.String("source", input.SourceID),
			zap.Error(err),
		)
	}
}

func (p *Projector) upsert(
	ctx context.Context,
	input watchtowerports.WatchtowerItemInput,
) error {
	saved, action, err := p.project(ctx, input)
	if err != nil || saved == nil {
		return err
	}
	p.publish(ctx, saved, action)

	return nil
}

func (p *Projector) project(
	ctx context.Context,
	input watchtowerports.WatchtowerItemInput,
) (*watchtower.Item, string, error) {
	item := ItemFromInput(input)
	multiErr := errortypes.NewMultiError()
	item.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, "", multiErr
	}

	ctx = dbscope.WithTenant(ctx, input.TenantInfo.DBTenant())
	if record := p.caseRecord(ctx, &input, item.AboutID); record != item.AboutID {
		item.SetAbout(record)
	}
	saved, inserted, err := p.repo.Upsert(ctx, item)
	if err != nil {
		return nil, "", err
	}

	saved = p.route(ctx, saved, input.OwnerID, inserted)
	if inserted && saved.AboutID.IsNotNil() && !saved.IsResolved() {
		p.owners.Route(ctx, caseownerports.CandidatesRequest{
			TenantInfo: input.TenantInfo,
			Kind:       saved.AboutKind,
			RecordID:   saved.AboutID,
		}, input.OwnerID)
	}
	if input.RaisesEvent && (inserted || saved.Reopened) && !saved.IsResolved() &&
		saved.EventKind != "" && saved.SubjectID.IsNotNil() {
		agenteventports.PublishAgentEvent(ctx, p.events, agenteventports.AgentEvent{
			Kind:       saved.EventKind,
			SubjectID:  saved.SubjectID,
			TenantInfo: input.TenantInfo,
			Detail:     saved.Title,
		})
	}
	if saved.Unchanged {
		return nil, "", nil
	}

	action := agentactivityports.ActivityUpdated
	if inserted {
		action = agentactivityports.ActivityCreated
	}
	if inserted && saved.Severity == watchtower.SeverityCritical {
		p.notifyCritical(ctx, saved)
	}

	return saved, action, nil
}

// Resolve closes the item for a source record that is no longer open.
func (p *Projector) Resolve(
	ctx context.Context,
	tenant pagination.TenantInfo,
	kind watchtower.SourceKind,
	sourceID string,
) {
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	item, err := p.repo.Resolve(ctx, watchtowerports.ResolveWatchtowerItemRequest{
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
	if item == nil {
		return
	}

	if err := realtimeinvalidation.Publish(ctx, p.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: item.OrganizationID,
		BusinessUnitID: item.BusinessUnitID,
		ActorType:      permissionports.PrincipalTypeSystem,
		Resource:       RealtimeResource,
		Action:         action,
		RecordID:       item.ID,
		Entity:         item,
	}); err != nil {
		p.l.Warn("watchtower invalidation lost",
			zap.String("item", item.ID.String()), zap.Error(err))
	}
}

func (p *Projector) publishBulk(ctx context.Context, tenant pagination.TenantInfo, action string) {
	if err := realtimeinvalidation.Publish(ctx, p.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		ActorType:      permissionports.PrincipalTypeSystem,
		Resource:       RealtimeResource,
		Action:         action,
	}); err != nil {
		p.l.Warn("watchtower bulk invalidation lost",
			zap.String("organization", tenant.OrgID.String()), zap.Error(err))
	}
}

// notifyCritical tells the people who could open the record that something
// critical appeared. Recipients are whoever may read the source's resource,
// bounded, so a critical weather warning reaches dispatch and not the whole
// company.
func (p *Projector) notifyCritical(ctx context.Context, item *watchtower.Item) {
	correlation := item.ID.String()
	link := item.Path
	if link == "" {
		link = FeedPath + "?item=" + item.ID.String()
	}
	if _, err := p.notifications.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
		Tenant:    pagination.TenantInfo{OrgID: item.OrganizationID, BuID: item.BusinessUnitID},
		Resource:  item.SourceKind.ReadResource(),
		Operation: permission.OpRead,
		Limit:     maxCriticalRecipients,
		Now:       p.now(),
		Notification: notification.Notification{
			EventType:       EventCritical,
			Priority:        notification.PriorityHigh,
			Title:           item.Title,
			Message:         item.Summary,
			Source:          notificationSource,
			CorrelationID:   &correlation,
			Data:            noticeData(item, link),
			RelatedEntities: map[string]any{"watchtowerItemId": item.ID.String()},
		},
	}); err != nil {
		p.l.Warn("could not tell anyone about a critical item",
			zap.String("item", item.ID.String()), zap.Error(err))
	}
}

/*
caseRecord is the record a decision's case is on. An agent proposes on the
thing it works: a detention desk on the detention occurrence, not on the
load. The occurrence's own item already says which record it is about, so a
plan or a change about the occurrence is filed under that record, where the
person finds the case, rather than under a record with no case of its own.
Only decisions are looked up; a condition says what it is about itself.
*/
func (p *Projector) caseRecord(
	ctx context.Context,
	input *watchtowerports.WatchtowerItemInput,
	about pulid.ID,
) pulid.ID {
	if about.IsNil() || (input.SourceKind != watchtower.SourceAgentProposal &&
		input.SourceKind != watchtower.SourceAgentPlan) {
		return about
	}

	raised, err := p.repo.List(ctx, watchtowerports.ListWatchtowerItemsQuery{
		TenantInfo: input.TenantInfo,
		SubjectID:  about,
		Limit:      caseRecordLookups,
	})
	if err != nil {
		p.l.Warn("could not read what a decision's record is about",
			zap.String("record", about.String()), zap.Error(err))

		return about
	}
	for _, item := range raised {
		if item.AboutID.IsNotNil() && item.AboutID != about &&
			desklens.Of(item.SourceKind) == desklens.Watchtower {
			return item.AboutID
		}
	}

	return about
}

const caseRecordLookups = 5
