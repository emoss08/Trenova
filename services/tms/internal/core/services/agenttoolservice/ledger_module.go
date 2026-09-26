package agenttoolservice

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingmappingservice"
	"github.com/emoss08/trenova/internal/core/services/accountingsyncservice"
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
) serviceports.AgentTool {
	return newConfirmAccountingMappingProposalsTool(mappings)
}

func provideRejectAccountingMappingProposalTool(
	mappings *accountingmappingservice.Service,
) serviceports.AgentTool {
	return newRejectAccountingMappingProposalTool(mappings)
}

func provideReleaseAccountingSyncTool(
	sync *accountingsyncservice.Service,
) serviceports.AgentTool {
	return newReleaseAccountingSyncTool(sync)
}

func provideChangeAccountingBackfillTool(
	sync *accountingsyncservice.Service,
) serviceports.AgentTool {
	return newChangeAccountingBackfillTool(sync)
}
