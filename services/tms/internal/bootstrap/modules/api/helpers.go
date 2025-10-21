package api

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"go.uber.org/fx"
)

var HelpersModule = fx.Module("api-helpers", fx.Provide(
	helpers.NewErrorHandler,
))
