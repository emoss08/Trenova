package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*invoicetools.PostInvoicesTool)(nil)

var _ agentports.ToolValidator = (*invoicetools.PostInvoicesTool)(nil)

var _ agentports.ToolResultReporter = (*invoicetools.PostInvoicesTool)(nil)

var _ agentports.ToolPreviewer = (*invoicetools.SendInvoicesTool)(nil)

var _ agentports.ToolValidator = (*invoicetools.SendInvoicesTool)(nil)

var _ agentports.ToolResultReporter = (*invoicetools.SendInvoicesTool)(nil)

var _ agentports.ToolPreviewer = (*invoicetools.ApproveBillingQueueItemsTool)(nil)

var _ agentports.ToolValidator = (*invoicetools.ApproveBillingQueueItemsTool)(nil)

var _ agentports.ToolResultReporter = (*invoicetools.ApproveBillingQueueItemsTool)(nil)
