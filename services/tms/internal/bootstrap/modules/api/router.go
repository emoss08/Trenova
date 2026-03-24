package api

import (
	"github.com/emoss08/trenova/internal/api"
	"go.uber.org/fx"
)

func RegisterRoutes(router *api.Router) {
	router.Setup()
}

var RouterModule = fx.Module("api-router", fx.Provide(api.NewRouter), fx.Invoke(RegisterRoutes))
