package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/equipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*equipmenttools.UpdateTractorStatusTool)(nil)

var _ agenttoolports.ToolPreviewer = (*equipmenttools.UpdateTrailerStatusTool)(nil)
