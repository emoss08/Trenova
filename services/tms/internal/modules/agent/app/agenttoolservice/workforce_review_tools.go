package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
)

var _ workforcetools.ReviewKeeper = (*performancereviewservice.Service)(nil)
