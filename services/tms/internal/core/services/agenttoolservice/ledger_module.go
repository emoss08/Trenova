package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/accountingmappingservice"
	"github.com/emoss08/trenova/internal/core/services/accountingsyncservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

func ledgerToolProviders() []any {
	return []any{
		provideDraftManualJournalTool,
		provideReviseManualJournalDraftTool,
		provideSubmitManualJournalTool,
		provideCancelManualJournalTool,
		providePostManualJournalTool,
		provideRequestJournalReversalTool,
		provideCancelJournalReversalTool,
		providePostJournalReversalTool,
		provideCloseFiscalPeriodTool,
		provideLockFiscalPeriodTool,
		provideUnlockFiscalPeriodTool,
		provideReopenFiscalPeriodTool,
		provideOpenFiscalPeriodTool,
		provideConfirmAccountingMappingProposalsTool,
		provideRejectAccountingMappingProposalTool,
		provideReleaseAccountingSyncTool,
		provideChangeAccountingBackfillTool,
		provideTriageBankReceiptWorkItemTool,
	}
}

func provideConfirmAccountingMappingProposalsTool(
	mappings *accountingmappingservice.Service,
) agentports.AgentTool {
	return newConfirmAccountingMappingProposalsTool(mappings)
}

func provideRejectAccountingMappingProposalTool(
	mappings *accountingmappingservice.Service,
) agentports.AgentTool {
	return newRejectAccountingMappingProposalTool(mappings)
}

func provideReleaseAccountingSyncTool(
	sync *accountingsyncservice.Service,
) agentports.AgentTool {
	return newReleaseAccountingSyncTool(sync)
}

func provideChangeAccountingBackfillTool(
	sync *accountingsyncservice.Service,
) agentports.AgentTool {
	return newChangeAccountingBackfillTool(sync)
}
