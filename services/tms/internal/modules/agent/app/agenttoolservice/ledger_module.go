package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/accountingmappingservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ledgertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

func ledgerToolProviders() []any {
	return []any{
		ledgertools.ProvideDraftManualJournalTool,
		ledgertools.ProvideReviseManualJournalDraftTool,
		ledgertools.ProvideSubmitManualJournalTool,
		ledgertools.ProvideCancelManualJournalTool,
		ledgertools.ProvidePostManualJournalTool,
		ledgertools.ProvideRequestJournalReversalTool,
		ledgertools.ProvideCancelJournalReversalTool,
		ledgertools.ProvidePostJournalReversalTool,
		receivabletools.ProvideCloseFiscalPeriodTool,
		receivabletools.ProvideLockFiscalPeriodTool,
		receivabletools.ProvideUnlockFiscalPeriodTool,
		receivabletools.ProvideReopenFiscalPeriodTool,
		receivabletools.ProvideOpenFiscalPeriodTool,
		provideConfirmAccountingMappingProposalsTool,
		provideRejectAccountingMappingProposalTool,
		provideReleaseAccountingSyncTool,
		provideChangeAccountingBackfillTool,
		receivabletools.ProvideTriageBankReceiptWorkItemTool,
	}
}

func provideConfirmAccountingMappingProposalsTool(
	mappings *accountingmappingservice.Service,
) agentports.AgentTool {
	return accountingtools.NewConfirmAccountingMappingProposalsTool(mappings)
}

func provideRejectAccountingMappingProposalTool(
	mappings *accountingmappingservice.Service,
) agentports.AgentTool {
	return accountingtools.NewRejectAccountingMappingProposalTool(mappings)
}

func provideReleaseAccountingSyncTool(
	sync *accountingsyncservice.Service,
) agentports.AgentTool {
	return accountingtools.NewReleaseAccountingSyncTool(sync)
}

func provideChangeAccountingBackfillTool(
	sync *accountingsyncservice.Service,
) agentports.AgentTool {
	return accountingtools.NewChangeAccountingBackfillTool(sync)
}
