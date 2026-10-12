package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workertrainingservice"
)

var _ workforcetools.TrainingKeeper = (*workertrainingservice.Service)(nil)
