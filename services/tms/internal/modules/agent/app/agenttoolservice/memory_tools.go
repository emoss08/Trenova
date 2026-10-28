package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
)

var _ agentmemoryports.MemoryRecordingTool = (*masterdatatools.RememberTool)(nil)
