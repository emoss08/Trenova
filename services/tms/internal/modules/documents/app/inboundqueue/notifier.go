package inboundqueue

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowersources"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxRecordInvalidations = 100
	realtimeActionCreated  = "created"
)

var _ inboundmessageports.TriageMessageNotifier = (*Notifier)(nil)

type NotifierParams struct {
	fx.In

	Logger     *zap.Logger
	Realtime   realtimeports.RealtimeService                 `optional:"true"`
	Watchtower watchtowerports.WatchtowerProjector           `optional:"true"`
	Briefing   inboundmessageports.TriageBriefingInvalidator `optional:"true"`
}

type notifierDeps struct {
	logger     *zap.Logger
	realtime   realtimeports.RealtimeService
	watchtower watchtowerports.WatchtowerProjector
	briefing   inboundmessageports.TriageBriefingInvalidator
}

type Notifier struct {
	l          *zap.Logger
	realtime   realtimeports.RealtimeService
	watchtower watchtowerports.WatchtowerProjector
	briefing   inboundmessageports.TriageBriefingInvalidator
}

func NewNotifier(p NotifierParams) *Notifier {
	return newNotifier(notifierDeps{
		logger:     p.Logger,
		realtime:   p.Realtime,
		watchtower: p.Watchtower,
		briefing:   p.Briefing,
	})
}

func newNotifier(deps notifierDeps) *Notifier {
	return &Notifier{
		l:          deps.logger.Named("service.triage-notifier"),
		realtime:   deps.realtime,
		watchtower: deps.watchtower,
		briefing:   deps.briefing,
	}
}

func (n *Notifier) MessagesChanged(
	ctx context.Context,
	messages ...*inboundmessage.InboundMessage,
) {
	n.changed(ctx, InboxRealtimeAction, messages)
}

func (n *Notifier) MessagesCreated(
	ctx context.Context,
	messages ...*inboundmessage.InboundMessage,
) {
	n.changed(ctx, realtimeActionCreated, messages)
}

func (n *Notifier) TenantChanged(ctx context.Context, tenantInfo pagination.TenantInfo) {
	kernelports.AfterCommit(ctx, func(committed context.Context) {
		n.Publish(committed, tenantInfo, pulid.Nil, InboxRealtimeAction)
		if n.briefing != nil {
			n.briefing.InvalidateBriefing(committed, tenantInfo)
		}
	})
}

func (n *Notifier) changed(
	ctx context.Context,
	action string,
	messages []*inboundmessage.InboundMessage,
) {
	if len(messages) == 0 {
		return
	}

	kernelports.AfterCommit(ctx, func(committed context.Context) {
		for _, message := range messages {
			n.project(committed, message)
		}
		n.publishAll(committed, action, messages)
		n.invalidateBriefings(committed, messages)
	})
}

func (n *Notifier) project(ctx context.Context, message *inboundmessage.InboundMessage) {
	if n.watchtower == nil {
		return
	}

	if !message.NeedsReview() {
		n.watchtower.Resolve(
			ctx,
			TenantOfMessage(message),
			watchtower.SourceInboundMessage,
			message.ID.String(),
		)

		return
	}

	n.watchtower.Upsert(ctx, watchtowersources.DescribeInboundMessage(message))
}

func (n *Notifier) publishAll(
	ctx context.Context,
	action string,
	messages []*inboundmessage.InboundMessage,
) {
	if n.realtime == nil {
		return
	}
	if len(messages) <= maxRecordInvalidations {
		for _, message := range messages {
			n.Publish(ctx, TenantOfMessage(message), message.ID, action)
		}

		return
	}

	seen := make(map[pagination.TenantInfo]struct{}, 1)
	for _, message := range messages {
		tenantInfo := TenantOfMessage(message)
		if _, ok := seen[tenantInfo]; ok {
			continue
		}
		seen[tenantInfo] = struct{}{}
		n.Publish(ctx, tenantInfo, pulid.Nil, action)
	}
}

func (n *Notifier) Publish(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
	action string,
) {
	if n.realtime == nil {
		return
	}

	if err := realtimeinvalidation.Publish(ctx, n.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorType:      permissionports.PrincipalTypeSystem,
		Resource:       permission.ResourceInboundMessage.String(),
		Action:         action,
		RecordID:       messageID,
	}); err != nil {
		n.l.Warn("inbox invalidation lost",
			zap.String("messageId", messageID.String()), zap.Error(err))
	}
}

func (n *Notifier) invalidateBriefings(
	ctx context.Context,
	messages []*inboundmessage.InboundMessage,
) {
	if n.briefing == nil {
		return
	}

	seen := make(map[pagination.TenantInfo]struct{}, 1)
	for _, message := range messages {
		tenantInfo := TenantOfMessage(message)
		if _, ok := seen[tenantInfo]; ok {
			continue
		}
		seen[tenantInfo] = struct{}{}
		n.briefing.InvalidateBriefing(ctx, tenantInfo)
	}
}

func TenantOfMessage(message *inboundmessage.InboundMessage) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: message.OrganizationID, BuID: message.BusinessUnitID}
}
