package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*carriertools.CarrierInvoiceMatchTool)(nil)

var _ agentports.ToolValidator = (*carriertools.CarrierInvoiceMatchTool)(nil)

var _ agentports.TargetedTool = (*carriertools.CarrierInvoiceMatchTool)(nil)

var _ agentports.ToolPreviewer = (*carriertools.CreateCarrierInvoiceMatchTool)(nil)

var _ agentports.ToolValidator = (*carriertools.CreateCarrierInvoiceMatchTool)(nil)

var _ agentports.ToolPreviewer = (*carriertools.LinkEDICarrierInvoiceTool)(nil)

var _ agentports.ToolValidator = (*carriertools.LinkEDICarrierInvoiceTool)(nil)
