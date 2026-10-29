package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/timeofftools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerleaveservice"
)

var _ timeofftools.LeaveKeeper = (*workerleaveservice.Service)(nil)
