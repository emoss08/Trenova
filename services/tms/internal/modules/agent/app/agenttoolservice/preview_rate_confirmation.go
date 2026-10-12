package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ratetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*ratetools.GenerateRateConfirmationTool)(nil)

var _ agentports.ToolPreviewer = (*ratetools.SendRateConfirmationTool)(nil)

var _ agentports.ToolPreviewer = (*ratetools.VoidRateConfirmationTool)(nil)

var _ agentports.ToolPreviewer = (*ratetools.RecordRateConfirmationConfirmedTool)(nil)

var _ agentports.ToolValidator = (*ratetools.GenerateRateConfirmationTool)(nil)

var _ agentports.ToolValidator = (*ratetools.SendRateConfirmationTool)(nil)

var _ agentports.ToolValidator = (*ratetools.VoidRateConfirmationTool)(nil)

var _ agentports.ToolValidator = (*ratetools.RecordRateConfirmationConfirmedTool)(nil)
