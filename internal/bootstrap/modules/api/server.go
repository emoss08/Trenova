package api

import (
	"github.com/emoss08/trenova/internal/api/server"
	"go.uber.org/fx"
)

var ServerModule = fx.Module("api.Server", fx.Provide(server.NewServer))
