package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/workertrainingservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
)

var _ workforcetools.TrainingKeeper = (*workertrainingservice.Service)(nil)
