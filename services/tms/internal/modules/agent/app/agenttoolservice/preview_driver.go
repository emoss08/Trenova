package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*operationstools.NotifyDriverTool)(nil)

var _ agentports.ToolPreviewer = (*operationstools.RejectWorkerPTOTool)(nil)

var _ agentports.ToolPreviewer = (*operationstools.CancelWorkerPTOTool)(nil)

var _ agentports.ToolPreviewer = (*masterdatatools.RequestCredentialRenewalTool)(nil)
