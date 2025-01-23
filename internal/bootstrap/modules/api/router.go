package api

import (
	"github.com/emoss08/trenova/internal/api/routes"
	"go.uber.org/fx"
)

func RegisterRoutes(router *routes.Router) {
	router.Setup()
}

var RouterModule = fx.Module("api.Router",
	fx.Provide(routes.NewRouter),
	fx.Invoke(RegisterRoutes),
)
