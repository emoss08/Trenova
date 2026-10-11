package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/carrierassignmentservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateconfirmationservice"
)

// The carrier assignment, tender and rate confirmation services are provided
// as concrete structs, so the dispatch and tendering tools are widened to
// the narrow interfaces they use here, as the rest of the package's are in
// module.go.

func provideAssignMoveToCarrierTool(
	carriers *carrierassignmentservice.Service,
) agentports.AgentTool {
	return newAssignMoveToCarrierTool(carriers)
}

func provideCancelCarrierAssignmentTool(
	carriers *carrierassignmentservice.Service,
) agentports.AgentTool {
	return newCancelCarrierAssignmentTool(carriers)
}

func provideCancelTenderTool(tenders *tenderservice.Service) agentports.AgentTool {
	return newCancelTenderTool(tenders)
}

func provideRecordTenderResponseTool(tenders *tenderservice.Service) agentports.AgentTool {
	return newRecordTenderResponseTool(tenders)
}

func provideGenerateRateConfirmationTool(
	rateCons *rateconfirmationservice.Service,
) agentports.AgentTool {
	return newGenerateRateConfirmationTool(rateCons)
}

func provideSendRateConfirmationTool(
	rateCons *rateconfirmationservice.Service,
) agentports.AgentTool {
	return newSendRateConfirmationTool(rateCons)
}

func provideVoidRateConfirmationTool(
	rateCons *rateconfirmationservice.Service,
) agentports.AgentTool {
	return newVoidRateConfirmationTool(rateCons)
}

func provideRecordRateConfirmationConfirmedTool(
	rateCons *rateconfirmationservice.Service,
) agentports.AgentTool {
	return newRecordRateConfirmationConfirmedTool(rateCons)
}
