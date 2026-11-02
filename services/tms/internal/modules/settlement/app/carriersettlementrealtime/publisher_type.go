package carriersettlementrealtime

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"go.uber.org/zap"
)

type Publisher struct {
	l        *zap.Logger
	realtime realtimeports.RealtimeService
}

// NewPublisher is the settlement module's one carrier settlement publisher,
// shared by every service that announces a settlement change.
func NewPublisher(l *zap.Logger, realtime realtimeports.RealtimeService) Publisher {
	return Publisher{l: l.Named("carrier-settlement-realtime"), realtime: realtime}
}
