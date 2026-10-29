package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/iftatools"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
)

var _ iftatools.IftaReturnKeeper = (*iftaservice.Service)(nil)

var _ iftatools.IftaMileageKeeper = (*iftaservice.Service)(nil)

var _ iftatools.JurisdictionMileRouter = shipmentdistanceports.DistanceCalculationService(nil)
