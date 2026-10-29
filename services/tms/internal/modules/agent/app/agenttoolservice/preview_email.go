package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/qualificationtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*operationstools.EmailCustomerTool)(nil)

var _ agenttoolports.ToolPreviewer = (*qualificationtools.RequestMissingDocsTool)(nil)
