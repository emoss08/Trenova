package api

import (
	"github.com/trenova-app/transport/internal/api/routes"
	"go.uber.org/fx"
)

func RegisterRoutes(router *routes.Router) {
	router.Setup()
}

var RouterModule = fx.Module("api.Router",
	fx.Provide(routes.NewRouter),
	fx.Invoke(RegisterRoutes),
)
