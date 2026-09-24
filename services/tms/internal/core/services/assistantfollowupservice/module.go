package assistantfollowupservice

import "go.uber.org/fx"

var Module = fx.Module("assistantfollowupservice",
	fx.Provide(New, NewResumer),
)
