package agentprompt_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprompt"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttaskframe"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var toolShaped = regexp.MustCompile(`\b[a-z]+_[a-z_]+\b`)

func playbookText(playbook *agenttaskframe.Playbook) string {
	return strings.Join(append([]string{playbook.Goal, playbook.Done, playbook.Need},
		playbook.Steps...), "\n")
}

func assertPlaybook(t *testing.T, name string, playbook *agenttaskframe.Playbook) {
	t.Helper()

	assert.NotEmpty(t, playbook.ID, name)
	assert.NotEmpty(t, playbook.Goal, name)
	assert.NotEmpty(t, playbook.Steps, name)
	assert.NotEmpty(t, playbook.Done, name)
	assert.NotEmpty(t, playbook.Need, name)
	assert.Empty(t, toolShaped.FindAllString(playbookText(playbook), -1),
		"%s is written in words, never in tool names, which another change may rename", name)
}

func TestEventPlaybook_EveryEventAnAgentCanStartOnHasOne(t *testing.T) {
	t.Parallel()

	for _, event := range agent.KnownEvents() {
		playbook, ok := agenttaskframe.EventPlaybook(event.Kind)
		require.True(t, ok, "%s has no playbook", event.Kind)
		assertPlaybook(t, string(event.Kind), &playbook)
	}
}

func TestScheduledPlaybook_EveryScheduledStarterHasOne(t *testing.T) {
	t.Parallel()

	for _, template := range agenttemplate.AllTemplates() {
		if agentpreset.StarterTrigger(template) != agentdefinition.TriggerScheduled {
			continue
		}
		playbook, ok := agenttaskframe.ScheduledPlaybook(template)
		require.True(t, ok, "%s runs on a schedule and has no playbook", template)
		assertPlaybook(t, string(template), &playbook)
	}
}

func TestBackgroundFrame_IsBuiltFromWhatStartedTheRun(t *testing.T) {
	t.Parallel()

	event := agenttaskframe.BackgroundFrame(agent.RunTriggerEvent,
		agent.EventBankReceiptException, agenttemplate.TemplateCashApplication)
	require.NotNil(t, event)
	assert.Equal(t, "apply-a-bank-receipt", event.Playbook)
	assert.False(t, event.Inferred, "a frame read off the trigger is not a guess")

	monitor := agenttaskframe.BackgroundFrame(agent.RunTriggerScheduled, "",
		agenttemplate.TemplateLoadMonitor)
	require.NotNil(t, monitor)
	assert.Equal(t, "watch-the-board", monitor.Playbook)

	other := agenttaskframe.BackgroundFrame(agent.RunTriggerContinuous, "",
		agenttemplate.TemplateBillingAssistant)
	require.NotNil(t, other)
	assert.Equal(t, "scheduled-pass", other.Playbook)

	wait := agenttaskframe.BackgroundFrame(agent.RunTriggerWait, "",
		agenttemplate.TemplateLoadMonitor)
	require.NotNil(t, wait)
	assert.Equal(t, "wait-ended", wait.Playbook)

	assert.Nil(t, agenttaskframe.BackgroundFrame(agent.RunTriggerManual, "",
		agenttemplate.TemplateLoadMonitor), "a person who started a run says what it is for")
	assert.Nil(t, agenttaskframe.BackgroundFrame(agent.RunTriggerEvent,
		agent.EventKind("shipment.teleported"), ""))
}

func TestChatFrame_KeepsTheClassifierToOneLineEach(t *testing.T) {
	t.Parallel()

	assert.Nil(t, agenttaskframe.ChatFrame("  ", "\n"), "nothing read, nothing framed")

	frame := agenttaskframe.ChatFrame("List the\n trailers  due.", strings.Repeat("trailer ", 80))
	require.NotNil(t, frame)
	assert.True(t, frame.Inferred)
	assert.Equal(t, "List the trailers due.", frame.Goal)
	assert.LessOrEqual(t, len([]rune(frame.Need)), 300)
	assert.NotContains(t, frame.Need, "\n")
}

func TestBuildSystemPrompt_PutsTheTaskInTheTurnsOwnPart(t *testing.T) {
	t.Parallel()

	d := definitionWithInstructions("Cover moves.")
	frame := agenttaskframe.BackgroundFrame(agent.RunTriggerEvent,
		agent.EventShipmentMoveUnassigned, agenttemplate.TemplateDispatchAssignment)
	parts := agentprompt.BuildSystemPromptParts(d, &agentturn.RuntimeContext{
		Trigger: agent.RunTriggerEvent,
		Frame:   frame,
	})

	assert.NotContains(t, parts.Stable, "## This task")
	assert.True(t, strings.HasPrefix(strings.TrimSpace(parts.Volatile), "## This task"),
		"the task opens the turn's own part, ahead of its context")
	assert.Contains(t, parts.Volatile, "Goal: "+frame.Goal)
	assert.Contains(t, parts.Volatile, "(playbook cover-a-move)")
	assert.Contains(t, parts.Volatile, "1. "+frame.Steps[0])
	assert.Contains(t, parts.Volatile, "Done when: "+frame.Done)
	assert.NotContains(
		t,
		parts.Volatile,
		frame.Need,
		"the need ranks tools; the model is not told it",
	)

	chat := agentprompt.BuildSystemPromptParts(d, &agentturn.RuntimeContext{
		Frame: agenttaskframe.ChatFrame("Find the late loads.", "list late loads"),
	})
	assert.Contains(t, chat.Volatile, "What the person wants: Find the late loads.")
	assert.Contains(t, chat.Volatile, "the message wins", "a guess never outranks the message")
	assert.NotContains(t, chat.Volatile, "Done when")

	assert.NotContains(
		t,
		agentprompt.BuildSystemPrompt(d, agentturn.RuntimeContext{}),
		"## This task",
	)
}

func TestBuildSystemPrompt_TeachesTheDomainOnceInTheCachedPart(t *testing.T) {
	t.Parallel()

	for _, template := range []agenttemplate.Template{
		agenttemplate.TemplateGeneralAssistant,
		agenttemplate.TemplateBillingException,
	} {
		d := snapshotlessDefinition(template)
		parts := agentprompt.BuildSystemPromptParts(d, &agentturn.RuntimeContext{})
		assert.Equal(t, 1, strings.Count(parts.Stable, "## How Trenova's records fit together"),
			string(template))
		assert.NotContains(t, parts.Volatile, "## How Trenova's records fit together")
	}
}

func snapshotlessDefinition(template agenttemplate.Template) *agentdefinition.Definition {
	d := definitionWithInstructions(agenttemplate.Description(template))
	d.Template = template

	return d
}
