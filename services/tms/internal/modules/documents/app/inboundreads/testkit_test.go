package inboundreads

import "github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"

type testService struct {
	mailboxRepo inboundmessageports.InboundMailboxRepository
	messageRepo inboundmessageports.InboundMessageRepository

	queueRepo inboundmessageports.TriageQueueRepository
	now       func() int64
}

func (t *testService) reads() *Service {
	return &Service{
		mailboxRepo: t.mailboxRepo,
		messageRepo: t.messageRepo,
		now:         t.now,
		queueRepo:   t.queueRepo,
	}
}
