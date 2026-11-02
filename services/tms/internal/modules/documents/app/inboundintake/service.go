package inboundintake

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmailbox"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundprocessing"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	ActivityRepo inboundmessageports.TriageActivityRepository
	Briefing     inboundmessageports.TriageBriefingInvalidator
	// ContentFetcher reads what a metadata-only webhook leaves out.
	ContentFetcher inboundmessageports.InboundContentFetcher
	Documents      documentports.DocumentRepository
	Logger         *zap.Logger
	Mailbox        *inboundmailbox.Service
	MailboxRepo    inboundmessageports.InboundMailboxRepository
	MailboxState   inboundmessageports.TriageMailboxStateRepository
	MessageRepo    inboundmessageports.InboundMessageRepository
	Now            timeutils.Clock
	Processing     *inboundprocessing.Service
	Queue          *inboundqueue.Service
	QueueRepo      inboundmessageports.TriageQueueRepository
	// Realtime keeps an open inbox current, and Notifications tells the people
	// who read the inbox that a message is waiting on them.
	Realtime realtimeports.RealtimeService
	Replies  inboundmessageports.InboundReplyReader
	Storage  kernelports.Client
	// Uploads and Documents carry attachments through the document pipeline.
	Uploads documentuploadports.DocumentUploadService
	// Watchtower and Events are how a message that needs a person, or a desk,
	// gets one.
	Watchtower watchtowerports.WatchtowerProjector
	// It lands at Received and waits, which is what a mailbox on AlwaysReview does with every
	// message anyway.
	Workflows kernelports.WorkflowStarter
}

type Service struct {
	activityRepo inboundmessageports.TriageActivityRepository
	documents    documentports.DocumentRepository
	fetcher      inboundmessageports.InboundContentFetcher
	l            *zap.Logger
	mailboxRepo  inboundmessageports.InboundMailboxRepository
	mailboxState inboundmessageports.TriageMailboxStateRepository
	messageRepo  inboundmessageports.InboundMessageRepository
	notify       *inboundqueue.Notifier
	now          func() int64
	queueRepo    inboundmessageports.TriageQueueRepository
	replies      inboundmessageports.InboundReplyReader
	storage      kernelports.Client
	uploads      documentuploadports.DocumentUploadService
	workflows    kernelports.WorkflowStarter
	mailbox      *inboundmailbox.Service
	processing   *inboundprocessing.Service
	queue        *inboundqueue.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	now := p.Now
	return &Service{
		activityRepo: p.ActivityRepo,
		documents:    p.Documents,
		fetcher:      p.ContentFetcher,
		l:            p.Logger.Named("service.inbound-message"),
		mailboxRepo:  p.MailboxRepo,
		mailboxState: p.MailboxState,
		messageRepo:  p.MessageRepo,
		notify: inboundqueue.NewNotifier(inboundqueue.NotifierParams{
			Logger:     p.Logger,
			Realtime:   p.Realtime,
			Watchtower: p.Watchtower,
			Briefing:   p.Briefing,
		}),
		now:        now,
		queueRepo:  p.QueueRepo,
		replies:    p.Replies,
		storage:    p.Storage,
		uploads:    p.Uploads,
		workflows:  p.Workflows,
		mailbox:    p.Mailbox,
		processing: p.Processing,
		queue:      p.Queue,
	}
}
