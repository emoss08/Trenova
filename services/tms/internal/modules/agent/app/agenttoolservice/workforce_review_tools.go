package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/reviewtools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
)

var _ reviewtools.ReviewKeeper = (*performancereviewservice.Service)(nil)
