package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/safetytools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
)

var _ safetytools.SafetyKeeper = (*workersafetyservice.Service)(nil)
