package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ratetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/reporttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/app/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateimportservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratesimulationservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
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

	for _, step := range ratetools.AgreementReviewSteps() {
		providers = append(providers, provideAgreementReviewTool(step))
	}

	return providers
}

func provideRecordFuelPurchaseTool(
	purchases *fuelpurchaseservice.Service,
	jurisdictions *iftaservice.Service,
) agentports.AgentTool {
	return receivabletools.NewRecordFuelPurchaseTool(purchases, jurisdictions)
}

func provideCorrectFuelPurchaseTool(
	purchases *fuelpurchaseservice.Service,
	jurisdictions *iftaservice.Service,
) agentports.AgentTool {
	return receivabletools.NewCorrectFuelPurchaseTool(purchases, jurisdictions)
}

func provideDeleteFuelPurchaseTool(
	purchases *fuelpurchaseservice.Service,
	jurisdictions *iftaservice.Service,
) agentports.AgentTool {
	return receivabletools.NewDeleteFuelPurchaseTool(purchases, jurisdictions)
}

func provideAssignFuelCardTool(cards *fuelpurchaseservice.Service) agentports.AgentTool {
	return receivabletools.NewAssignFuelCardTool(cards)
}

func provideCommitFuelPurchaseImportTool(
	imports *fuelpurchaseservice.Service,
) agentports.AgentTool {
	return receivabletools.NewCommitFuelPurchaseImportTool(imports)
}

func provideResolveFuelPurchaseImportRowsTool(
	imports *fuelpurchaseservice.Service,
) agentports.AgentTool {
	return receivabletools.NewResolveFuelPurchaseImportRowsTool(imports)
}

func provideDiscardFuelPurchaseImportTool(
	imports *fuelpurchaseservice.Service,
) agentports.AgentTool {
	return receivabletools.NewDiscardFuelPurchaseImportTool(imports)
}

func provideRecordFuelIndexPriceTool(prices *fuelsurchargeservice.Service) agentports.AgentTool {
	return receivabletools.NewRecordFuelIndexPriceTool(prices)
}

func provideCorrectFuelIndexPriceTool(prices *fuelsurchargeservice.Service) agentports.AgentTool {
	return receivabletools.NewCorrectFuelIndexPriceTool(prices)
}

func provideGenerateIFTAReturnTool(returns *iftaservice.Service) agentports.AgentTool {
	return receivabletools.NewGenerateIFTAReturnTool(returns)
}

func provideRecomputeIFTAReturnTool(returns *iftaservice.Service) agentports.AgentTool {
	return receivabletools.NewRecomputeIFTAReturnTool(returns)
}

func provideAmendIFTAReturnTool(returns *iftaservice.Service) agentports.AgentTool {
	return receivabletools.NewAmendIFTAReturnTool(returns)
}

func provideDeleteIFTAReturnTool(returns *iftaservice.Service) agentports.AgentTool {
	return receivabletools.NewDeleteIFTAReturnTool(returns)
}

func provideRecordIFTAMileageEntryTool(entries *iftaservice.Service) agentports.AgentTool {
	return receivabletools.NewRecordIFTAMileageEntryTool(entries)
}

func provideCorrectIFTAMileageEntryTool(entries *iftaservice.Service) agentports.AgentTool {
	return receivabletools.NewCorrectIFTAMileageEntryTool(entries)
}

func provideDeleteIFTAMileageEntryTool(entries *iftaservice.Service) agentports.AgentTool {
	return receivabletools.NewDeleteIFTAMileageEntryTool(entries)
}

func provideRecalculateMoveJurisdictionMilesTool(
	router shipmentports.DistanceCalculationService,
) agentports.AgentTool {
	return receivabletools.NewRecalculateMoveJurisdictionMilesTool(router)
}

func provideBackfillJurisdictionMilesTool(
	returns *iftaservice.Service,
	router shipmentports.DistanceCalculationService,
) agentports.AgentTool {
	return receivabletools.NewBackfillJurisdictionMilesTool(returns, router)
}

func provideDraftRateAgreementTool(
	agreements *rateagreementservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return ratetools.NewDraftRateAgreementTool(agreements, states)
}

func provideReviseRateAgreementDraftTool(
	agreements *rateagreementservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return ratetools.NewReviseRateAgreementDraftTool(agreements, states)
}

func provideDuplicateRateAgreementTool(
	agreements *rateagreementservice.Service,
) agentports.AgentTool {
	return ratetools.NewDuplicateRateAgreementTool(agreements)
}

func provideAmendRateAgreementRulesTool(
	agreements *rateagreementservice.Service,
	states usstateports.UsStateRepository,
) agentports.AgentTool {
	return ratetools.NewAmendRateAgreementRulesTool(agreements, states)
}

func provideApplyRateIncreaseTool(agreements *rateagreementservice.Service) agentports.AgentTool {
	return ratetools.NewApplyRateIncreaseTool(agreements)
}

func provideAgreementReviewTool(
	step *ratetools.AgreementReviewStep,
) func(*rateagreementservice.Service) agentports.AgentTool {
	return func(agreements *rateagreementservice.Service) agentports.AgentTool {
		return ratetools.NewAgreementReviewTool(agreements, step)
	}
}

func provideCommitRateImportTool(imports *rateimportservice.Service) agentports.AgentTool {
	return ratetools.NewCommitRateImportTool(imports)
}

func provideDiscardRateImportTool(imports *rateimportservice.Service) agentports.AgentTool {
	return ratetools.NewDiscardRateImportTool(imports)
}

func provideRunRateSimulationTool(
	simulations *ratesimulationservice.Service,
) agentports.AgentTool {
	return ratetools.NewRunRateSimulationTool(simulations)
}

func provideCancelReportRunTool(reports *reporting.Service) agentports.AgentTool {
	return reporttools.NewCancelReportRunTool(reports)
}

func provideDeleteReportTool(reports *reporting.Service) agentports.AgentTool {
	return reporttools.NewDeleteReportTool(reports)
}

func provideResetReportForkTool(reports *reporting.Service) agentports.AgentTool {
	return reporttools.NewResetReportForkTool(reports)
}

func provideDeleteDashboardTool(reports *reporting.Service) agentports.AgentTool {
	return reporttools.NewDeleteDashboardTool(reports)
}

func provideUpdateReportScheduleTool(reports *reporting.Service) agentports.AgentTool {
	return reporttools.NewUpdateReportScheduleTool(reports)
}

func provideDeleteReportScheduleTool(reports *reporting.Service) agentports.AgentTool {
	return reporttools.NewDeleteReportScheduleTool(reports)
}
