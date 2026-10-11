package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/workerchecklistservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
)

var _ workforcetools.ChecklistKeeper = (*workerchecklistservice.Service)(nil)
