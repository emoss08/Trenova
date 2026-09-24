package migrations

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/stretchr/testify/assert"
)

const (
	systemAgentEventKindsUp   = "20261231006510_system_agent_event_kinds.tx.up.sql"
	systemAgentEventKindsDown = "20261231006510_system_agent_event_kinds.tx.down.sql"
)

type seededEvents struct {
	systemKey string
	template  agentdefinition.Template
	seeded    []agent.EventKind
}

func seededSystemAgentEvents() []seededEvents {
	return []seededEvents{
		{
			systemKey: "billing_exception",
			template:  agentdefinition.TemplateBillingException,
			seeded:    []agent.EventKind{agent.EventBillingQueueItemException},
		},
		{
			systemKey: "dispatch_assignment",
			template:  agentdefinition.TemplateDispatchAssignment,
			seeded:    []agent.EventKind{agent.EventShipmentMoveCoverageAtRisk},
		},
	}
}

func textArray(kinds []agent.EventKind) string {
	quoted := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		quoted = append(quoted, "'"+string(kind)+"'")
	}

	return "ARRAY[" + strings.Join(quoted, ", ") + "]::TEXT[]"
}

func TestSystemAgentEventKindsMigration_MovesOnlyRowsStillOnTheSeededDefault(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, systemAgentEventKindsUp))

	for _, agentEvents := range seededSystemAgentEvents() {
		assert.Contains(t, up, fmt.Sprintf(
			`SET "event_kinds" = %s`, textArray(agentEvents.template.StarterEvents()),
		), "%s is moved to what its template now listens for", agentEvents.systemKey)
		assert.Contains(t, up, fmt.Sprintf(
			`WHERE "system_key" = '%s' AND "event_kinds" = %s`,
			agentEvents.systemKey, textArray(agentEvents.seeded),
		), "%s is left alone once somebody changed its events", agentEvents.systemKey)
	}
}

func TestSystemAgentEventKindsMigration_DownRestoresTheSeededDefault(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, systemAgentEventKindsDown))

	for _, agentEvents := range seededSystemAgentEvents() {
		assert.Contains(t, down, fmt.Sprintf(
			`SET "event_kinds" = %s`, textArray(agentEvents.seeded),
		), agentEvents.systemKey)
		assert.Contains(t, down, fmt.Sprintf(
			`WHERE "system_key" = '%s' AND "event_kinds" = %s`,
			agentEvents.systemKey, textArray(agentEvents.template.StarterEvents()),
		), agentEvents.systemKey)
	}
}
