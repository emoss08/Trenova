package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/findingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*findingtools.EscalateDetentionTool)(nil)

var _ agenttoolports.ToolPreviewer = (*findingtools.PlaceWorkerDispatchHoldTool)(nil)

var _ agenttoolports.ToolPreviewer = (*findingtools.AcknowledgeCarrierIntelEventTool)(nil)

var _ agenttoolports.ToolPreviewer = (*findingtools.ResolveCarrierIntelEventTool)(nil)
