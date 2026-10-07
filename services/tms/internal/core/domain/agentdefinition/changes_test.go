package agentdefinition

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/editchange"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func baseDefinition() *Definition {
	budget := decimal.RequireFromString("50")
	return &Definition{
		ID:               pulid.MustNew(IDPrefix),
		Name:             "Billing desk",
		Instructions:     "Answer billing questions.",
		ToolNames:        []string{"read_invoice", "send_reminder"},
		ToolTiers:        map[string]agent.AutonomyTier{"send_reminder": agent.TierPropose},
		TriggerMode:      TriggerChat,
		MonthlyBudgetUSD: &budget,
		Enabled:          true,
		Version:          3,
	}
}

func TestChangesNamesEachSettingOnceInBuilderOrder(t *testing.T) {
	t.Parallel()

	before := baseDefinition()
	after := baseDefinition()
	after.Instructions = "Answer billing questions politely."
	after.TriggerMode = TriggerScheduled
	after.CronExpression = "0 7 * * 1-5"
	after.ShadowMode = true

	assert.Equal(t, []editchange.Change{
		{Field: "instructions", Label: "Instructions"},
		{Field: "triggerMode", Label: "When it runs"},
		{Field: "mode", Label: "Mode"},
	}, Changes(before, after))
}

func TestChangesIgnoresWhatNoPersonChanged(t *testing.T) {
	t.Parallel()

	before := baseDefinition()
	after := baseDefinition()
	after.ToolNames = []string{"send_reminder", "read_invoice"}
	sameBudget := decimal.RequireFromString("50.000")
	after.MonthlyBudgetUSD = &sameBudget
	after.Version = 9
	lastRun := int64(1_800_000_000)
	after.LastRunAt = &lastRun

	assert.Empty(t, Changes(before, after),
		"reordered grants, a budget spelled differently and bookkeeping are not changes")
	assert.Nil(t, Changes(nil, after), "a new agent has no changes of its own")
}

func TestChangeSummaryJoinsTheLabels(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Instructions, Tools", editchange.Summary([]editchange.Change{
		{Field: "instructions", Label: "Instructions"},
		{Field: "toolNames", Label: "Tools"},
	}))
	assert.Empty(t, editchange.Summary(nil))
}

func TestNewVersionSnapshotsTheSaveAndSaysWhatChanged(t *testing.T) {
	t.Parallel()

	before := baseDefinition()
	saved := baseDefinition()
	saved.Name = "Billing assistant"
	saved.Version = 4
	author := pulid.MustNew("usr_")

	version := NewVersion(saved, before, &author)

	assert.Equal(t, saved.ID, version.AgentDefinitionID)
	assert.Equal(t, int64(4), version.Version)
	assert.Equal(t, "Name", version.Summary)
	assert.Equal(t, &author, version.AuthorID)
	assert.Equal(t, "Billing assistant", version.Snapshot.Name)

	saved.Name = "Changed after"
	assert.Equal(t, "Billing assistant", version.Snapshot.Name, "the snapshot is a copy")

	assert.Equal(t, "Created", NewVersion(saved, nil, nil).Summary)
}
