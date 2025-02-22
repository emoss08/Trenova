package statemachine

import "go.uber.org/fx"

var Module = fx.Module("statemachine", fx.Provide(
	NewStateMachineManager,
))
