package supportaccess

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

func PermissionOptions() fx.Option {
	return fx.Provide(
		NewPermissionSource,
		fx.Annotate(
			func(source *PermissionSource) *PermissionSource { return source },
			fx.As(new(services.DelegatedPermissionSource)),
		),
	)
}
