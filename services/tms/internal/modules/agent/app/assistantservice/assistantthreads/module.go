package assistantthreads

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"go.uber.org/fx"
)

var Module = fx.Module("assistant-threads",
	fx.Provide(fx.Annotate(
		New,
		fx.As(new(assistantports.AssistantService)),
		fx.As(new(assistantports.PageAssistant)),
	)),
)
