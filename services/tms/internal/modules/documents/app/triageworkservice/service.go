package triageworkservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundreplyservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageworkports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxMessageWriteAttempts = 3

type documentFiler interface {
	AttachLineageToResource(
		ctx context.Context,
		documentID pulid.ID,
		resourceType string,
		resourceID string,
		tenantInfo pagination.TenantInfo,
		userID pulid.ID,
	) (*document.Document, error)
	GetDownloadContent(
		ctx context.Context,
		req documentports.GetDocumentByIDRequest,
	) (*documentports.DocumentContent, error)
	Delete(ctx context.Context, req documentports.DeleteDocumentRequest, userID pulid.ID) error
}

type replyDrafter interface {
	DraftReplyFor(
		ctx context.Context,
		message *inboundmessage.InboundMessage,
		mode triagenova.Mode,
	) (*triage.Reply, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Messages      inboundmessageports.InboundMessageRepository
	Mailboxes     inboundmessageports.InboundMailboxRepository
	Work          triageworkports.TriageWorkRepository
	Activities    inboundmessageports.TriageActivityRecorder
	Notifier      inboundmessageports.TriageMessageNotifier
	Shipments     inboundmessageports.InboundShipmentFinder
	DocumentRepo  documentports.DocumentRepository
	Documents     *documentservice.Service
	DocumentTypes documenttypeports.DocumentTypeRepository
	Uploads       documentuploadports.DocumentUploadService
	Imaging       kernelports.Imaging
	Replies       *inboundreplyservice.Service
	ReplyReader   inboundmessageports.InboundReplyReader
	Nova          *triagenova.Caller
	Workflows     kernelports.WorkflowStarter            `optional:"true"`
	Events        inboundmessageports.TriageEventSink    `optional:"true"`
	Loads         triageworkports.TriageLoadCreator      `optional:"true"`
	Exceptions    triageworkports.TriageExceptionLogger  `optional:"true"`
	Detention     triageworkports.TriageDetentionSettler `optional:"true"`
	CarrierPay    triageworkports.TriageCarrierPayQueuer `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	messages      inboundmessageports.InboundMessageRepository
	mailboxes     inboundmessageports.InboundMailboxRepository
	work          triageworkports.TriageWorkRepository
	activities    inboundmessageports.TriageActivityRecorder
	notifier      inboundmessageports.TriageMessageNotifier
	shipments     inboundmessageports.InboundShipmentFinder
	documentRepo  documentports.DocumentRepository
	documents     documentFiler
	documentTypes documenttypeports.DocumentTypeRepository
	uploads       documentuploadports.DocumentUploadService
	imaging       kernelports.Imaging
	replies       replyDrafter
	latest        inboundmessageports.InboundReplyReader
	nova          triagenova.Completer
	workflows     kernelports.WorkflowStarter
	events        inboundmessageports.TriageEventSink
	loads         triageworkports.TriageLoadCreator
	exceptions    triageworkports.TriageExceptionLogger
	detention     triageworkports.TriageDetentionSettler
	carrierPay    triageworkports.TriageCarrierPayQueuer
	now           func() int64
}

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		l:             p.Logger.Named("service.triage-work"),
		messages:      p.Messages,
		mailboxes:     p.Mailboxes,
		work:          p.Work,
		activities:    p.Activities,
		notifier:      p.Notifier,
		shipments:     p.Shipments,
		documentRepo:  p.DocumentRepo,
		documents:     p.Documents,
		documentTypes: p.DocumentTypes,
		uploads:       p.Uploads,
		imaging:       p.Imaging,
		replies:       p.Replies,
		latest:        p.ReplyReader,
		nova:          p.Nova,
		workflows:     p.Workflows,
		events:        p.Events,
		loads:         p.Loads,
		exceptions:    p.Exceptions,
		detention:     p.Detention,
		carrierPay:    p.CarrierPay,
		now:           timeutils.NowUnix,
	}
}

func (s *Service) message(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) (*inboundmessage.InboundMessage, error) {
	return s.messages.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:                 messageID,
		TenantInfo:         tenantInfo,
		IncludeAttachments: true,
	})
}

func (s *Service) mutate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
	change func(*inboundmessage.InboundMessage) error,
) (*inboundmessage.InboundMessage, error) {
	var lastErr error
	for range maxMessageWriteAttempts {
		message, err := s.message(ctx, tenantInfo, messageID)
		if err != nil {
			return nil, err
		}
		if err = change(message); err != nil {
			return nil, err
		}

		attachments := message.Attachments
		updated, err := s.messages.Update(ctx, message)
		if err == nil {
			updated.Attachments = attachments

			return updated, nil
		}
		if !errortypes.IsVersionMismatchError(err) {
			return nil, err
		}
		lastErr = err
	}

	return nil, lastErr
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

func (s *Service) changed(ctx context.Context, messages ...*inboundmessage.InboundMessage) {
	if len(messages) == 0 {
		return
	}
	s.notifier.MessagesChanged(ctx, messages...)
}

func personActivity(
	message *inboundmessage.InboundMessage,
	kind triage.ActivityKind,
	userID pulid.ID,
) *triage.Activity {
	activity := triage.NewActivity(message, kind)
	if userID.IsNotNil() {
		return activity.ByPerson(userID)
	}

	return activity.ByNova()
}

func tenantOf(message *inboundmessage.InboundMessage) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: message.OrganizationID, BuID: message.BusinessUnitID}
}
