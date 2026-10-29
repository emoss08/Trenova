package inboundqueue

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	ActivityRepo  inboundmessageports.TriageActivityRepository
	Briefing      inboundmessageports.TriageBriefingInvalidator `optional:"true"`
	DB            kernelports.Transactor
	DocumentTypes inboundmessageports.InboundDocumentTypeFinder
	Logger        *zap.Logger
	MessageRepo   inboundmessageports.InboundMessageRepository
	Now           func() int64 `optional:"true"`
	Organizations tenantports.OrganizationRepository
	Parties       PartyFinder `optional:"true"`
	QueueRepo     inboundmessageports.TriageQueueRepository
	// Realtime keeps an open inbox current, and Notifications tells the people
	// who read the inbox that a message is waiting on them. Both optional: the
	// inbox is still right on the next load without either.
	Realtime     realtimeports.RealtimeService `optional:"true"`
	Roles        permissionports.RoleRepository
	RuleTeacher  inboundmessageports.TriageRuleTeacher `optional:"true"`
	SettingsRepo inboundmessageports.TriageSettingsRepository
	// Shipments and Parties resolve what a message is about. Both are optional:
	// an unmatched message is still one a person can read and link by hand.
	Shipments  ShipmentFinder `optional:"true"`
	TargetRepo inboundmessageports.TriageTargetRepository
	Wakes      inboundmessageports.TriageWakeScheduler `optional:"true"`
	// Watchtower and Events are how a message that needs a person, or a desk,
	// gets one. Both are optional: without them the inbox still holds every
	// message and its reason, and nothing else is told about it.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
}

type Service struct {
	activityRepo  inboundmessageports.TriageActivityRepository
	db            kernelports.Transactor
	directory     assigneeDirectory
	documentTypes inboundmessageports.InboundDocumentTypeFinder
	l             *zap.Logger
	messageRepo   inboundmessageports.InboundMessageRepository
	notify        *Notifier
	now           func() int64
	organizations organizationReader
	parties       PartyFinder
	queueRepo     inboundmessageports.TriageQueueRepository
	ruleTeacher   inboundmessageports.TriageRuleTeacher
	settingsRepo  inboundmessageports.TriageSettingsRepository
	shipments     ShipmentFinder
	targetRepo    inboundmessageports.TriageTargetRepository
	wakes         inboundmessageports.TriageWakeScheduler
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	now := p.Now
	if now == nil {
		now = timeutils.NowUnix
	}
	return &Service{
		activityRepo:  p.ActivityRepo,
		db:            p.DB,
		directory:     p.Roles,
		documentTypes: p.DocumentTypes,
		l:             p.Logger.Named("service.inbound-message"),
		messageRepo:   p.MessageRepo,
		notify: NewNotifier(NotifierParams{
			Logger:     p.Logger,
			Realtime:   p.Realtime,
			Watchtower: p.Watchtower,
			Briefing:   p.Briefing,
		}),
		now:           now,
		organizations: p.Organizations,
		parties:       p.Parties,
		queueRepo:     p.QueueRepo,
		ruleTeacher:   p.RuleTeacher,
		settingsRepo:  p.SettingsRepo,
		shipments:     p.Shipments,
		targetRepo:    p.TargetRepo,
		wakes:         p.Wakes,
	}
}
