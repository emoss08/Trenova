package inboundprocessing

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"go.uber.org/zap"
)

type testService struct {
	l           *zap.Logger
	messageRepo inboundmessageports.InboundMessageRepository
	completion  llmports.TaskRunner
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
	dues          inboundmessageports.TriageDueRecomputer
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
	svc.dues = fakes.Dues
	svc.notify = inboundqueue.NewNotifier(inboundqueue.NotifierParams{
		Logger:     svc.l,
		Realtime:   &inboundqueuetest.RecordingRealtime{},
		Watchtower: &mocks.NoopWatchtowerProjector{},
		Briefing:   inboundqueuetest.NoopBriefing{},
	})
	svc.now = func() int64 { return inboundqueuetest.TestNow }

	return svc, fakes
}

type noRules struct{}

func (noRules) ApplyRules(
	context.Context,
	*inboundmessage.InboundMessage,
) (*inboundmessageports.RuleApplication, error) {
	return nil, nil
}

func (t *testService) supplyDefaults() {
	if t.realtime == nil {
		t.realtime = &inboundqueuetest.RecordingRealtime{}
	}
	if t.watchtower == nil {
		t.watchtower = &mocks.NoopWatchtowerProjector{}
	}
	if t.hints == nil {
		t.hints = inboundqueuetest.NoSenderHints{}
	}
	if t.parties == nil {
		t.parties = &inboundqueuetest.StubParties{}
	}
	if t.shipments == nil {
		t.shipments = &inboundqueuetest.StubShipments{}
	}
	if t.notifier == nil {
		t.notifier = &recordingNotifier{}
	}
	if t.indexer == nil {
		t.indexer = &fakeIndexer{}
	}
	if t.ruleEvaluator == nil {
		t.ruleEvaluator = noRules{}
	}
}

func (t *testService) processing() *Service {
	t.supplyDefaults()
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
		Dues:          t.dues,
		Watchtower:    t.watchtower,
		Briefing:      inboundqueuetest.NoopBriefing{},
	})
	return &Service{
		activityRepo:  t.activityRepo,
		nova:          t.completion,
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
	t.supplyDefaults()
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
		Dues:          t.dues,
		Watchtower:    t.watchtower,
		Briefing:      inboundqueuetest.NoopBriefing{},
	})
	return queue
}
