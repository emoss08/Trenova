package dispatchtools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/rateconfirmationtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/tendertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderbounce"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercancel"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderresponse"
	"github.com/emoss08/trenova/internal/modules/rating/app/carrierassignmentservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateconfirmationservice"
)

func ProvideAssignMoveToCarrierTool(
	carriers *carrierassignmentservice.Service,
) agenttoolports.AgentTool {
	return carriertools.NewAssignMoveToCarrierTool(carriers)
}

func ProvideCancelCarrierAssignmentTool(
	carriers *carrierassignmentservice.Service,
) agenttoolports.AgentTool {
	return carriertools.NewCancelCarrierAssignmentTool(carriers)
}

func ProvideCancelTenderTool(tenders *tendercancel.Service) agenttoolports.AgentTool {
	return tendertools.NewCancelTenderTool(tenders)
}

func ProvideRecordTenderResponseTool(
	responses *tenderresponse.Service,
	bounces *tenderbounce.Service,
) agenttoolports.AgentTool {
	return tendertools.NewRecordTenderResponseTool(responses, bounces)
}

func ProvideGenerateRateConfirmationTool(
	rateCons *rateconfirmationservice.Service,
) agenttoolports.AgentTool {
	return rateconfirmationtools.NewGenerateRateConfirmationTool(rateCons)
}

func ProvideSendRateConfirmationTool(
	rateCons *rateconfirmationservice.Service,
) agenttoolports.AgentTool {
	return rateconfirmationtools.NewSendRateConfirmationTool(rateCons)
}

func ProvideVoidRateConfirmationTool(
	rateCons *rateconfirmationservice.Service,
) agenttoolports.AgentTool {
	return rateconfirmationtools.NewVoidRateConfirmationTool(rateCons)
}

func ProvideRecordRateConfirmationConfirmedTool(
	rateCons *rateconfirmationservice.Service,
) agenttoolports.AgentTool {
	return rateconfirmationtools.NewRecordRateConfirmationConfirmedTool(rateCons)
}
