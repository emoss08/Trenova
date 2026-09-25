package migrations

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/stretchr/testify/assert"
)

type eventKindsMove struct {
	migration string
	systemKey string
	from      []agent.EventKind
	to        []agent.EventKind
}

var systemAgentTemplates = map[string]agentdefinition.Template{
	"billing_exception":   agentdefinition.TemplateBillingException,
	"dispatch_assignment": agentdefinition.TemplateDispatchAssignment,
}

func systemAgentEventKindMoves() []eventKindsMove {
	return []eventKindsMove{
		{
			migration: "20261231006510_system_agent_event_kinds.tx",
			systemKey: "billing_exception",
			from:      []agent.EventKind{agent.EventBillingQueueItemException},
			to: []agent.EventKind{
				agent.EventBillingQueueItemException,
				agent.EventBillingQueueItemOnHold,
			},
		},
		{
			migration: "20261231006510_system_agent_event_kinds.tx",
			systemKey: "dispatch_assignment",
			from:      []agent.EventKind{agent.EventShipmentMoveCoverageAtRisk},
			to: []agent.EventKind{
				agent.EventShipmentMoveCoverageAtRisk,
				agent.EventShipmentMoveUnassigned,
			},
		},
		{
			migration: "20261231006620_billing_exception_accounting_event.tx",
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

func textArray(kinds []agent.EventKind) string {
	quoted := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		quoted = append(quoted, "'"+string(kind)+"'")
	}

	return "ARRAY[" + strings.Join(quoted, ", ") + "]::TEXT[]"
}

func TestSystemAgentEventKindsMigrations_MoveOnlyRowsStillOnThePreviousDefault(t *testing.T) {
	t.Parallel()

	for _, move := range systemAgentEventKindMoves() {
		up := compactSQL(readMigration(t, move.migration+".up.sql"))

		assert.Contains(t, up, fmt.Sprintf(`SET "event_kinds" = %s`, textArray(move.to)),
			"%s moves %s to its new events", move.migration, move.systemKey)
		assert.Contains(t, up, fmt.Sprintf(
			`WHERE "system_key" = '%s' AND "event_kinds" = %s`,
			move.systemKey, textArray(move.from),
		), "%s leaves %s alone once somebody changed its events", move.migration, move.systemKey)
	}
}

func TestSystemAgentEventKindsMigrations_DownRestoresThePreviousDefault(t *testing.T) {
	t.Parallel()

	for _, move := range systemAgentEventKindMoves() {
		down := compactSQL(readMigration(t, move.migration+".down.sql"))

		assert.Contains(t, down, fmt.Sprintf(`SET "event_kinds" = %s`, textArray(move.from)),
			"%s restores %s", move.migration, move.systemKey)
		assert.Contains(t, down, fmt.Sprintf(
			`WHERE "system_key" = '%s' AND "event_kinds" = %s`,
			move.systemKey, textArray(move.to),
		), "%s restores %s only from what it wrote", move.migration, move.systemKey)
	}
}

func TestSystemAgentEventKindsMigrations_EndOnWhatEachTemplateListensFor(t *testing.T) {
	t.Parallel()

	latest := make(map[string][]agent.EventKind, len(systemAgentTemplates))
	for _, move := range systemAgentEventKindMoves() {
		latest[move.systemKey] = move.to
	}

	for systemKey, template := range systemAgentTemplates {
		assert.Truef(t, slices.Equal(latest[systemKey], template.StarterEvents()),
			"%s's template listens for %v but the last migration leaves it on %v; add a migration "+
				"that moves rows still on the previous default",
			systemKey, template.StarterEvents(), latest[systemKey])
	}
}
