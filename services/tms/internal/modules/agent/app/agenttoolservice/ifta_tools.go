package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
)

var _ receivabletools.IftaReturnKeeper = (*iftaservice.Service)(nil)

var _ receivabletools.IftaMileageKeeper = (*iftaservice.Service)(nil)

var _ receivabletools.JurisdictionMileRouter = shipmentdistanceports.DistanceCalculationService(nil)
