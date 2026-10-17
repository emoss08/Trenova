package agentextension_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckAnalysisCodeRefusesWhatCannotRunAsOneCell(t *testing.T) {
	t.Parallel()

	require.NoError(t, agentextension.CheckAnalysisCode("import pandas as pd\nprint(1)"))
	require.ErrorIs(t, agentextension.CheckAnalysisCode("  \n "), agentextension.ErrAnalysisCodeEmpty)
	require.ErrorIs(t,
		agentextension.CheckAnalysisCode(strings.Repeat("x", agentextension.MaxAnalysisCodeRunes+1)),
		agentextension.ErrAnalysisCodeTooLong,
	)
}

func TestCheckDatasetNameKeepsNamesTheCodeCanOpenWithoutQuoting(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"lanes", "q3_margin", "a1"} {
		require.NoError(t, agentextension.CheckDatasetName(name), name)
	}
	for _, name := range []string{"", "Lanes", "1lanes", "lanes.csv", "../lanes", "lane-margin",
		strings.Repeat("a", agentextension.MaxAnalysisDatasetName+1)} {
		require.Error(t, agentextension.CheckDatasetName(name), name)
	}
}

func TestDatasetPathPutsEveryDatasetInTheDataFolder(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/home/user/data/lanes.csv", agentextension.DatasetPath("lanes", "csv"))
}

func TestAnalysisRunFinishBoundsWhatItKeeps(t *testing.T) {
	t.Parallel()

	results := make([]string, 0, agentextension.MaxAnalysisResults+3)
	for range agentextension.MaxAnalysisResults + 3 {
		results = append(results, "0.18")
	}
	results = append([]string{"  "}, results...)

	run := &agentextension.AnalysisRun{Status: agentextension.AnalysisRunStatusRunning}
	run.Finish(&agentextension.AnalysisRunOutcome{
		Status:     agentextension.AnalysisRunStatusErrored,
		Stdout:     strings.Repeat("a", agentextension.MaxAnalysisOutputRunes+50),
		Results:    results,
		Error:      &agentextension.AnalysisError{Name: "KeyError", Value: "'margin'", Traceback: "tb"},
		ChartCount: agentextension.MaxAnalysisCharts + 2,
		DurationMs: 1200,
		CostUSD:    decimal.RequireFromString("0.0004"),
	}, 1_790_000_000)

	assert.Equal(t, agentextension.AnalysisRunStatusErrored, run.Status)
	assert.Len(t, []rune(run.Stdout), agentextension.MaxAnalysisOutputRunes)
	assert.Len(t, run.Results, agentextension.MaxAnalysisResults)
	assert.Equal(t, "0.18", run.Results[0], "a blank result is not kept")
	assert.Equal(t, "KeyError", run.ErrorName)
	assert.Equal(t, "'margin'", run.ErrorValue)
	assert.Equal(t, agentextension.MaxAnalysisCharts, run.ChartCount)
	assert.Equal(t, int64(1200), run.DurationMs)
	require.NotNil(t, run.FinishedAt)
	assert.True(t, run.Status.Finished())
	assert.False(t, agentextension.AnalysisRunStatusRunning.Finished())
}

func TestAnalysisRunFinishClearsAnEarlierError(t *testing.T) {
	t.Parallel()

	run := &agentextension.AnalysisRun{ErrorName: "KeyError", ErrorValue: "x", Traceback: "tb"}
	run.Finish(&agentextension.AnalysisRunOutcome{Status: agentextension.AnalysisRunStatusSucceeded}, 1)

	assert.Empty(t, run.ErrorName)
	assert.Empty(t, run.ErrorValue)
	assert.Empty(t, run.Traceback)
	assert.Equal(t, []string{}, run.Results)
}

func TestParseE2BSettingsDefaultsAndRefusals(t *testing.T) {
	t.Parallel()

	settings, err := agentextension.ParseE2BSettings(map[string]string{"apiKey": " e2b_key "})
	require.NoError(t, err)
	assert.Equal(t, "e2b_key", settings.APIKey)
	assert.Equal(t, agentextension.DefaultAnalysisRunSeconds, settings.RunSeconds)
	assert.Equal(t, agentextension.DefaultDailyAnalyses, settings.DailyRequestLimit)
	assert.Equal(t, agentextension.DefaultE2BTemplate, settings.Template)

	custom, err := agentextension.ParseE2BSettings(map[string]string{
		"runSeconds":      "45",
		"sandboxTemplate": "lanes-v2:latest",
	})
	require.NoError(t, err)
	assert.Equal(t, 45, custom.RunSeconds)
	assert.Equal(t, "lanes-v2:latest", custom.Template)

	_, err = agentextension.ParseE2BSettings(map[string]string{
		"runSeconds":      "600",
		"sandboxTemplate": "../../evil template",
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t,
		[]string{"configuration.runSeconds", "configuration.sandboxTemplate"}, fields)
}

func TestDataAnalysisIsItsOwnCapabilityWithoutOutsideContent(t *testing.T) {
	t.Parallel()

	spec, ok := agentextension.SpecFor(agentextension.TypeE2B)
	require.True(t, ok)
	assert.Equal(t, agentextension.CapabilityDataAnalysis, spec.Capability)
	assert.Equal(t, []string{agentextension.ToolAnalyzeData}, spec.Tools())
	assert.Equal(t, agentextension.DefaultDailyAnalyses, spec.DailyLimit(nil, errortypes.NewMultiError()))
	assert.False(t, agentextension.ReturnsExternalContent(agentextension.ToolAnalyzeData),
		"an analysis is marked by the attachments it read, not capped like a web read")
}
