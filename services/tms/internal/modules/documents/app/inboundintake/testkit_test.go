package inboundintake

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmailbox"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundprocessing"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"go.uber.org/zap"
)

type testService struct {
	l           *zap.Logger
	mailboxRepo inboundmessageports.InboundMailboxRepository
	messageRepo inboundmessageports.InboundMessageRepository
	storage     kernelports.Client
	encryption  inboundmailbox.SecretKeeper
	completion  llmports.CompletionService
	workflows   kernelports.WorkflowStarter
	shipments   inboundqueue.ShipmentFinder
	parties     inboundqueue.PartyFinder
	uploads     documentuploadports.DocumentUploadService
	documents   documentports.DocumentRepository
	watchtower  watchtowerports.WatchtowerProjector
	events      agentports.AgentEventPublisher
	audit       auditports.AuditService
	realtime    realtimeports.RealtimeService
	indexer     retrievalports.RetrievalIndexer
	fetcher     inboundmessageports.InboundContentFetcher

	db            kernelports.Transactor
	queueRepo     inboundmessageports.TriageQueueRepository
	mailboxState  inboundmessageports.TriageMailboxStateRepository
	activityRepo  inboundmessageports.TriageActivityRepository
	targetRepo    inboundmessageports.TriageTargetRepository
	settingsRepo  inboundmessageports.TriageSettingsRepository
	documentTypes inboundmessageports.InboundDocumentTypeFinder
	replies       inboundmessageports.InboundReplyReader
	ruleEvaluator inboundmessageports.TriageRuleEvaluator
	ruleTeacher   inboundmessageports.TriageRuleTeacher
	hints         inboundmessageports.SenderHintProvider
	wakes         inboundmessageports.TriageWakeScheduler
	email         emailports.EmailService
	notify        *inboundqueue.Notifier
	now           func() int64
}

func withQueue(
	svc *testService,
	states ...*inboundmessageports.QueueState,
) (*testService, *inboundqueuetest.QueueFakes) {
	fakes := inboundqueuetest.NewQueueFakes(states...)
	if svc.l == nil {
		svc.l = zap.NewNop()
	}
	svc.db = fakes.DB
	svc.queueRepo = fakes.Queue
	svc.activityRepo = fakes.Activities
	svc.mailboxState = fakes.Mailboxes
	svc.targetRepo = fakes.Targets
	svc.wakes = fakes.Wakes
	svc.notify = inboundqueue.NewNotifier(inboundqueue.NotifierParams{Logger: svc.l})
	svc.now = func() int64 { return inboundqueuetest.TestNow }

	return svc, fakes
}

func (t *testService) intake() *Service {
	logger := t.l
	if logger == nil {
		logger = zap.NewNop()
	}
	queue := inboundqueue.New(inboundqueue.Params{
		ActivityRepo:  t.activityRepo,
		DB:            t.db,
		DocumentTypes: t.documentTypes,
		Logger:        logger,
		MessageRepo:   t.messageRepo,
		Now:           t.now,
		Parties:       t.parties,
		QueueRepo:     t.queueRepo,
		Realtime:      t.realtime,
		RuleTeacher:   t.ruleTeacher,
		SettingsRepo:  t.settingsRepo,
		Shipments:     t.shipments,
		TargetRepo:    t.targetRepo,
		Wakes:         t.wakes,
		Watchtower:    t.watchtower,
	})
	mailbox := inboundmailbox.New(inboundmailbox.Params{
		Audit:        t.audit,
		Email:        t.email,
		Encryption:   t.encryption,
		Logger:       logger,
		MailboxRepo:  t.mailboxRepo,
		MailboxState: t.mailboxState,
		Now:          t.now,
	})
	processing := inboundprocessing.New(inboundprocessing.Params{
		ActivityRepo:  t.activityRepo,
		Completion:    t.completion,
		Events:        t.events,
		Hints:         t.hints,
		Indexer:       t.indexer,
		Logger:        logger,
		MessageRepo:   t.messageRepo,
		Parties:       t.parties,
		Queue:         queue,
		Realtime:      t.realtime,
		RuleEvaluator: t.ruleEvaluator,
		Shipments:     t.shipments,
		Watchtower:    t.watchtower,
	})
	return &Service{
		activityRepo: t.activityRepo,
		documents:    t.documents,
		fetcher:      t.fetcher,
		l:            t.l,
		mailboxRepo:  t.mailboxRepo,
		mailboxState: t.mailboxState,
		messageRepo:  t.messageRepo,
		notify:       t.notify,
		now:          t.now,
		queueRepo:    t.queueRepo,
		replies:      t.replies,
		storage:      t.storage,
		uploads:      t.uploads,
		workflows:    t.workflows,
		mailbox:      mailbox,
		processing:   processing,
		queue:        queue,
	}
}
