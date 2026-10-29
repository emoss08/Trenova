package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/findingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ptodecisiontools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*operationstools.NotifyDriverTool)(nil)

var _ agenttoolports.ToolPreviewer = (*ptodecisiontools.RejectWorkerPTOTool)(nil)

var _ agenttoolports.ToolPreviewer = (*ptodecisiontools.CancelWorkerPTOTool)(nil)

var _ agenttoolports.ToolPreviewer = (*findingtools.RequestCredentialRenewalTool)(nil)
