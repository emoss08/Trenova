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
	Now         func() int64 `optional:"true"`
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
	if now == nil {
		now = timeutils.NowUnix
	}
	return &Service{
		mailboxRepo: p.MailboxRepo,
		messageRepo: p.MessageRepo,
		now:         now,
		queueRepo:   p.QueueRepo,
	}
}
