package inboundprocessing

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/zap"
)

type testService struct {
	l           *zap.Logger
	messageRepo inboundmessageports.InboundMessageRepository
	completion  llmports.CompletionService
	shipments   inboundqueue.ShipmentFinder
	parties     inboundqueue.PartyFinder
	watchtower  watchtowerports.WatchtowerProjector
	events      agenteventports.AgentEventPublisher
	realtime    realtimeports.RealtimeService
	notifier    reviewNotifier
	indexer     retrievalports.RetrievalIndexer

	db            kernelports.Transactor
	queueRepo     inboundmessageports.TriageQueueRepository
	mailboxState  inboundmessageports.TriageMailboxStateRepository
	activityRepo  inboundmessageports.TriageActivityRepository
	targetRepo    inboundmessageports.TriageTargetRepository
	settingsRepo  inboundmessageports.TriageSettingsRepository
	documentTypes inboundmessageports.InboundDocumentTypeFinder
	ruleEvaluator inboundmessageports.TriageRuleEvaluator
	ruleTeacher   inboundmessageports.TriageRuleTeacher
	hints         inboundmessageports.SenderHintProvider
	wakes         inboundmessageports.TriageWakeScheduler
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

func (t *testService) processing() *Service {
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
	return &Service{
		activityRepo:  t.activityRepo,
		completion:    t.completion,
		events:        t.events,
		hints:         t.hints,
		indexer:       t.indexer,
		l:             t.l,
		messageRepo:   t.messageRepo,
		notifier:      t.notifier,
		notify:        t.notify,
		parties:       t.parties,
		ruleEvaluator: t.ruleEvaluator,
		shipments:     t.shipments,
		queue:         queue,
	}
}

func (t *testService) queue() *inboundqueue.Service {
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
	return queue
}
