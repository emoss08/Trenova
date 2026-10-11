package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
)

var _ workforcetools.DriverPayReviewer = (*driversettlementservice.Service)(nil)
