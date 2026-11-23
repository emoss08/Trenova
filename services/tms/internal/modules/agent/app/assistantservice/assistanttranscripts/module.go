package assistanttranscripts

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"go.uber.org/fx"
)

var Module = fx.Module("assistant-transcripts",
	fx.Provide(fx.Annotate(
		New,
		fx.As(new(assistantports.AssistantTranscriptService)),
		fx.As(new(agentruntimeports.AgentRunTranscriptService)),
		fx.As(new(desktranscriptports.TranscriptRenderer)),
	)),
)
