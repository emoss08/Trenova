package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/safetytools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerinjuryservice"
)

var _ safetytools.InjuryKeeper = (*workerinjuryservice.Service)(nil)
