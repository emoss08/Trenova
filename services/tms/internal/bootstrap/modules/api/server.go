package api

import (
	"github.com/emoss08/trenova/internal/api"
	"go.uber.org/fx"
)

var ServerModule = fx.Module("api-server", fx.Provide(api.NewServer))
