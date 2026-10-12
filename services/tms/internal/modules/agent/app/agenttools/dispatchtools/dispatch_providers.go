package dispatchtools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ratetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/tendertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/carrierassignmentservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateconfirmationservice"
)

func ProvideAssignMoveToCarrierTool(
	carriers *carrierassignmentservice.Service,
) agentports.AgentTool {
	return carriertools.NewAssignMoveToCarrierTool(carriers)
}

func ProvideCancelCarrierAssignmentTool(
	carriers *carrierassignmentservice.Service,
) agentports.AgentTool {
	return carriertools.NewCancelCarrierAssignmentTool(carriers)
}

func ProvideCancelTenderTool(tenders *tenderservice.Service) agentports.AgentTool {
	return tendertools.NewCancelTenderTool(tenders)
}

func ProvideRecordTenderResponseTool(tenders *tenderservice.Service) agentports.AgentTool {
	return tendertools.NewRecordTenderResponseTool(tenders)
}

func ProvideGenerateRateConfirmationTool(
	rateCons *rateconfirmationservice.Service,
) agentports.AgentTool {
	return ratetools.NewGenerateRateConfirmationTool(rateCons)
}

func ProvideSendRateConfirmationTool(
	rateCons *rateconfirmationservice.Service,
) agentports.AgentTool {
	return ratetools.NewSendRateConfirmationTool(rateCons)
}

func ProvideVoidRateConfirmationTool(
	rateCons *rateconfirmationservice.Service,
) agentports.AgentTool {
	return ratetools.NewVoidRateConfirmationTool(rateCons)
}

func ProvideRecordRateConfirmationConfirmedTool(
	rateCons *rateconfirmationservice.Service,
) agentports.AgentTool {
	return ratetools.NewRecordRateConfirmationConfirmedTool(rateCons)
}
