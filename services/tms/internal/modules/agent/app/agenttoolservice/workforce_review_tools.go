package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/performancereviewservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
)

var _ workforcetools.ReviewKeeper = (*performancereviewservice.Service)(nil)
