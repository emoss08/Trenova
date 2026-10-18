package agentextension_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/stretchr/testify/assert"
)

func TestBrowserTaskReadsTheWordsAfterTheCommand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		content string
		task    string
		command bool
	}{
		{
			content: "/browser check the dock appointment at Acme DC",
			task:    "check the dock appointment at Acme DC",
			command: true,
		},
		{content: "  /BROWSER   look up PO 4512  ", task: "look up PO 4512", command: true},
		{content: "/browser\nbook the 9am slot", task: "book the 9am slot", command: true},
		{
			content: "/browser\u00a0open the carrier portal",
			task:    "open the carrier portal",
			command: true,
		},
		{content: "/browser", task: "", command: true},
		{content: "/browser   ", task: "", command: true},
		{content: "/browsers list them", task: "", command: false},
		{content: "/browse the portal", task: "", command: false},
		{content: "use /browser to check", task: "", command: false},
		{content: "", task: "", command: false},
		{content: "/status S12345", task: "", command: false},
	}
	for _, tc := range cases {
		task, command := agentextension.BrowserTask(tc.content)
		assert.Equal(t, tc.command, command, tc.content)
		assert.Equal(t, tc.task, task, tc.content)
	}
}

func TestWorksInBrowserNeedsAToolThatOpensAPortal(t *testing.T) {
	t.Parallel()

	assert.True(t, agentextension.WorksInBrowser([]string{
		agentextension.ToolListPortalSites, agentextension.ToolReadPortal,
	}))
	assert.True(t, agentextension.WorksInBrowser([]string{agentextension.ToolActOnPortal}))
	assert.False(t, agentextension.WorksInBrowser([]string{agentextension.ToolListPortalSites}))
	assert.False(t, agentextension.WorksInBrowser([]string{agentextension.ToolWebSearch}))
	assert.False(t, agentextension.WorksInBrowser(nil))
}
