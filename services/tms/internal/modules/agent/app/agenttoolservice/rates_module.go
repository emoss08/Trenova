package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/settlement/app/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateimportservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratesimulationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
)

func ratesToolProviders() []any {
	providers := []any{
		provideRecordFuelPurchaseTool,
		provideCorrectFuelPurchaseTool,
		provideDeleteFuelPurchaseTool,
		provideAssignFuelCardTool,
		provideCommitFuelPurchaseImportTool,
		provideResolveFuelPurchaseImportRowsTool,
		provideDiscardFuelPurchaseImportTool,
		provideRecordFuelIndexPriceTool,
		provideCorrectFuelIndexPriceTool,
		provideGenerateIFTAReturnTool,
		provideRecomputeIFTAReturnTool,
		provideAmendIFTAReturnTool,
		provideDeleteIFTAReturnTool,
		provideRecordIFTAMileageEntryTool,
		provideCorrectIFTAMileageEntryTool,
		provideDeleteIFTAMileageEntryTool,
		provideRecalculateMoveJurisdictionMilesTool,
		provideBackfillJurisdictionMilesTool,
		provideDraftRateAgreementTool,
		provideReviseRateAgreementDraftTool,
		provideDuplicateRateAgreementTool,
		provideAmendRateAgreementRulesTool,
		provideApplyRateIncreaseTool,
		provideCommitRateImportTool,
		provideDiscardRateImportTool,
		provideRunRateSimulationTool,
		provideCancelReportRunTool,
		provideDeleteReportTool,
		provideResetReportForkTool,
		provideDeleteDashboardTool,
		provideUpdateReportScheduleTool,
		provideDeleteReportScheduleTool,
	}

	for _, step := range agreementReviewSteps() {
		providers = append(providers, provideAgreementReviewTool(step))
	}

	return providers
}

func provideRecordFuelPurchaseTool(
	purchases *fuelpurchaseservice.Service,
	jurisdictions *iftaservice.Service,
) agentports.AgentTool {
	return newRecordFuelPurchaseTool(purchases, jurisdictions)
}

func provideCorrectFuelPurchaseTool(
	purchases *fuelpurchaseservice.Service,
	jurisdictions *iftaservice.Service,
) agentports.AgentTool {
	return newCorrectFuelPurchaseTool(purchases, jurisdictions)
}

func provideDeleteFuelPurchaseTool(
	purchases *fuelpurchaseservice.Service,
	jurisdictions *iftaservice.Service,
) agentports.AgentTool {
	return newDeleteFuelPurchaseTool(purchases, jurisdictions)
}

func provideAssignFuelCardTool(cards *fuelpurchaseservice.Service) agentports.AgentTool {
	return newAssignFuelCardTool(cards)
}

func provideCommitFuelPurchaseImportTool(
	imports *fuelpurchaseservice.Service,
) agentports.AgentTool {
	return newCommitFuelPurchaseImportTool(imports)
}

func provideResolveFuelPurchaseImportRowsTool(
	imports *fuelpurchaseservice.Service,
) agentports.AgentTool {
	return newResolveFuelPurchaseImportRowsTool(imports)
}

func provideDiscardFuelPurchaseImportTool(
	imports *fuelpurchaseservice.Service,
) agentports.AgentTool {
	return newDiscardFuelPurchaseImportTool(imports)
}

func provideRecordFuelIndexPriceTool(prices *fuelsurchargeservice.Service) agentports.AgentTool {
	return newRecordFuelIndexPriceTool(prices)
}

func provideCorrectFuelIndexPriceTool(prices *fuelsurchargeservice.Service) agentports.AgentTool {
	return newCorrectFuelIndexPriceTool(prices)
}

func provideGenerateIFTAReturnTool(returns *iftaservice.Service) agentports.AgentTool {
	return newGenerateIFTAReturnTool(returns)
}

func provideRecomputeIFTAReturnTool(returns *iftaservice.Service) agentports.AgentTool {
	return newRecomputeIFTAReturnTool(returns)
}

func provideAmendIFTAReturnTool(returns *iftaservice.Service) agentports.AgentTool {
	return newAmendIFTAReturnTool(returns)
}

func provideDeleteIFTAReturnTool(returns *iftaservice.Service) agentports.AgentTool {
	return newDeleteIFTAReturnTool(returns)
}

func provideRecordIFTAMileageEntryTool(entries *iftaservice.Service) agentports.AgentTool {
	return newRecordIFTAMileageEntryTool(entries)
}

func provideCorrectIFTAMileageEntryTool(entries *iftaservice.Service) agentports.AgentTool {
	return newCorrectIFTAMileageEntryTool(entries)
}

func provideDeleteIFTAMileageEntryTool(entries *iftaservice.Service) agentports.AgentTool {
	return newDeleteIFTAMileageEntryTool(entries)
}

func provideRecalculateMoveJurisdictionMilesTool(
	router shipmentports.DistanceCalculationService,
) agentports.AgentTool {
	return newRecalculateMoveJurisdictionMilesTool(router)
}

func provideBackfillJurisdictionMilesTool(
	returns *iftaservice.Service,
	router shipmentports.DistanceCalculationService,
) agentports.AgentTool {
	return newBackfillJurisdictionMilesTool(returns, router)
}

func provideDraftRateAgreementTool(
	agreements *rateagreementservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newDraftRateAgreementTool(agreements, states)
}

func provideReviseRateAgreementDraftTool(
	agreements *rateagreementservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newReviseRateAgreementDraftTool(agreements, states)
}

func provideDuplicateRateAgreementTool(
	agreements *rateagreementservice.Service,
) agentports.AgentTool {
	return newDuplicateRateAgreementTool(agreements)
}

func provideAmendRateAgreementRulesTool(
	agreements *rateagreementservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return newAmendRateAgreementRulesTool(agreements, states)
}

func provideApplyRateIncreaseTool(agreements *rateagreementservice.Service) agentports.AgentTool {
	return newApplyRateIncreaseTool(agreements)
}

func provideAgreementReviewTool(
	step *agreementReviewStep,
) func(*rateagreementservice.Service) agentports.AgentTool {
	return func(agreements *rateagreementservice.Service) agentports.AgentTool {
		return newAgreementReviewTool(agreements, step)
	}
}

func provideCommitRateImportTool(imports *rateimportservice.Service) agentports.AgentTool {
	return newCommitRateImportTool(imports)
}

func provideDiscardRateImportTool(imports *rateimportservice.Service) agentports.AgentTool {
	return newDiscardRateImportTool(imports)
}

func provideRunRateSimulationTool(
	simulations *ratesimulationservice.Service,
) agentports.AgentTool {
	return newRunRateSimulationTool(simulations)
}

func provideCancelReportRunTool(reports *reporting.Service) agentports.AgentTool {
	return newCancelReportRunTool(reports)
}

func provideDeleteReportTool(reports *reporting.Service) agentports.AgentTool {
	return newDeleteReportTool(reports)
}

func provideResetReportForkTool(reports *reporting.Service) agentports.AgentTool {
	return newResetReportForkTool(reports)
}

func provideDeleteDashboardTool(reports *reporting.Service) agentports.AgentTool {
	return newDeleteDashboardTool(reports)
}

func provideUpdateReportScheduleTool(reports *reporting.Service) agentports.AgentTool {
	return newUpdateReportScheduleTool(reports)
}

func provideDeleteReportScheduleTool(reports *reporting.Service) agentports.AgentTool {
	return newDeleteReportScheduleTool(reports)
}
