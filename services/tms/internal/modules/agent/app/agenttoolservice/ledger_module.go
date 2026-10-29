package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingreviewtools"
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
	mappings accountingmappingports.AccountingMappingDecisionService,
) agenttoolports.AgentTool {
	return accountingreviewtools.NewConfirmAccountingMappingProposalsTool(mappings)
}

func provideRejectAccountingMappingProposalTool(
	mappings accountingmappingports.AccountingMappingDecisionService,
) agenttoolports.AgentTool {
	return accountingreviewtools.NewRejectAccountingMappingProposalTool(mappings)
}

func provideReleaseAccountingSyncTool(
	reads accountingsyncports.AccountingSyncReader,
	actions accountingsyncports.AccountingSyncActionService,
) agenttoolports.AgentTool {
	return accountingreviewtools.NewReleaseAccountingSyncTool(reads, actions)
}

func provideChangeAccountingBackfillTool(
	reads accountingsyncports.AccountingSyncReader,
	backfills accountingsyncports.AccountingBackfillService,
) agenttoolports.AgentTool {
	return accountingreviewtools.NewChangeAccountingBackfillTool(reads, backfills)
}
