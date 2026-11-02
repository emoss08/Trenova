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

	Briefing    inboundmessageports.TriageBriefingInvalidator
	Logger      *zap.Logger
	MessageRepo inboundmessageports.InboundMessageRepository
	// Realtime keeps an open inbox current, and Notifications tells the people
	// who read the inbox that a message is waiting on them.
	Realtime realtimeports.RealtimeService
	// Remittances opens a factor's remittance from a FactorRemittance message. Billing provides it.
	Remittances inboundmessageports.RemittanceIntake
	Storage     kernelports.Client
	// Watchtower and Events are how a message that needs a person, or a desk,
	// gets one.
	Watchtower watchtowerports.WatchtowerProjector
}

type Service struct {
	l           *zap.Logger
	messageRepo inboundmessageports.InboundMessageRepository
	notify      *inboundqueue.Notifier
	remittances inboundmessageports.RemittanceIntake
	storage     kernelports.Client
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
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
