package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytermtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementgenerationtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementtools"
)

func settlementToolProviders() []any {
	return append(
		append(
			settlementtools.SettlementDecisionProviders(),
			settlementtools.SettlementBulkProviders()...),
		paytools.ProvideHoldDriverPayEventTool,
		paytools.ProvideReleaseDriverPayEventTool,
		paytools.ProvideAttachPayEventsTool,
		paytools.ProvideDetachPayEventTool,
		settlementgenerationtools.ProvideGenerateDriverSettlementTool,
		settlementgenerationtools.ProvideGenerateDriverSettlementBatchTool,
		settlementgenerationtools.ProvideGenerateCarrierSettlementBatchTool,
		paytools.ProvidePayWorkerNowTool,
		paytools.ProvideIssuePayAdvanceTool,
		paytools.ProvideWriteOffPayAdvanceTool,
		paytermtools.ProvideOpenEscrowAccountTool,
		paytermtools.ProvideUpdateEscrowAccountTool,
		paytermtools.ProvideAdjustEscrowAccountTool,
		paytermtools.ProvideCloseEscrowAccountTool,
		paytools.ProvideAssignPayProfileTool,
		paytools.ProvideEndPayAssignmentTool,
		paytermtools.ProvideCreateRecurringDeductionTool,
		paytermtools.ProvideUpdateRecurringDeductionTool,
		paytermtools.ProvideCreateRecurringEarningTool,
		paytermtools.ProvideUpdateRecurringEarningTool,
		carriertools.ProvideCreateCarrierInvoiceMatchTool,
		carriertools.ProvideAcceptCarrierInvoiceMatchTool,
		carriertools.ProvideAcceptCarrierInvoiceMatchWithVarianceTool,
		carriertools.ProvideRejectCarrierInvoiceMatchTool,
		carriertools.ProvideLinkEDICarrierInvoiceTool,
	)
}
