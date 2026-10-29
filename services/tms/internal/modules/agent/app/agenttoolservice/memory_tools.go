package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/memorytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
)

var _ agentmemoryports.MemoryRecordingTool = (*memorytools.RememberTool)(nil)
