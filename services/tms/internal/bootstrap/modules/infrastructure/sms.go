package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/sms"
	"go.uber.org/fx"
)

var SMSModule = fx.Module("sms", fx.Provide(sms.New))
