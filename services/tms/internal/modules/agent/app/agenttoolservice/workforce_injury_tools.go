package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerinjuryservice"
)

var _ workforcetools.InjuryKeeper = (*workerinjuryservice.Service)(nil)
