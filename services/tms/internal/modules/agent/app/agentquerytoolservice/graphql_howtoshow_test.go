package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqllisttool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardhint"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphQLListTool_APageCarriesItsListsHowToShow(t *testing.T) {
	t.Parallel()

	spec := *keyedSpec(t, "worker_list")
	spec.Shows = &boardhint.Show{Records: &boardhint.Records{Words: "these workers"}}
	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkers": workersAnswer(),
	}}
	tool, err := graphqllisttool.New(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{}))
	require.NoError(t, err)

	document := querytooltest.EncodedDocument(t, data)
	assert.Equal(t, boardhint.Line(spec.Shows), document[graphqllisttool.OutcomeHowToShow])
	assert.Contains(t, document[graphqllisttool.OutcomeHowToShow], "these workers")
}

func TestUnpagedList_CarriesItsListsHowToShow(t *testing.T) {
	t.Parallel()

	spec := *keyedSpec(t, "worker_list_checklists")
	spec.Shows = &boardhint.Show{Records: &boardhint.Records{Words: "these checklists"}}
	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkerChecklists":         checklistAnswer(),
		"DeskWorkerChecklistTemplates": templateAnswer(false),
	}}
	tool, err := graphqllisttool.NewUnpaged(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		querytool.ParamWorkerID: pulid.MustNew("wrk_").String(),
	}))
	require.NoError(t, err)

	outcome, ok := data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, boardhint.Line(spec.Shows), outcome[graphqllisttool.OutcomeHowToShow])

	bare, err := graphqllisttool.NewUnpaged(
		keyedSpec(t, "worker_list_checklists"),
		reader,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)
	data, err = bare.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		querytool.ParamWorkerID: pulid.MustNew("wrk_").String(),
	}))
	require.NoError(t, err)
	outcome, _ = data.(map[string]any)
	assert.NotContains(t, outcome, graphqllisttool.OutcomeHowToShow,
		"a list with nothing to show carries no hint")
}
