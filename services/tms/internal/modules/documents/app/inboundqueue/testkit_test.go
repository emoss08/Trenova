package inboundqueue

import (
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/zap"
)

type testService struct {
	l           *zap.Logger
	messageRepo inboundmessageports.InboundMessageRepository
	shipments   ShipmentFinder
	parties     PartyFinder

	db            kernelports.Transactor
	queueRepo     inboundmessageports.TriageQueueRepository
	mailboxState  inboundmessageports.TriageMailboxStateRepository
	activityRepo  inboundmessageports.TriageActivityRepository
	targetRepo    inboundmessageports.TriageTargetRepository
	settingsRepo  inboundmessageports.TriageSettingsRepository
	documentTypes inboundmessageports.InboundDocumentTypeFinder
	directory     assigneeDirectory
	organizations organizationReader
	ruleTeacher   inboundmessageports.TriageRuleTeacher
	wakes         inboundmessageports.TriageWakeScheduler
	notify        *Notifier
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
	svc.notify = NewNotifier(NotifierParams{Logger: svc.l})
	svc.now = func() int64 { return inboundqueuetest.TestNow }

	return svc, fakes
}

func (t *testService) queue() *Service {
	return &Service{
		activityRepo:  t.activityRepo,
		db:            t.db,
		directory:     t.directory,
		documentTypes: t.documentTypes,
		l:             t.l,
		messageRepo:   t.messageRepo,
		notify:        t.notify,
		now:           t.now,
		organizations: t.organizations,
		parties:       t.parties,
		queueRepo:     t.queueRepo,
		ruleTeacher:   t.ruleTeacher,
		settingsRepo:  t.settingsRepo,
		shipments:     t.shipments,
		targetRepo:    t.targetRepo,
		wakes:         t.wakes,
	}
}
