package carriersettlementrealtime

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"go.uber.org/zap"
)

type Publisher struct {
	l        *zap.Logger
	realtime realtimeports.RealtimeService
}

func NewPublisher(l *zap.Logger, realtime realtimeports.RealtimeService) Publisher {
	return Publisher{l: l, realtime: realtime}
}
