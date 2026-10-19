package agentextension_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCommandReadsTheCommandAndTheWordsAfterIt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		content string
		command agentextension.Command
		task    string
		ok      bool
	}{
		{
			content: "/browser check the dock appointment at Acme DC",
			command: agentextension.CommandBrowser,
			task:    "check the dock appointment at Acme DC",
			ok:      true,
		},
		{
			content: "  /BROWSER   look up PO 4512  ",
			command: agentextension.CommandBrowser,
			task:    "look up PO 4512",
			ok:      true,
		},
		{
			content: "/browser\nbook the 9am slot",
			command: agentextension.CommandBrowser,
			task:    "book the 9am slot",
			ok:      true,
		},
		{
			content: "/browser open the carrier portal",
			command: agentextension.CommandBrowser,
			task:    "open the carrier portal",
			ok:      true,
		},
		{content: "/browser", command: agentextension.CommandBrowser, ok: true},
		{content: "/browser   ", command: agentextension.CommandBrowser, ok: true},
		{
			content: "/search FMCSA hours of service changes this year",
			command: agentextension.CommandSearch,
			task:    "FMCSA hours of service changes this year",
			ok:      true,
		},
		{
			content: " /Search\tdiesel price outlook ",
			command: agentextension.CommandSearch,
			task:    "diesel price outlook",
			ok:      true,
		},
		{content: "/search", command: agentextension.CommandSearch, ok: true},
		{
			content: "/watch FMCSA finalizing the broker transparency rule",
			command: agentextension.CommandWatch,
			task:    "FMCSA finalizing the broker transparency rule",
			ok:      true,
		},
		{content: "/WATCH", command: agentextension.CommandWatch, ok: true},
		{content: "/watching the news"},
		{content: "/browsers list them"},
		{content: "/browse the portal"},
		{content: "/searching for it"},
		{content: "use /browser to check"},
		{content: "search /search"},
		{content: "/"},
		{content: ""},
		{content: "/status S12345"},
	}
	for _, tc := range cases {
		command, task, ok := agentextension.ParseCommand(tc.content)
		assert.Equal(t, tc.ok, ok, tc.content)
		assert.Equal(t, tc.command, command, tc.content)
		assert.Equal(t, tc.task, task, tc.content)
	}
}

func TestEveryCommandIsServedByItsCapability(t *testing.T) {
	t.Parallel()

	for _, command := range agentextension.AllCommands() {
		assert.True(t, command.IsValid(), command)
		assert.Equal(t, "/"+command.String(), command.Text())

		spec, ok := agentextension.CommandSpecFor(command)
		require.True(t, ok, command)
		capability, ok := agentextension.CapabilitySpecFor(spec.Capability)
		require.True(t, ok, command)
		require.NotEmpty(t, spec.Requires, command)
		for _, name := range spec.Requires {
			assert.True(t, capability.HasTool(name), "%s requires %s", command, name)
		}
		for _, name := range spec.Honours {
			assert.True(t, capability.HasTool(name), "%s is honoured by %s", command, name)
		}
		assert.NotEmpty(t, spec.MissingTaskReason, command)
		assert.NotEmpty(t, spec.DirectedReason, command)
		assert.Contains(t, spec.UnavailableReason, "{0}", command)
	}
}

func TestHonouredByNeedsOneOfTheToolsTheCommandRequires(t *testing.T) {
	t.Parallel()

	browser, _ := agentextension.CommandSpecFor(agentextension.CommandBrowser)
	assert.True(t, browser.HonouredBy([]string{
		agentextension.ToolListPortalSites, agentextension.ToolReadPortal,
	}))
	assert.True(t, browser.HonouredBy([]string{agentextension.ToolActOnPortal}))
	assert.False(t, browser.HonouredBy([]string{agentextension.ToolListPortalSites}))
	assert.False(t, browser.HonouredBy([]string{agentextension.ToolWebSearch}))
	assert.False(t, browser.HonouredBy(nil))

	search, _ := agentextension.CommandSpecFor(agentextension.CommandSearch)
	assert.True(t, search.HonouredBy([]string{agentextension.ToolWebSearch}))
	assert.False(t, search.HonouredBy([]string{agentextension.ToolWebRead}))
	assert.False(t, search.HonouredBy([]string{agentextension.ToolReadPortal}))
}

func TestToolsInKeepsTheCapabilitysToolsInItsOwnOrder(t *testing.T) {
	t.Parallel()

	search, _ := agentextension.CommandSpecFor(agentextension.CommandSearch)
	assert.Equal(t,
		[]string{agentextension.ToolWebSearch, agentextension.ToolWebRead},
		search.ToolsIn([]string{
			"worker_list", agentextension.ToolWebRead, agentextension.ToolWebSearch,
			agentextension.ToolReadPortal,
		}),
	)
	assert.Empty(t, search.ToolsIn([]string{"worker_list"}))
}

func TestAWatchIsSetOnlyWithTheWatchTools(t *testing.T) {
	t.Parallel()

	watch, _ := agentextension.CommandSpecFor(agentextension.CommandWatch)
	assert.Equal(t, agentextension.CapabilityWebResearch, watch.Capability)
	assert.True(t, watch.HonouredBy([]string{agentextension.ToolWatchWeb}))
	assert.False(t, watch.HonouredBy([]string{agentextension.ToolWebSearch}),
		"a vendor that cannot watch leaves the command out")

	for _, name := range agentextension.WebWatchTools {
		assert.True(t, watch.HonouredByCall(name), name)
	}
	assert.False(t, watch.HonouredByCall(agentextension.ToolWebSearch),
		"a search alone does not stand in for setting a watch")
	assert.False(t, watch.HonouredByCall(agentextension.ToolReadPortal))
}

func TestACommandWithoutItsOwnToolsIsHonouredByAnyOfItsCapability(t *testing.T) {
	t.Parallel()

	search, _ := agentextension.CommandSpecFor(agentextension.CommandSearch)
	assert.True(t, search.HonouredByCall(agentextension.ToolWebSearch))
	assert.True(t, search.HonouredByCall(agentextension.ToolWebRead))
	assert.False(t, search.HonouredByCall(agentextension.ToolReadPortal))
	assert.False(t, search.HonouredByCall("worker_list"))

	browser, _ := agentextension.CommandSpecFor(agentextension.CommandBrowser)
	assert.True(t, browser.HonouredByCall(agentextension.ToolListPortalSites))
	assert.False(t, browser.HonouredByCall(agentextension.ToolWebSearch))
}
