package inboundupkeep

import (
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/zap"
)

type testService struct {
	l           *zap.Logger
	messageRepo inboundmessageports.InboundMessageRepository
	storage     kernelports.Client
	remittances inboundmessageports.RemittanceIntake

	notify *inboundqueue.Notifier
}

func (t *testService) upkeep() *Service {
	return &Service{
		l:           t.l,
		messageRepo: t.messageRepo,
		notify:      t.notify,
		remittances: t.remittances,
		storage:     t.storage,
	}
}
