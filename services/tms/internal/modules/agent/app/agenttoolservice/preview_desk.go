package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*masterdatatools.EscalateDetentionTool)(nil)

var _ agentports.ToolPreviewer = (*masterdatatools.PlaceWorkerDispatchHoldTool)(nil)

var _ agentports.ToolPreviewer = (*masterdatatools.AcknowledgeCarrierIntelEventTool)(nil)

var _ agentports.ToolPreviewer = (*masterdatatools.ResolveCarrierIntelEventTool)(nil)
