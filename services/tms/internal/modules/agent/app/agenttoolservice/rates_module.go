package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/fueltools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/iftatools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ratetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/reporttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/rating/app/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateimportservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratesimulationservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportdashboardservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportscheduleservice"
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
) agenttoolports.AgentTool {
	return fueltools.NewRecordFuelPurchaseTool(purchases, jurisdictions)
}

func provideCorrectFuelPurchaseTool(
	purchases *fuelpurchaseservice.Service,
	jurisdictions *iftaservice.Service,
) agenttoolports.AgentTool {
	return fueltools.NewCorrectFuelPurchaseTool(purchases, jurisdictions)
}

func provideDeleteFuelPurchaseTool(
	purchases *fuelpurchaseservice.Service,
	jurisdictions *iftaservice.Service,
) agenttoolports.AgentTool {
	return fueltools.NewDeleteFuelPurchaseTool(purchases, jurisdictions)
}

func provideAssignFuelCardTool(cards *fuelpurchaseservice.Service) agenttoolports.AgentTool {
	return fueltools.NewAssignFuelCardTool(cards)
}

func provideCommitFuelPurchaseImportTool(
	imports *fuelpurchaseservice.Service,
) agenttoolports.AgentTool {
	return fueltools.NewCommitFuelPurchaseImportTool(imports)
}

func provideResolveFuelPurchaseImportRowsTool(
	imports *fuelpurchaseservice.Service,
) agenttoolports.AgentTool {
	return fueltools.NewResolveFuelPurchaseImportRowsTool(imports)
}

func provideDiscardFuelPurchaseImportTool(
	imports *fuelpurchaseservice.Service,
) agenttoolports.AgentTool {
	return fueltools.NewDiscardFuelPurchaseImportTool(imports)
}

func provideRecordFuelIndexPriceTool(
	prices *fuelsurchargeservice.Service,
) agenttoolports.AgentTool {
	return fueltools.NewRecordFuelIndexPriceTool(prices)
}

func provideCorrectFuelIndexPriceTool(
	prices *fuelsurchargeservice.Service,
) agenttoolports.AgentTool {
	return fueltools.NewCorrectFuelIndexPriceTool(prices)
}

func provideGenerateIFTAReturnTool(returns *iftaservice.Service) agenttoolports.AgentTool {
	return iftatools.NewGenerateIFTAReturnTool(returns)
}

func provideRecomputeIFTAReturnTool(returns *iftaservice.Service) agenttoolports.AgentTool {
	return iftatools.NewRecomputeIFTAReturnTool(returns)
}

func provideAmendIFTAReturnTool(returns *iftaservice.Service) agenttoolports.AgentTool {
	return iftatools.NewAmendIFTAReturnTool(returns)
}

func provideDeleteIFTAReturnTool(returns *iftaservice.Service) agenttoolports.AgentTool {
	return iftatools.NewDeleteIFTAReturnTool(returns)
}

func provideRecordIFTAMileageEntryTool(entries *iftaservice.Service) agenttoolports.AgentTool {
	return iftatools.NewRecordIFTAMileageEntryTool(entries)
}

func provideCorrectIFTAMileageEntryTool(entries *iftaservice.Service) agenttoolports.AgentTool {
	return iftatools.NewCorrectIFTAMileageEntryTool(entries)
}

func provideDeleteIFTAMileageEntryTool(entries *iftaservice.Service) agenttoolports.AgentTool {
	return iftatools.NewDeleteIFTAMileageEntryTool(entries)
}

func provideRecalculateMoveJurisdictionMilesTool(
	router shipmentdistanceports.DistanceCalculationService,
) agenttoolports.AgentTool {
	return iftatools.NewRecalculateMoveJurisdictionMilesTool(router)
}

func provideBackfillJurisdictionMilesTool(
	returns *iftaservice.Service,
	router shipmentdistanceports.DistanceCalculationService,
) agenttoolports.AgentTool {
	return iftatools.NewBackfillJurisdictionMilesTool(returns, router)
}

func provideDraftRateAgreementTool(
	agreements *rateagreementservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return ratetools.NewDraftRateAgreementTool(agreements, states)
}

func provideReviseRateAgreementDraftTool(
	agreements *rateagreementservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return ratetools.NewReviseRateAgreementDraftTool(agreements, states)
}

func provideDuplicateRateAgreementTool(
	agreements *rateagreementservice.Service,
) agenttoolports.AgentTool {
	return ratetools.NewDuplicateRateAgreementTool(agreements)
}

func provideAmendRateAgreementRulesTool(
	agreements *rateagreementservice.Service,
	states usstateports.UsStateRepository,
) agenttoolports.AgentTool {
	return ratetools.NewAmendRateAgreementRulesTool(agreements, states)
}

func provideApplyRateIncreaseTool(
	agreements *rateagreementservice.Service,
) agenttoolports.AgentTool {
	return ratetools.NewApplyRateIncreaseTool(agreements)
}

func provideAgreementReviewTool(
	step *ratetools.AgreementReviewStep,
) func(*rateagreementservice.Service) agenttoolports.AgentTool {
	return func(agreements *rateagreementservice.Service) agenttoolports.AgentTool {
		return ratetools.NewAgreementReviewTool(agreements, step)
	}
}

func provideCommitRateImportTool(imports *rateimportservice.Service) agenttoolports.AgentTool {
	return ratetools.NewCommitRateImportTool(imports)
}

func provideDiscardRateImportTool(imports *rateimportservice.Service) agenttoolports.AgentTool {
	return ratetools.NewDiscardRateImportTool(imports)
}

func provideRunRateSimulationTool(
	simulations *ratesimulationservice.Service,
) agenttoolports.AgentTool {
	return ratetools.NewRunRateSimulationTool(simulations)
}

func provideCancelReportRunTool(reports *reporting.Service) agenttoolports.AgentTool {
	return reporttools.NewCancelReportRunTool(reports)
}

func provideDeleteReportTool(reports *reporting.Service) agenttoolports.AgentTool {
	return reporttools.NewDeleteReportTool(reports)
}

func provideResetReportForkTool(reports *reporting.Service) agenttoolports.AgentTool {
	return reporttools.NewResetReportForkTool(reports)
}

func provideDeleteDashboardTool(
	dashboards *reportdashboardservice.Service,
) agenttoolports.AgentTool {
	return reporttools.NewDeleteDashboardTool(dashboards)
}

func provideUpdateReportScheduleTool(
	schedules *reportscheduleservice.Service,
) agenttoolports.AgentTool {
	return reporttools.NewUpdateReportScheduleTool(schedules)
}

func provideDeleteReportScheduleTool(
	schedules *reportscheduleservice.Service,
) agenttoolports.AgentTool {
	return reporttools.NewDeleteReportScheduleTool(schedules)
}
