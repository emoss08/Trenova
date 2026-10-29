package assistantartifacts

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"go.uber.org/fx"
)

var Module = fx.Module("assistant-artifacts",
	fx.Provide(fx.Annotate(New, fx.As(new(assistantports.AssistantArtifactService)))),
)
