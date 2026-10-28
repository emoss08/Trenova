package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*masterdatatools.EscalateDetentionTool)(nil)

var _ agenttoolports.ToolPreviewer = (*masterdatatools.PlaceWorkerDispatchHoldTool)(nil)

var _ agenttoolports.ToolPreviewer = (*masterdatatools.AcknowledgeCarrierIntelEventTool)(nil)

var _ agenttoolports.ToolPreviewer = (*masterdatatools.ResolveCarrierIntelEventTool)(nil)
