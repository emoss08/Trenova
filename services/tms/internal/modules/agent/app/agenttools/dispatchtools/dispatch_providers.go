package dispatchtools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/rateconfirmationtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/tendertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
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

func ProvideCancelTenderTool(tenders *tenderservice.Service) agenttoolports.AgentTool {
	return tendertools.NewCancelTenderTool(tenders)
}

func ProvideRecordTenderResponseTool(tenders *tenderservice.Service) agenttoolports.AgentTool {
	return tendertools.NewRecordTenderResponseTool(tenders)
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
