package agentextension_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryTypeHasASpec(t *testing.T) {
	t.Parallel()

	for _, typ := range agentextension.AllTypes() {
		spec, ok := agentextension.SpecFor(typ)
		require.True(t, ok, typ)
		assert.NotEmpty(t, spec.Tools, typ)
		for _, tool := range spec.Tools {
			owner, found := agentextension.ExtensionForTool(tool)
			assert.True(t, found, tool)
			assert.Equal(t, typ, owner, tool)
		}
	}
}

func TestExtensionForToolIgnoresOtherTools(t *testing.T) {
	t.Parallel()

	_, found := agentextension.ExtensionForTool("get_shipment")
	assert.False(t, found)
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

func TestConfiguredNeedsTheAPIKey(t *testing.T) {
	t.Parallel()

	spec, _ := agentextension.SpecFor(agentextension.TypeExa)
	assert.False(t, spec.Configured(map[string]any{"searchType": "auto"}))
	assert.True(t, spec.Configured(map[string]any{"apiKey": "sealed"}))
}
