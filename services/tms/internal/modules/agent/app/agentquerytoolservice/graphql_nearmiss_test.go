package agentquerytoolservice

import (
	"context"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type conditionalReader struct {
	root  string
	rows  []map[string]any
	holds func(variables string) bool
}

func (r *conditionalReader) Read(
	_ context.Context,
	req *graphqlreadports.GraphQLReadRequest,
) (*graphqlreadports.GraphQLReadResult, error) {
	encoded, err := sonic.MarshalString(req.Variables)
	if err != nil {
		return nil, err
	}
	edges := make([]any, 0, len(r.rows))
	if r.holds(encoded) {
		for _, row := range r.rows {
			edges = append(edges, map[string]any{connectionNode: row})
		}
	}

	return &graphqlreadports.GraphQLReadResult{Data: map[string]any{r.root: map[string]any{
		connectionEdges:    edges,
		connectionPageInfo: map[string]any{pageHasNext: false},
	}}}, nil
}

/*
Asked for Emily Chen's open settlement, Haiku listed hers by Draft, PendingApproval
and Approved, was told only that the status kept them out, and read the list again
without it to learn her one settlement was posted: a seventh call on a six-call
task. The note now names the rows the condition kept out and what each holds.
*/
func TestNearMiss_NamesTheRowsTheConditionKeptOutAndWhatTheyHold(t *testing.T) {
	t.Parallel()

	reader := &conditionalReader{
		root: graphQLReads[toolListDriverSettlements].RootField,
		rows: []map[string]any{{
			"id": "dstl_1", "settlementNumber": "SEED-STL-1005", "status": "Posted",
		}},
		holds: func(variables string) bool {
			return !strings.Contains(variables, "PendingApproval")
		},
	}
	tool, err := newGraphQLListTool(keyedSpec(t, toolListDriverSettlements), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	result, err := tool.Query(t.Context(), dispatchCall(map[string]any{
		"status":   []any{"Draft", "PendingApproval", "Approved"},
		"workerId": "wrk_01M5ZQX9BB3RWCN048WTDPFDKN",
	}))
	require.NoError(t, err)

	note, _ := encodedDocument(t, result)[outcomeNote].(string)
	assert.Contains(t, note, "Without the condition on status, 1 does: "+
		"SEED-STL-1005 (status Posted)")
	assert.Contains(t, note, "That condition alone keeps them out.")
}

/*
Asked to rerun Acme's quarantined files, Haiku searched by sender, by status, by
partner and by status again, each answer saying only that nothing matched those
filters. When the one condition a call made keeps every row out and dropping it
finds none either, the note says there is nothing to find.
*/
func TestNearMiss_ACallWhoseOnlyConditionFindsNothingIsToldThereIsNothing(t *testing.T) {
	t.Parallel()

	reader := &conditionalReader{
		root:  graphQLReads[toolEDIListInboundFiles].RootField,
		holds: func(string) bool { return false },
	}
	tool, err := newGraphQLListTool(keyedSpec(t, toolEDIListInboundFiles), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	result, err := tool.Query(t.Context(), keyedCall(map[string]any{"isaSender": "ACMEMFG"}))
	require.NoError(t, err)

	note, _ := encodedDocument(t, result)[outcomeNote].(string)
	assert.Contains(t, note, "without that condition none do either")
	assert.Contains(t, note, "there are no inbound EDI files here for you to find")

	result, err = tool.Query(t.Context(), keyedCall(map[string]any{
		"isaSender": "ACMEMFG",
		"status":    []any{"Quarantined"},
	}))
	require.NoError(t, err)
	note, _ = encodedDocument(t, result)[outcomeNote].(string)
	assert.NotContains(t, note, "none do either",
		"two conditions that each find nothing alone say nothing of the whole set")
}

func TestKeptOutRows_ReadsAPresentedAmountAndCountsAPartialSample(t *testing.T) {
	t.Parallel()

	page := &graphQLPage{
		items: []map[string]any{
			{"number": "INV-1", "status": "Paid"},
			{"name": "Acme", "status": "Paid"},
			{"status": "Void"},
		},
		hasMore: true,
	}

	assert.Equal(t, "INV-1 (status Paid); Acme (status Paid); status Void",
		keptOutRows(page, "status", "status"))

	tool := &graphQLListTool{spec: graphQLListSpec{entityPlural: "invoices"}}
	assert.Contains(t, tool.keptOut("status", page), "Without the condition on status, "+
		"at least 3 do: ")
}
