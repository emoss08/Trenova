package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytools"
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
		settlementtools.ProvideGenerateDriverSettlementTool,
		settlementtools.ProvideGenerateDriverSettlementBatchTool,
		settlementtools.ProvideGenerateCarrierSettlementBatchTool,
		paytools.ProvidePayWorkerNowTool,
		paytools.ProvideIssuePayAdvanceTool,
		paytools.ProvideWriteOffPayAdvanceTool,
		settlementtools.ProvideOpenEscrowAccountTool,
		settlementtools.ProvideUpdateEscrowAccountTool,
		settlementtools.ProvideAdjustEscrowAccountTool,
		settlementtools.ProvideCloseEscrowAccountTool,
		paytools.ProvideAssignPayProfileTool,
		paytools.ProvideEndPayAssignmentTool,
		settlementtools.ProvideCreateRecurringDeductionTool,
		settlementtools.ProvideUpdateRecurringDeductionTool,
		settlementtools.ProvideCreateRecurringEarningTool,
		settlementtools.ProvideUpdateRecurringEarningTool,
		carriertools.ProvideCreateCarrierInvoiceMatchTool,
		carriertools.ProvideAcceptCarrierInvoiceMatchTool,
		carriertools.ProvideAcceptCarrierInvoiceMatchWithVarianceTool,
		carriertools.ProvideRejectCarrierInvoiceMatchTool,
		carriertools.ProvideLinkEDICarrierInvoiceTool,
	)
}
