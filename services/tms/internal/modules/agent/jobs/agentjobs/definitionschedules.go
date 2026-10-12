package agentjobs

import (
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs/agentcontract"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
)

var _ agentdefinitionports.AgentDefinitionScheduler = (*agentcontract.DefinitionSchedules)(nil)
