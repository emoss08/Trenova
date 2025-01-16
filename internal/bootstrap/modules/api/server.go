package api

import (
	"github.com/trenova-app/transport/internal/api/server"
	"go.uber.org/fx"
)

var ServerModule = fx.Module("api.Server", fx.Provide(server.NewServer))
