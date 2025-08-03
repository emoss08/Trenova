/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
