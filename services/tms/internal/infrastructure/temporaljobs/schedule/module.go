package schedule

import "go.uber.org/fx"

var Module = fx.Module("schedule",
	fx.Provide(NewScheduler),
)

func AsProvider(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Provider)),
		fx.ResultTags(`group:"schedule_providers"`),
	)
}
