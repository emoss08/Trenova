package agentextension_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryTypeHasASpecOnAKnownCapability(t *testing.T) {
	t.Parallel()

	for _, typ := range agentextension.AllTypes() {
		spec, ok := agentextension.SpecFor(typ)
		require.True(t, ok, typ)
		require.True(t, spec.Capability.IsValid(), typ)
		assert.NotNil(t, spec.Validate, typ)
		assert.NotEmpty(t, spec.Tools(), typ)
		assert.Contains(t, agentextension.TypesFor(spec.Capability), typ)

		keys := make(map[string]bool, len(spec.Fields))
		for _, field := range spec.Fields {
			keys[field.Key] = true
		}
		assert.True(t, keys[agentextension.ConfigKeyAPIKey], "%s needs an API key field", typ)
		assert.True(t, keys[agentextension.ConfigKeyDailyRequestLimit],
			"%s needs a daily request limit so metering applies", typ)
	}
}

func TestEveryToolBelongsToOneCapability(t *testing.T) {
	t.Parallel()

	owners := make(map[string]agentextension.Capability)
	for _, capability := range agentextension.AllCapabilities() {
		spec, ok := agentextension.CapabilitySpecFor(capability)
		require.True(t, ok, capability)
		assert.NotEmpty(t, spec.Unchecked, capability)
		assert.NotEmpty(t, spec.RequestNoun, capability)
		for _, tool := range spec.Tools {
			previous, taken := owners[tool]
			assert.False(t, taken, "%s is claimed by %s and %s", tool, previous, capability)
			owners[tool] = capability

			owner, found := agentextension.CapabilityForTool(tool)
			assert.True(t, found, tool)
			assert.Equal(t, capability, owner, tool)
		}
	}
}

func TestCapabilityForToolIgnoresOtherTools(t *testing.T) {
	t.Parallel()

	_, found := agentextension.CapabilityForTool("get_shipment")
	assert.False(t, found)
	assert.False(t, agentextension.IsExtensionTool("get_shipment"))
	assert.False(t, agentextension.ReturnsExternalContent("get_shipment"))
	assert.True(t, agentextension.ReturnsExternalContent(agentextension.ToolWebSearch))
}

func readyExa(availability agentextension.Availability) *agentextension.Extension {
	return &agentextension.Extension{
		Type:          agentextension.TypeExa,
		Enabled:       true,
		Availability:  availability,
		Configuration: map[string]any{agentextension.ConfigKeyAPIKey: "sealed"},
	}
}

func TestActiveFromNeedsOnAndSetUp(t *testing.T) {
	t.Parallel()

	off := readyExa(agentextension.AvailabilityAllAgents)
	off.Enabled = false
	assert.Empty(t, agentextension.ActiveFrom([]*agentextension.Extension{off}))

	unset := readyExa(agentextension.AvailabilityAllAgents)
	unset.Configuration = map[string]any{}
	assert.Empty(t, agentextension.ActiveFrom([]*agentextension.Extension{unset}))

	active := agentextension.ActiveFrom(
		[]*agentextension.Extension{readyExa(agentextension.AvailabilitySelectedAgents)},
	)
	assert.Equal(t, agentextension.ActiveExtension{
		Type:         agentextension.TypeExa,
		Availability: agentextension.AvailabilitySelectedAgents,
	}, active[agentextension.CapabilityWebResearch])
}

func TestActiveOffersGrantsAndServes(t *testing.T) {
	t.Parallel()

	none := agentextension.Active{}
	assert.True(t, none.Offers("get_shipment"))
	assert.False(t, none.Offers(agentextension.ToolWebSearch))
	assert.Empty(t, none.Grants())

	chosen := agentextension.ActiveFrom(
		[]*agentextension.Extension{readyExa(agentextension.AvailabilitySelectedAgents)},
	)
	assert.True(t, chosen.Offers(agentextension.ToolWebRead))
	assert.Empty(t, chosen.Grants())
	assert.False(t, chosen.GrantsEveryAgent(agentextension.ToolWebSearch))

	everyone := agentextension.ActiveFrom(
		[]*agentextension.Extension{readyExa(agentextension.AvailabilityAllAgents)},
	)
	assert.Equal(t,
		[]string{
			agentextension.ToolWebSearch, agentextension.ToolWebRead,
			agentextension.ToolWatchWeb, agentextension.ToolListWebWatches,
			agentextension.ToolStopWebWatch,
		},
		everyone.Grants(),
		"Exa can watch the web, so every agent is given the watch tools too",
	)
	assert.True(t, everyone.GrantsEveryAgent(agentextension.ToolWebRead))
	served, ok := everyone.Serving(agentextension.ToolWebSearch)
	require.True(t, ok)
	assert.Equal(t, agentextension.TypeExa, served.Type)
}

func TestParseExaSettingsDefaults(t *testing.T) {
	t.Parallel()

	settings, err := agentextension.ParseExaSettings(map[string]string{"apiKey": " key "})
	require.NoError(t, err)
	assert.Equal(t, "key", settings.APIKey)
	assert.Equal(t, agentextension.ExaSearchTypeAuto, settings.SearchType)
	assert.Equal(t, agentextension.DefaultResultsPerSearch, settings.ResultsPerSearch)
	assert.Equal(t, agentextension.DefaultDailyRequestLimit, settings.DailyRequestLimit)
	assert.Empty(t, settings.ExcludedDomains)
}

func TestParseExaSettingsNormalizesExcludedSites(t *testing.T) {
	t.Parallel()

	settings, err := agentextension.ParseExaSettings(map[string]string{
		"excludedDomains": "Example.com, *.spam.example.org\nexample.com; blog.vendor.io.",
	})
	require.NoError(t, err)
	assert.Equal(t,
		[]string{"example.com", "*.spam.example.org", "blog.vendor.io"},
		settings.ExcludedDomains,
	)
}

func TestParseExaSettingsRejectsBadValues(t *testing.T) {
	t.Parallel()

	_, err := agentextension.ParseExaSettings(map[string]string{
		"searchType":        "exhaustive",
		"resultsPerSearch":  "25",
		"dailyRequestLimit": "zero",
		"excludedDomains":   "https://example.com/path",
	})
	require.Error(t, err)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t, []string{
		"configuration.searchType",
		"configuration.resultsPerSearch",
		"configuration.dailyRequestLimit",
		"configuration.excludedDomains",
	}, fields)
}

func TestValidateSettingsUsesTheTypesValidator(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	agentextension.ValidateSettings(
		agentextension.TypeExa,
		map[string]string{"searchType": "exhaustive"},
		multiErr,
	)
	require.True(t, multiErr.HasErrors())
	assert.Equal(t, "configuration.searchType", multiErr.Errors[0].Field)
}

func TestConfiguredNeedsTheAPIKey(t *testing.T) {
	t.Parallel()

	spec, _ := agentextension.SpecFor(agentextension.TypeExa)
	assert.False(t, spec.Configured(map[string]any{"searchType": "auto"}))
	assert.True(t, spec.Configured(map[string]any{"apiKey": "sealed"}))
}
