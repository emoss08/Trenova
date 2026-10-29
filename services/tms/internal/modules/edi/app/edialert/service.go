package edialert

import (
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Notifications *notificationservice.Service
}

type Service struct {
	l             *zap.Logger
	notifications *notificationservice.Service
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.edi"),
		notifications: p.Notifications,
	}
}
