package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/documenttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*documenttools.AttachDocumentTool)(nil)
