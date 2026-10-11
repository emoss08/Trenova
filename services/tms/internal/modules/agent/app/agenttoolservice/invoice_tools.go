package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*invoicetools.PostInvoiceTool)(nil)

var _ agentports.ToolValidator = (*invoicetools.PostInvoiceTool)(nil)

var _ agentports.TargetedTool = (*invoicetools.PostInvoiceTool)(nil)

var _ agentports.ToolPreviewer = (*invoicetools.SendInvoiceTool)(nil)

var _ agentports.ToolValidator = (*invoicetools.SendInvoiceTool)(nil)

var _ agentports.TargetedTool = (*invoicetools.SendInvoiceTool)(nil)
