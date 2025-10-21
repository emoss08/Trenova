package statemanager

import (
	"github.com/emoss08/trenova/pkg/statemachine"
	"go.uber.org/fx"
)

var Module = fx.Module("statemanager", fx.Provide(
	statemachine.NewManager,
))
