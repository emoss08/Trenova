package migrations

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/stretchr/testify/assert"
)

type systemAgentEventMove struct {
	migration string
	systemKey string
	from      []agent.EventKind
	to        []agent.EventKind
}

type systemAgentTemplate struct {
	systemKey string
	template  agentdefinition.Template
}

func systemAgentEventMoves() []systemAgentEventMove {
	return []systemAgentEventMove{
		{
			migration: "20261231006510_system_agent_event_kinds",
			systemKey: "billing_exception",
			from:      []agent.EventKind{agent.EventBillingQueueItemException},
			to: []agent.EventKind{
				agent.EventBillingQueueItemException,
				agent.EventBillingQueueItemOnHold,
			},
		},
		{
			migration: "20261231006510_system_agent_event_kinds",
			systemKey: "dispatch_assignment",
			from:      []agent.EventKind{agent.EventShipmentMoveCoverageAtRisk},
			to: []agent.EventKind{
				agent.EventShipmentMoveCoverageAtRisk,
				agent.EventShipmentMoveUnassigned,
			},
		},
		{
			migration: "20261231006660_billing_exception_accounting_event",
			systemKey: "billing_exception",
			from: []agent.EventKind{
				agent.EventBillingQueueItemException,
				agent.EventBillingQueueItemOnHold,
			},
			to: []agent.EventKind{
				agent.EventBillingQueueItemException,
				agent.EventBillingQueueItemOnHold,
				agent.EventAccountingConnectionDegraded,
			},
		},
	}
}

func systemAgentTemplates() []systemAgentTemplate {
	return []systemAgentTemplate{
		{systemKey: "billing_exception", template: agentdefinition.TemplateBillingException},
		{systemKey: "dispatch_assignment", template: agentdefinition.TemplateDispatchAssignment},
	}
}

func textArray(kinds []agent.EventKind) string {
	quoted := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		quoted = append(quoted, "'"+string(kind)+"'")
	}

	return "ARRAY[" + strings.Join(quoted, ", ") + "]::TEXT[]"
}

func TestSystemAgentEventKindsMigrations_MoveOnlyRowsStillOnThePreviousDefault(t *testing.T) {
	t.Parallel()

	for _, move := range systemAgentEventMoves() {
		up := compactSQL(readMigration(t, move.migration+".tx.up.sql"))

		assert.Contains(t, up, fmt.Sprintf(
			`SET "event_kinds" = %s`, textArray(move.to),
		), "%s: %s is moved to its new events", move.migration, move.systemKey)
		assert.Contains(t, up, fmt.Sprintf(
			`WHERE "system_key" = '%s' AND "event_kinds" = %s`,
			move.systemKey, textArray(move.from),
		), "%s: %s is left alone once somebody changed its events", move.migration, move.systemKey)
	}
}

func TestSystemAgentEventKindsMigrations_DownRestoresThePreviousDefault(t *testing.T) {
	t.Parallel()

	for _, move := range systemAgentEventMoves() {
		down := compactSQL(readMigration(t, move.migration+".tx.down.sql"))

		assert.Contains(t, down, fmt.Sprintf(
			`SET "event_kinds" = %s`, textArray(move.from),
		), "%s: %s", move.migration, move.systemKey)
		assert.Contains(t, down, fmt.Sprintf(
			`WHERE "system_key" = '%s' AND "event_kinds" = %s`,
			move.systemKey, textArray(move.to),
		), "%s: %s", move.migration, move.systemKey)
	}
}

func TestSystemAgentEventKindsMigrations_LatestMoveMatchesTheTemplate(t *testing.T) {
	t.Parallel()

	moves := systemAgentEventMoves()
	for _, system := range systemAgentTemplates() {
		var latest []agent.EventKind
		for _, move := range moves {
			if move.systemKey == system.systemKey {
				latest = move.to
			}
		}

		assert.Equal(t, system.template.StarterEvents(), latest,
			"%s listens for events its seeded rows never receive; add a migration that moves them",
			system.systemKey)
	}
}
