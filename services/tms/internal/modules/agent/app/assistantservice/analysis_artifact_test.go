package assistantservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func analysisResult() map[string]any {
	return map[string]any{
		"analysisRunId": "aanr_01",
		"status":        "Succeeded",
		"title":         "Margin by lane for Q3",
		"purpose":       "Margin by lane for Q3",
		"datasets": []any{map[string]any{
			"name": "lanes", "source": "ReportRun", "label": "Margin by lane", "rows": 42.0,
		}},
		"output": "lane  margin\nDAL-ATL 0.18\n",
		"charts": []any{map[string]any{
			"kind": "bar", "title": "Margin by lane",
			"x":      map[string]any{"kind": "text", "values": []any{"DAL-ATL"}},
			"series": []any{map[string]any{"name": "Margin", "values": []any{0.18}}},
		}},
		"calls":  []any{},
		"limits": map[string]any{"calls": 0.0},
		"result": map[string]any{"lanes": 1.0},
		"note":   "What it showed is beside your reply.",
	}
}

func analysisObservation(result map[string]any, code string) agentruntimeports.ToolObservation {
	call := observation(codemode.ToolRunScript, result)
	call.Call.Arguments = map[string]any{"code": code, "purpose": "Margin by lane for Q3"}

	return call
}

func TestArtifactFromObservation_AnAnalysisKeepsItsFiguresChartsAndCode(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(analysisObservation(analysisResult(), "print(1)"))

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindAnalysis, artifact.Kind)
	assert.Equal(t, assistantartifact.StatusReady, artifact.Status)
	assert.Equal(t, "Margin by lane for Q3", artifact.Title)
	assert.Equal(t, "print(1)", artifact.Payload["code"])
	assert.Equal(t, "aanr_01", artifact.Payload["analysisRunId"])
	assert.NotContains(t, artifact.Payload, "note", "the note is for the model")
	assert.NotContains(t, artifact.Payload, "calls", "what the reads cost is for the model")
	assert.NotContains(t, artifact.Payload, "result", "the one result is for the model")
	assert.Equal(t, assistantartifact.FamilyReport,
		assistantartifact.FamilyOf(artifact.Kind, artifact.Payload))
	assert.False(t, assistantartifact.IsLookup(artifact.Kind),
		"an analysis is made on purpose, and kept whether or not the reply names it")
}

// A continuation's call holds only the lines it added; its answer carries the
// whole code that ran, which is what the card shows.
func TestArtifactFromObservation_AContinuationKeepsTheWholeCode(t *testing.T) {
	t.Parallel()

	result := analysisResult()
	result["code"] = "total = 1\n\nshow_figure('Total', total)"

	artifact := artifactFromObservation(
		analysisObservation(result, "show_figure('Total', total)"),
	)

	require.NotNil(t, artifact)
	assert.Equal(t, "total = 1\n\nshow_figure('Total', total)", artifact.Payload["code"])
}

func TestArtifactFromObservation_AnAnalysisWithoutARunIsNotKept(t *testing.T) {
	t.Parallel()

	result := analysisResult()
	delete(result, "analysisRunId")

	assert.Nil(t, artifactFromObservation(analysisObservation(result, "print(1)")))
}

func TestArtifactFromObservation_AnAnalysisFitsItsPayload(t *testing.T) {
	t.Parallel()

	result := analysisResult()
	result["output"] = strings.Repeat("row\n", assistantartifact.MaxPayloadBytes)

	artifact := artifactFromObservation(analysisObservation(result, "print(1)"))
	require.NotNil(t, artifact)
	assert.LessOrEqual(t, payloadSize(artifact.Payload), assistantartifact.MaxPayloadBytes)
	assert.Equal(t, true, artifact.Payload["truncated"])
	assert.Equal(t, "print(1)", artifact.Payload["code"], "the code is kept over what it printed")
}

func TestAnAnalysisMustBeNamedInTheReply(t *testing.T) {
	t.Parallel()

	shown := agentruntimeports.ShownArtifact{Kind: string(assistantartifact.KindAnalysis)}
	assert.True(t, shown.MustBeLinked())
}

func TestArtifactFromObservation_AnAnalysisIsNamedByItsShortTitle(t *testing.T) {
	t.Parallel()

	result := analysisResult()
	result["title"] = "Lane margin"
	result["figures"] = []any{map[string]any{"label": "Average margin", "value": 0.18}}

	artifact := artifactFromObservation(analysisObservation(result, "print(1)"))
	require.NotNil(t, artifact)
	assert.Equal(t, "Lane margin", artifact.Title,
		"the short title names the result, not the sentence of purpose")
	assert.Contains(t, artifact.Payload, "figures")
}

func TestArtifactFromObservation_AnAnalysisThatDidNotRunToTheEndIsNotKept(t *testing.T) {
	t.Parallel()

	for _, status := range []string{"Errored", "Failed", "TimedOut"} {
		result := analysisResult()
		result["status"] = status
		result["error"] = map[string]any{"name": "KeyError", "value": "'Open Balance'"}

		assert.Nil(t, artifactFromObservation(analysisObservation(result, "print(1)")),
			"%s: the agent fixes the code, and the person sees the result it reaches", status)
	}
}

// An agent that loaded a workbook to read its columns before analyzing it
// printed what it saw; that look is its own, so it makes no card the reply has
// to name.
func TestArtifactFromObservation_AnAnalysisThatOnlyPrintedIsNotKept(t *testing.T) {
	t.Parallel()

	result := analysisResult()
	delete(result, "charts")

	assert.Nil(t, artifactFromObservation(analysisObservation(result, "print(df.columns)")))

	result["tables"] = []any{map[string]any{"title": "Margin by lane"}}
	assert.NotNil(t, artifactFromObservation(analysisObservation(result, "show_table(df)")),
		"a table the person was shown is kept")
}
