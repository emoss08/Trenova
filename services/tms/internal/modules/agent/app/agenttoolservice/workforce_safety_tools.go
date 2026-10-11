package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
)

var _ workforcetools.SafetyKeeper = (*workersafetyservice.Service)(nil)
