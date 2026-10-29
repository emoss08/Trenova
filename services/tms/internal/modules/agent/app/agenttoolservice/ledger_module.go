package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ledgertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
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
		ledgertools.ProvideCloseFiscalPeriodTool,
		ledgertools.ProvideLockFiscalPeriodTool,
		ledgertools.ProvideUnlockFiscalPeriodTool,
		ledgertools.ProvideReopenFiscalPeriodTool,
		ledgertools.ProvideOpenFiscalPeriodTool,
		provideConfirmAccountingMappingProposalsTool,
		provideRejectAccountingMappingProposalTool,
		provideReleaseAccountingSyncTool,
		provideChangeAccountingBackfillTool,
		ledgertools.ProvideTriageBankReceiptWorkItemTool,
	}
}

func provideConfirmAccountingMappingProposalsTool(
	mappings *accountingmappingservice.Service,
) agenttoolports.AgentTool {
	return accountingtools.NewConfirmAccountingMappingProposalsTool(mappings)
}

func provideRejectAccountingMappingProposalTool(
	mappings *accountingmappingservice.Service,
) agenttoolports.AgentTool {
	return accountingtools.NewRejectAccountingMappingProposalTool(mappings)
}

func provideReleaseAccountingSyncTool(
	sync *accountingsyncservice.Service,
) agenttoolports.AgentTool {
	return accountingtools.NewReleaseAccountingSyncTool(sync)
}

func provideChangeAccountingBackfillTool(
	sync *accountingsyncservice.Service,
) agenttoolports.AgentTool {
	return accountingtools.NewChangeAccountingBackfillTool(sync)
}
