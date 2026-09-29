package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/core/services/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/core/services/iftaservice"
	"github.com/emoss08/trenova/internal/core/services/rateagreementservice"
	"github.com/emoss08/trenova/internal/core/services/rateimportservice"
	"github.com/emoss08/trenova/internal/core/services/ratesimulationservice"
	"github.com/emoss08/trenova/internal/core/services/reporting"
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
) serviceports.AgentTool {
	return newRecordFuelPurchaseTool(purchases, jurisdictions)
}

func provideCorrectFuelPurchaseTool(
	purchases *fuelpurchaseservice.Service,
	jurisdictions *iftaservice.Service,
) serviceports.AgentTool {
	return newCorrectFuelPurchaseTool(purchases, jurisdictions)
}

func provideDeleteFuelPurchaseTool(
	purchases *fuelpurchaseservice.Service,
	jurisdictions *iftaservice.Service,
) serviceports.AgentTool {
	return newDeleteFuelPurchaseTool(purchases, jurisdictions)
}

func provideAssignFuelCardTool(cards *fuelpurchaseservice.Service) serviceports.AgentTool {
	return newAssignFuelCardTool(cards)
}

func provideCommitFuelPurchaseImportTool(
	imports *fuelpurchaseservice.Service,
) serviceports.AgentTool {
	return newCommitFuelPurchaseImportTool(imports)
}

func provideResolveFuelPurchaseImportRowsTool(
	imports *fuelpurchaseservice.Service,
) serviceports.AgentTool {
	return newResolveFuelPurchaseImportRowsTool(imports)
}

func provideDiscardFuelPurchaseImportTool(
	imports *fuelpurchaseservice.Service,
) serviceports.AgentTool {
	return newDiscardFuelPurchaseImportTool(imports)
}

func provideRecordFuelIndexPriceTool(prices *fuelsurchargeservice.Service) serviceports.AgentTool {
	return newRecordFuelIndexPriceTool(prices)
}

func provideCorrectFuelIndexPriceTool(prices *fuelsurchargeservice.Service) serviceports.AgentTool {
	return newCorrectFuelIndexPriceTool(prices)
}

func provideGenerateIFTAReturnTool(returns *iftaservice.Service) serviceports.AgentTool {
	return newGenerateIFTAReturnTool(returns)
}

func provideRecomputeIFTAReturnTool(returns *iftaservice.Service) serviceports.AgentTool {
	return newRecomputeIFTAReturnTool(returns)
}

func provideAmendIFTAReturnTool(returns *iftaservice.Service) serviceports.AgentTool {
	return newAmendIFTAReturnTool(returns)
}

func provideDeleteIFTAReturnTool(returns *iftaservice.Service) serviceports.AgentTool {
	return newDeleteIFTAReturnTool(returns)
}

func provideRecordIFTAMileageEntryTool(entries *iftaservice.Service) serviceports.AgentTool {
	return newRecordIFTAMileageEntryTool(entries)
}

func provideCorrectIFTAMileageEntryTool(entries *iftaservice.Service) serviceports.AgentTool {
	return newCorrectIFTAMileageEntryTool(entries)
}

func provideDeleteIFTAMileageEntryTool(entries *iftaservice.Service) serviceports.AgentTool {
	return newDeleteIFTAMileageEntryTool(entries)
}

func provideRecalculateMoveJurisdictionMilesTool(
	router serviceports.DistanceCalculationService,
) serviceports.AgentTool {
	return newRecalculateMoveJurisdictionMilesTool(router)
}

func provideBackfillJurisdictionMilesTool(
	returns *iftaservice.Service,
	router serviceports.DistanceCalculationService,
) serviceports.AgentTool {
	return newBackfillJurisdictionMilesTool(returns, router)
}

func provideDraftRateAgreementTool(
	agreements *rateagreementservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newDraftRateAgreementTool(agreements, states)
}

func provideReviseRateAgreementDraftTool(
	agreements *rateagreementservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newReviseRateAgreementDraftTool(agreements, states)
}

func provideDuplicateRateAgreementTool(
	agreements *rateagreementservice.Service,
) serviceports.AgentTool {
	return newDuplicateRateAgreementTool(agreements)
}

func provideAmendRateAgreementRulesTool(
	agreements *rateagreementservice.Service,
	states repositories.UsStateRepository,
) serviceports.AgentTool {
	return newAmendRateAgreementRulesTool(agreements, states)
}

func provideApplyRateIncreaseTool(agreements *rateagreementservice.Service) serviceports.AgentTool {
	return newApplyRateIncreaseTool(agreements)
}

func provideAgreementReviewTool(
	step *agreementReviewStep,
) func(*rateagreementservice.Service) serviceports.AgentTool {
	return func(agreements *rateagreementservice.Service) serviceports.AgentTool {
		return newAgreementReviewTool(agreements, step)
	}
}

func provideCommitRateImportTool(imports *rateimportservice.Service) serviceports.AgentTool {
	return newCommitRateImportTool(imports)
}

func provideDiscardRateImportTool(imports *rateimportservice.Service) serviceports.AgentTool {
	return newDiscardRateImportTool(imports)
}

func provideRunRateSimulationTool(
	simulations *ratesimulationservice.Service,
) serviceports.AgentTool {
	return newRunRateSimulationTool(simulations)
}

func provideCancelReportRunTool(reports *reporting.Service) serviceports.AgentTool {
	return newCancelReportRunTool(reports)
}

func provideDeleteReportTool(reports *reporting.Service) serviceports.AgentTool {
	return newDeleteReportTool(reports)
}

func provideResetReportForkTool(reports *reporting.Service) serviceports.AgentTool {
	return newResetReportForkTool(reports)
}

func provideDeleteDashboardTool(reports *reporting.Service) serviceports.AgentTool {
	return newDeleteDashboardTool(reports)
}

func provideUpdateReportScheduleTool(reports *reporting.Service) serviceports.AgentTool {
	return newUpdateReportScheduleTool(reports)
}

func provideDeleteReportScheduleTool(reports *reporting.Service) serviceports.AgentTool {
	return newDeleteReportScheduleTool(reports)
}
