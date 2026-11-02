package inboundreads

import (
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	MailboxRepo inboundmessageports.InboundMailboxRepository
	MessageRepo inboundmessageports.InboundMessageRepository
	Now         timeutils.Clock
	QueueRepo   inboundmessageports.TriageQueueRepository
}

type Service struct {
	mailboxRepo inboundmessageports.InboundMailboxRepository
	messageRepo inboundmessageports.InboundMessageRepository
	now         func() int64
	queueRepo   inboundmessageports.TriageQueueRepository
}

func New(p Params) *Service {
	now := p.Now
	return &Service{
		mailboxRepo: p.MailboxRepo,
		messageRepo: p.MessageRepo,
		now:         now,
		queueRepo:   p.QueueRepo,
	}
}
