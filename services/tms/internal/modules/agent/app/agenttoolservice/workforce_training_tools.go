package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/trainingtools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workertrainingservice"
)

var _ trainingtools.TrainingKeeper = (*workertrainingservice.Service)(nil)
