package agentwaitservice

import "go.uber.org/fx"

var Module = fx.Module("agentwaitservice",
	fx.Provide(New, NewWaitService, NewNotifier, NewWorker),
)
