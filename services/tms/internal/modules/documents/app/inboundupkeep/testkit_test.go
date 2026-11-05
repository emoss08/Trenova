package inboundupkeep

import (
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"go.uber.org/zap"
)

type testService struct {
	l           *zap.Logger
	messageRepo inboundmessageports.InboundMessageRepository
	storage     kernelports.Client
	remittances inboundmessageports.RemittanceIntake

	notify *inboundqueue.Notifier
}

func discardingNotifier() *inboundqueue.Notifier {
	return inboundqueue.NewNotifier(inboundqueue.NotifierParams{
		Logger:     zap.NewNop(),
		Realtime:   &inboundqueuetest.RecordingRealtime{},
		Watchtower: &mocks.NoopWatchtowerProjector{},
		Briefing:   inboundqueuetest.NoopBriefing{},
	})
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
