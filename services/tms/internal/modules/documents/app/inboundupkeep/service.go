package inboundupkeep

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Briefing    inboundmessageports.TriageBriefingInvalidator `optional:"true"`
	Logger      *zap.Logger
	MessageRepo inboundmessageports.InboundMessageRepository
	// Realtime keeps an open inbox current, and Notifications tells the people
	// who read the inbox that a message is waiting on them. Both optional: the
	// inbox is still right on the next load without either.
	Realtime realtimeports.RealtimeService `optional:"true"`
	// Remittances opens a factor's remittance from a FactorRemittance message.
	// Billing provides it; without it those messages wait in the inbox.
	Remittances inboundmessageports.RemittanceIntake `optional:"true"`
	Storage     kernelports.Client
	// Watchtower and Events are how a message that needs a person, or a desk,
	// gets one. Both are optional: without them the inbox still holds every
	// message and its reason, and nothing else is told about it.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	messageRepo inboundmessageports.InboundMessageRepository
	notify      *inboundqueue.Notifier
	remittances inboundmessageports.RemittanceIntake
	storage     kernelports.Client
}

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.inbound-message"),
		messageRepo: p.MessageRepo,
		notify: inboundqueue.NewNotifier(inboundqueue.NotifierParams{
			Logger:     p.Logger,
			Realtime:   p.Realtime,
			Watchtower: p.Watchtower,
			Briefing:   p.Briefing,
		}),
		remittances: p.Remittances,
		storage:     p.Storage,
	}
}
