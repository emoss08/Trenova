package assistantports

import (
	"strconv"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldsOf(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func TestListThreadsRequestAcceptsAShelfRange(t *testing.T) {
	t.Parallel()

	for _, req := range []ListThreadsRequest{
		{},
		{TouchedFrom: 1_000},
		{TouchedBefore: 2_000},
		{TouchedFrom: 1_000, TouchedBefore: 2_000},
	} {
		multiErr := errortypes.NewMultiError()
		req.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), "%+v", req)
	}
}

func TestListThreadsRequestRefusesARangeThatHoldsNothing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		req    ListThreadsRequest
		fields []string
	}{
		{ListThreadsRequest{TouchedFrom: -1}, []string{"touchedFrom"}},
		{ListThreadsRequest{TouchedBefore: -1}, []string{"touchedBefore"}},
		{ListThreadsRequest{TouchedFrom: 2_000, TouchedBefore: 2_000}, []string{"touchedBefore"}},
		{ListThreadsRequest{TouchedFrom: 3_000, TouchedBefore: 2_000}, []string{"touchedBefore"}},
	}
	for _, tc := range cases {
		multiErr := errortypes.NewMultiError()
		tc.req.Validate(multiErr)
		assert.Equal(t, tc.fields, fieldsOf(multiErr), "%+v", tc.req)
	}
}

func TestCountThreadShelvesRequestAcceptsDescendingEdges(t *testing.T) {
	t.Parallel()

	req := CountThreadShelvesRequest{Edges: []int64{4_000, 3_000, 2_000, 1_000}}
	multiErr := errortypes.NewMultiError()
	req.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestCountThreadShelvesRequestRefusesEdgesThatDoNotBandTheList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		edges  []int64
		fields []string
	}{
		{"none", nil, []string{"edge"}},
		{"too many", []int64{9, 8, 7, 6, 5, 4, 3, 2, 1}, []string{"edge"}},
		{"not a time", []int64{2_000, 0, -5}, []string{"edge[1]", "edge[2]"}},
		{"repeated", []int64{2_000, 2_000}, []string{"edge[1]"}},
		{"ascending", []int64{1_000, 2_000, 3_000}, []string{"edge[1]", "edge[2]"}},
	}
	for _, tc := range cases {
		multiErr := errortypes.NewMultiError()
		(&CountThreadShelvesRequest{Edges: tc.edges}).Validate(multiErr)
		assert.Equal(t, tc.fields, fieldsOf(multiErr), tc.name)
	}
}

func TestCountThreadShelvesRequestNamesItsLimit(t *testing.T) {
	t.Parallel()

	edges := make([]int64, MaxThreadShelfEdges+1)
	for index := range edges {
		edges[index] = int64(len(edges) - index)
	}
	multiErr := errortypes.NewMultiError()
	(&CountThreadShelvesRequest{Edges: edges}).Validate(multiErr)

	require.Len(t, multiErr.Errors, 1)
	assert.Equal(
		t,
		"At most "+strconv.Itoa(MaxThreadShelfEdges)+" edges are allowed",
		multiErr.Errors[0].Message,
	)

	multiErr = errortypes.NewMultiError()
	(&CountThreadShelvesRequest{Edges: edges[1:]}).Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}
