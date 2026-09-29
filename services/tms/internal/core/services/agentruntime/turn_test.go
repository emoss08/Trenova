package agentruntime

import (
	"regexp"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var clockLinePattern = regexp.MustCompile(
	`^Now: \d{4}-\d{2}-\d{2} \d{2}:\d{2} America/Chicago\n\n`,
)

func openedTurn(t *testing.T, definition *agentdefinition.Definition, input string) *Turn {
	t.Helper()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)

	return rt.OpenTurn(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      input,
		Context: agentdefinition.RuntimeContext{
			Timezone: "America/Chicago",
			Now:      1790000000,
		},
	})
}

func TestOpenTurn_PrefixesTheQuestionWithTheTimeInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	turn := openedTurn(t, testDefinition(), "what is due before noon?")

	asked := turn.messages[len(turn.messages)-1]
	require.Equal(t, serviceports.RoleUser, asked.Role)
	assert.Regexp(t, clockLinePattern, asked.Content)
	assert.True(t, strings.HasSuffix(asked.Content, "what is due before noon?"))
	assert.Equal(t, "what is due before noon?", turn.result.Messages[0].Content,
		"the conversation keeps what the person typed")
}

func TestOpenTurn_KeepsTheSystemPromptClockToTheDay(t *testing.T) {
	t.Parallel()

	turn := openedTurn(t, testDefinition(), "what is due before noon?")

	assert.Contains(t, turn.system, "Today is: ")
	assert.NotRegexp(t, regexp.MustCompile(`Today is: [^\n]*\d{2}:\d{2}`), turn.system,
		"the time of day would change the cached prefix every minute")
}

func TestOpenTurn_LeavesTheQuestionAloneWhenTheAgentReadsNoClock(t *testing.T) {
	t.Parallel()

	definition := testDefinition()
	definition.ContextProviders = []agentdefinition.ContextProvider{
		agentdefinition.ContextOrganization,
	}

	turn := openedTurn(t, definition, "what is due before noon?")

	assert.Equal(t, "what is due before noon?", turn.messages[len(turn.messages)-1].Content)
}

func TestClockLine_FallsBackToUTCForAnUnknownZone(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Now: 2026-09-21 14:13 UTC\n\n", clockLine(1790000000, "Mars/Olympus"))
	assert.Equal(t, "Now: 2026-09-21 09:13 America/Chicago\n\n",
		clockLine(1790000000, "America/Chicago"))
}
