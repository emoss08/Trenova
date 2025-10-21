package emailjobs

import (
	"go.uber.org/fx"
)

var Module = fx.Module("temporal:emailjobs",
	fx.Provide(NewActivities),
	fx.Provide(NewRegistry),
)
