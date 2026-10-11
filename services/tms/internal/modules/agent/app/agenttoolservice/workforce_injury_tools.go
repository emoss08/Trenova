package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/workerinjuryservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
)

var _ workforcetools.InjuryKeeper = (*workerinjuryservice.Service)(nil)
