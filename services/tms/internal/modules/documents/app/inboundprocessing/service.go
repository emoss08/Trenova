package inboundprocessing

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	ActivityRepo inboundmessageports.TriageActivityRepository
	Briefing     inboundmessageports.TriageBriefingInvalidator `optional:"true"`
	// Completion is optional so the ingest path still works on an installation
	// with no provider configured. Mail lands and waits for a person instead of
	// being classified, which is the behaviour a mailbox on AlwaysReview has
	// anyway.
	Completion    llmports.CompletionService             `optional:"true"`
	Events        agenteventports.AgentEventPublisher    `optional:"true"`
	Hints         inboundmessageports.SenderHintProvider `optional:"true"`
	Indexer       retrievalports.RetrievalIndexer        `optional:"true"`
	Logger        *zap.Logger
	MessageRepo   inboundmessageports.InboundMessageRepository
	Notifications *notificationservice.Service `optional:"true"`
	Parties       inboundqueue.PartyFinder     `optional:"true"`
	Queue         *inboundqueue.Service
	// Realtime keeps an open inbox current, and Notifications tells the people
	// who read the inbox that a message is waiting on them. Both optional: the
	// inbox is still right on the next load without either.
	Realtime      realtimeports.RealtimeService           `optional:"true"`
	RuleEvaluator inboundmessageports.TriageRuleEvaluator `optional:"true"`
	// Shipments and Parties resolve what a message is about. Both are optional:
	// an unmatched message is still one a person can read and link by hand.
	Shipments inboundqueue.ShipmentFinder `optional:"true"`
	// Watchtower and Events are how a message that needs a person, or a desk,
	// gets one. Both are optional: without them the inbox still holds every
	// message and its reason, and nothing else is told about it.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
}

type Service struct {
	activityRepo  inboundmessageports.TriageActivityRepository
	completion    llmports.CompletionService
	events        agenteventports.AgentEventPublisher
	hints         inboundmessageports.SenderHintProvider
	indexer       retrievalports.RetrievalIndexer
	l             *zap.Logger
	messageRepo   inboundmessageports.InboundMessageRepository
	notifier      reviewNotifier
	notify        *inboundqueue.Notifier
	parties       inboundqueue.PartyFinder
	ruleEvaluator inboundmessageports.TriageRuleEvaluator
	shipments     inboundqueue.ShipmentFinder
	queue         *inboundqueue.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	var notifier reviewNotifier
	if p.Notifications != nil {
		notifier = p.Notifications
	}
	return &Service{
		activityRepo: p.ActivityRepo,
		completion:   p.Completion,
		events:       p.Events,
		hints:        p.Hints,
		indexer:      p.Indexer,
		l:            p.Logger.Named("service.inbound-message"),
		messageRepo:  p.MessageRepo,
		notifier:     notifier,
		notify: inboundqueue.NewNotifier(inboundqueue.NotifierParams{
			Logger:     p.Logger,
			Realtime:   p.Realtime,
			Watchtower: p.Watchtower,
			Briefing:   p.Briefing,
		}),
		parties:       p.Parties,
		ruleEvaluator: p.RuleEvaluator,
		shipments:     p.Shipments,
		queue:         p.Queue,
	}
}

// reviewNotifier is the slice of the notification service the inbox uses.
type reviewNotifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
}
