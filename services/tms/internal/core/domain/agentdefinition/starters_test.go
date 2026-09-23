package agentdefinition_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func starterPrompts(starters []agentdefinition.Starter) []string {
	prompts := make([]string, 0, len(starters))
	for _, starter := range starters {
		prompts = append(prompts, starter.Prompt)
	}

	return prompts
}

func TestStarters_EveryTemplateOffersItsOwnQuestions(t *testing.T) {
	t.Parallel()

	for _, template := range agentdefinition.AllTemplates() {
		t.Run(string(template), func(t *testing.T) {
			t.Parallel()

			starters := agentdefinition.StartersFor(template, template.StarterTools())

			require.GreaterOrEqual(t, len(starters), 3)
			require.LessOrEqual(t, len(starters), agentdefinition.MaxStarters)
			for _, starter := range starters {
				assert.NotEmpty(t, starter.Label)
				assert.NotEmpty(t, starter.Prompt)
			}
		})
	}
}

func TestStarters_ReportBuilderAsksAboutReports(t *testing.T) {
	t.Parallel()

	definition := &agentdefinition.Definition{
		Name:      "Report Builder",
		ToolNames: []string{"create_report", "list_reports", "run_report", "describe_report_dataset"},
	}

	starters := definition.Starters()

	require.NotEmpty(t, starters)
	assert.Equal(t, "Build a report of in-transit shipments by customer.", starters[0].Prompt)
	assert.Contains(t, starterPrompts(starters), "Which reports can I run, and what does each one show?")
	assert.Contains(
		t,
		starterPrompts(starters),
		"Which datasets can I build a report from, and what does each one hold?",
	)
	assert.NotContains(t, starterPrompts(starters), "Which shipments are scheduled to pick up today?")
}

func TestStarters_TemplateQuestionsNeedTheirTools(t *testing.T) {
	t.Parallel()

	starters := agentdefinition.StartersFor(
		agentdefinition.TemplateDispatchAssistant,
		[]string{"list_time_off"},
	)

	prompts := starterPrompts(starters)
	assert.Contains(t, prompts, "Which drivers have time off booked this week?")
	assert.NotContains(
		t,
		prompts,
		"Rank the best available drivers for the next uncovered move and explain the top pick.",
	)
	assert.Len(t, starters, 1)
}

func TestStarters_CustomAgentMixesItsToolFamilies(t *testing.T) {
	t.Parallel()

	starters := agentdefinition.StartersFor("", []string{
		"list_reports",
		"create_report",
		"add_home_widget",
		"list_home_widgets",
		"get_dispatch_board",
	})

	require.Len(t, starters, agentdefinition.MaxStarters)
	prompts := starterPrompts(starters)
	assert.Equal(t, "Build a report of in-transit shipments by customer.", prompts[0])
	assert.Equal(t, "Add a widget to my home page that shows today's late shipments.", prompts[1])
	assert.Equal(t, "Which moves on the dispatch board are still unassigned for today?", prompts[2])
}

func TestStarters_AgentWithoutToolsGetsHowToQuestions(t *testing.T) {
	t.Parallel()

	starters := agentdefinition.StartersFor("", nil)

	require.Len(t, starters, 3)
	assert.Equal(t, "How do I create a new shipment, step by step?", starters[0].Prompt)
}

func TestStarters_NeverRepeatAQuestion(t *testing.T) {
	t.Parallel()

	starters := agentdefinition.StartersFor(
		agentdefinition.TemplateDispatchAssignment,
		agentdefinition.TemplateDispatchAssignment.StarterTools(),
	)

	seen := make(map[string]struct{}, len(starters))
	for _, starter := range starters {
		_, duplicate := seen[starter.Prompt]
		assert.Falsef(t, duplicate, "%q is offered twice", starter.Prompt)
		seen[starter.Prompt] = struct{}{}
	}
}
