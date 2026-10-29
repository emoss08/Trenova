package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/trainingtools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerchecklistservice"
)

var _ trainingtools.ChecklistKeeper = (*workerchecklistservice.Service)(nil)
