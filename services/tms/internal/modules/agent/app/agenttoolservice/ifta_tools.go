package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
)

var _ receivabletools.IftaReturnKeeper = (*iftaservice.Service)(nil)

var _ receivabletools.IftaMileageKeeper = (*iftaservice.Service)(nil)

var _ receivabletools.JurisdictionMileRouter = shipmentports.DistanceCalculationService(nil)
