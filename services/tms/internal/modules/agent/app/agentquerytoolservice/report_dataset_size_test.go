package agentquerytoolservice

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// toolResultBound is the runtime's cut-off for one tool result. A result past
// it reaches the model cut off mid-record.
const toolResultBound = 12000

/*
Every dataset describes within one tool result, and the dataset list pages.

The Report Builder transcript shows both cut off: report_list_datasets sent
all seventy datasets with every edge, and report_get_dataset inlined every
related dataset's field list. The model was left picking fields from a list it
had been told was incomplete.
*/
func TestDescribeReportDataset_EveryDatasetFitsInOneResult(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	for _, entity := range service.kinds.Catalog().Entities() {
		key := entity.Key
		result, err := tools["report_get_dataset"].Query(
			t.Context(),
			testParams(map[string]any{
				"dataset": key,
			}),
		)
		require.NoError(t, err, key)

		encoded, err := sonic.Marshal(result)
		require.NoError(t, err)
		assert.LessOrEqualf(t, len(encoded), toolResultBound,
			"describing %s runs to %d characters", key, len(encoded))
	}
}

func TestListReportDatasets_APageFitsInOneResultAndSaysWhereItContinues(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_list_datasets"].Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(encoded), toolResultBound)

	outcome := result.(searchOutcome)
	require.True(t, outcome.HasMore, "seventy datasets do not fit one page")
	require.NotNil(t, outcome.NextOffset)

	next, err := tools["report_list_datasets"].Query(t.Context(), testParams(map[string]any{
		"offset": float64(*outcome.NextOffset),
	}))
	require.NoError(t, err)
	first := outcome.Items.([]datasetRow)
	second := next.(searchOutcome).Items.([]datasetRow)
	require.NotEmpty(t, second)
	assert.NotEqual(t, first[0].Dataset, second[0].Dataset)
}
