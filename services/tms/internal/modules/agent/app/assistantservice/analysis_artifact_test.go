package assistantservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func analysisResult() map[string]any {
	return map[string]any{
		"analysisRunId": "aanr_01",
		"status":        "Succeeded",
		"purpose":       "Margin by lane for Q3",
		"datasets": []any{map[string]any{
			"name": "lanes", "path": "/home/user/data/lanes.csv", "source": "ReportRun",
			"label": "Margin by lane", "rows": 42.0,
		}},
		"output":  "lane  margin\nDAL-ATL 0.18\n",
		"results": []any{"0.18"},
		"charts":  []any{map[string]any{"position": 1.0, "title": "Margin by lane"}},
		"note":    "The charts are shown to the person beside your reply.",
	}
}

func analysisObservation(result map[string]any, code string) agentports.ToolObservation {
	call := observation("analyze_data", result)
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
	assert.Equal(t, assistantartifact.FamilyReport,
		assistantartifact.FamilyOf(artifact.Kind, artifact.Payload))
	assert.False(t, assistantartifact.IsLookup(artifact.Kind),
		"an analysis is made on purpose, and kept whether or not the reply names it")
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

	shown := agentports.ShownArtifact{Kind: string(assistantartifact.KindAnalysis)}
	assert.True(t, shown.MustBeLinked())
}
