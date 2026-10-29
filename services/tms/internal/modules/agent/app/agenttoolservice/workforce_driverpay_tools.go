package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/payrolltools"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementreviewservice"
)

var _ payrolltools.DriverPayReviewer = (*driversettlementreviewservice.Service)(nil)
