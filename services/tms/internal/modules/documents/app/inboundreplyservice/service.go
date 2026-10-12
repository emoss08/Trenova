package inboundreplyservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Messages      inboundmessageports.InboundMessageRepository
	Replies       inboundmessageports.InboundReplyRepository
	Templates     inboundmessageports.TriageReplyTemplateRepository
	Work          inboundmessageports.TriageWorkRepository
	Activities    inboundmessageports.TriageActivityRecorder
	Notifier      inboundmessageports.TriageMessageNotifier
	Organizations tenantports.OrganizationRepository
	Nova          *triagenova.Caller
	Email         emailports.EmailService                   `optional:"true"`
	Peeks         inboundmessageports.TriageShipmentPeeker  `optional:"true"`
	Stops         inboundmessageports.TriageStopTimesReader `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	messages      inboundmessageports.InboundMessageRepository
	replies       inboundmessageports.InboundReplyRepository
	templates     inboundmessageports.TriageReplyTemplateRepository
	work          inboundmessageports.TriageWorkRepository
	activities    inboundmessageports.TriageActivityRecorder
	notifier      inboundmessageports.TriageMessageNotifier
	organizations tenantports.OrganizationRepository
	nova          triagenova.Completer
	email         emailports.EmailService
	peeks         inboundmessageports.TriageShipmentPeeker
	stops         inboundmessageports.TriageStopTimesReader
	now           func() int64
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.inbound-reply"),
		messages:      p.Messages,
		replies:       p.Replies,
		templates:     p.Templates,
		work:          p.Work,
		activities:    p.Activities,
		notifier:      p.Notifier,
		organizations: p.Organizations,
		nova:          p.Nova,
		email:         p.Email,
		peeks:         p.Peeks,
		stops:         p.Stops,
		now:           timeutils.NowUnix,
	}
}

func (s *Service) message(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
	withAttachments bool,
) (*inboundmessage.InboundMessage, error) {
	return s.messages.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:                 messageID,
		TenantInfo:         tenantInfo,
		IncludeAttachments: withAttachments,
	})
}

func (s *Service) record(ctx context.Context, activities ...*triage.Activity) {
	if len(activities) == 0 {
		return
	}
	if err := s.activities.RecordActivity(ctx, activities...); err != nil {
		s.l.Error("could not record a triage activity",
			zap.String("messageId", activities[0].MessageID.String()),
			zap.String("kind", string(activities[0].Kind)),
			zap.Error(err),
		)
	}
}

func (s *Service) changed(ctx context.Context, message *inboundmessage.InboundMessage) {
	s.notifier.MessagesChanged(ctx, message)
}

func tenantOf(message *inboundmessage.InboundMessage) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: message.OrganizationID, BuID: message.BusinessUnitID}
}
