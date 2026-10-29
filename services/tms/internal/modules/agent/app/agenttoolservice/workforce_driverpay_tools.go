package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/payrolltools"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
)

var _ payrolltools.DriverPayReviewer = (*driversettlementservice.Service)(nil)
