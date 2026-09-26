package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carrierassignmentservice"
	"github.com/emoss08/trenova/internal/core/services/rateconfirmationservice"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
)

// The carrier assignment, tender and rate confirmation services are provided
// as concrete structs, so the dispatch and tendering tools are widened to
// the narrow interfaces they use here, as the rest of the package's are in
// module.go.

func provideAssignMoveToCarrierTool(
	carriers *carrierassignmentservice.Service,
) services.AgentTool {
	return newAssignMoveToCarrierTool(carriers)
}

func provideCancelCarrierAssignmentTool(
	carriers *carrierassignmentservice.Service,
) services.AgentTool {
	return newCancelCarrierAssignmentTool(carriers)
}

func provideCancelTenderTool(tenders *tenderservice.Service) services.AgentTool {
	return newCancelTenderTool(tenders)
}

func provideRecordTenderResponseTool(tenders *tenderservice.Service) services.AgentTool {
	return newRecordTenderResponseTool(tenders)
}

func provideGenerateRateConfirmationTool(
	rateCons *rateconfirmationservice.Service,
) services.AgentTool {
	return newGenerateRateConfirmationTool(rateCons)
}

func provideSendRateConfirmationTool(
	rateCons *rateconfirmationservice.Service,
) services.AgentTool {
	return newSendRateConfirmationTool(rateCons)
}

func provideVoidRateConfirmationTool(
	rateCons *rateconfirmationservice.Service,
) services.AgentTool {
	return newVoidRateConfirmationTool(rateCons)
}

func provideRecordRateConfirmationConfirmedTool(
	rateCons *rateconfirmationservice.Service,
) services.AgentTool {
	return newRecordRateConfirmationConfirmedTool(rateCons)
}
