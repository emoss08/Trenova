package aianalyticsservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolFailureDetailListsTheProvidersThatServedTheToolBusiestFirst(t *testing.T) {
	t.Parallel()

	got := failureDetail(failureDetailInput{
		request: &aianalyticsports.ToolFailureDetailRequest{
			AnalyticsRequest: aianalyticsports.AnalyticsRequest{
				CompareBy: aianalytics.CompareByModel,
			},
			SeriesKey: llamaKey,
			ToolName:  "assign_driver",
		},
		rows: &aianalyticsports.ToolFailureRows{
			Calls:  140,
			Failed: 12,
			Causes: []aianalyticsports.CauseRow{
				{Cause: aianalytics.CauseToolTimedOut, Calls: 4},
				{Cause: aianalytics.CauseWrongArguments, Calls: 8},
			},
			Providers: []string{groqID.String(), vllmID.String()},
			Examples: []aianalyticsports.ToolFailureExample{
				{
					UnitID:            "unit_1",
					ToolName:          "assign_driver",
					AgentDefinitionID: pulid.ID("agd_dispatch"),
				},
			},
		},
		agents:    map[pulid.ID]string{"agd_dispatch": "Dispatch desk"},
		providers: providerIndex(i18n.EN, providers()),
	})

	require.Len(t, got.Providers, 2)
	assert.Equal(t, "Groq", got.Providers[0].Name)
	assert.Equal(t, "Workstation vLLM", got.Providers[1].Name)
	assert.Equal(
		t,
		aianalytics.CauseWrongArguments,
		got.Causes[0].Cause,
		"the commonest cause leads",
	)
	assert.Equal(t, aianalytics.BlameModel, got.Causes[0].Blame)
	require.Len(t, got.Recent, 1)
	require.NotNil(t, got.Recent[0].Agent)
	assert.Equal(t, "Dispatch desk", got.Recent[0].Agent.Name)
}

func TestToolFailureDetailForAProviderNamesItEvenWithNoToolRows(t *testing.T) {
	t.Parallel()

	got := failureDetail(failureDetailInput{
		request: &aianalyticsports.ToolFailureDetailRequest{
			AnalyticsRequest: aianalyticsports.AnalyticsRequest{
				CompareBy: aianalytics.CompareByProvider,
			},
			SeriesKey: bedrockID.String(),
			ToolName:  "assign_driver",
		},
		rows:      &aianalyticsports.ToolFailureRows{},
		providers: providerIndex(i18n.EN, providers()),
	})

	require.Len(t, got.Providers, 1)
	assert.Equal(t, "AWS Bedrock", got.Providers[0].Name)
}
