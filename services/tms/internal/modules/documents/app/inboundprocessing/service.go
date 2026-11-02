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
	Briefing     inboundmessageports.TriageBriefingInvalidator
	// Mail lands and waits for a person instead of being classified, which is the behaviour a
	// mailbox on AlwaysReview has anyway.
	Nova          llmports.TaskRunner
	Events        agenteventports.AgentEventPublisher
	Hints         inboundmessageports.SenderHintProvider
	Indexer       retrievalports.RetrievalIndexer
	Logger        *zap.Logger
	MessageRepo   inboundmessageports.InboundMessageRepository
	Notifications *notificationservice.Service
	Parties       inboundqueue.PartyFinder
	Queue         *inboundqueue.Service
	// Realtime keeps an open inbox current, and Notifications tells the people
	// who read the inbox that a message is waiting on them.
	Realtime      realtimeports.RealtimeService
	RuleEvaluator inboundmessageports.TriageRuleEvaluator
	// Shipments and Parties resolve what a message is about. Both are optional:
	// an unmatched message is still one a person can read and link by hand.
	Shipments inboundqueue.ShipmentFinder
	// Watchtower and Events are how a message that needs a person, or a desk,
	// gets one.
	Watchtower watchtowerports.WatchtowerProjector
}

type Service struct {
	activityRepo  inboundmessageports.TriageActivityRepository
	nova          llmports.TaskRunner
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
	notifier = p.Notifications
	return &Service{
		activityRepo: p.ActivityRepo,
		nova:         p.Nova,
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
