package agentevalgate_test

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type axisEmbedder struct {
	axes     map[string]int
	requests map[serviceports.EmbeddingPurpose]int
}

func (e *axisEmbedder) embed(
	_ context.Context,
	purpose serviceports.EmbeddingPurpose,
	inputs []string,
) ([][]float32, error) {
	e.requests[purpose] += len(inputs)
	vectors := make([][]float32, 0, len(inputs))
	for _, input := range inputs {
		vector := make([]float32, airetrieval.Dimensions768)
		index, ok := e.axes[input]
		if !ok {
			index = airetrieval.Dimensions768 - 1
		}
		vector[index] = 1
		vectors = append(vectors, vector)
	}

	return vectors, nil
}

func syntheticInputs(kit *agentevalgate.Kit) (agentevalgate.EmbeddingInputs, map[string]int) {
	items := kit.Catalog.Items()
	axes := make(map[string]int, len(items)+1)
	for idx, item := range items {
		axes[item.Text] = idx
	}

	target := -1
	for idx, item := range items {
		if item.Key == "list_time_off" {
			target = idx
		}
	}
	axes["glorp the frobnicator"] = target

	return agentevalgate.EmbeddingInputs{
		Documents: append(items, items[0]),
		Queries:   []string{"glorp the frobnicator", "glorp the frobnicator", "zzzz"},
	}, axes
}

func TestRecordEmbeddingFixture_KeysEveryVectorByItsHash(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	inputs, axes := syntheticInputs(kit)
	embedder := &axisEmbedder{axes: axes, requests: map[serviceports.EmbeddingPurpose]int{}}

	fixture, err := agentevalgate.RecordEmbeddingFixture(
		t.Context(), embedder.embed, airetrieval.Dimensions768, inputs,
	)
	require.NoError(t, err)

	assert.Len(t, fixture.Documents, len(inputs.Documents)-1, "a repeated text is embedded once")
	assert.Len(t, fixture.Queries, 2)
	assert.Equal(t,
		len(inputs.Documents)-1,
		embedder.requests[serviceports.EmbeddingPurposeDocument],
	)
	assert.Equal(t, 2, embedder.requests[serviceports.EmbeddingPurposeQuery])
	require.NoError(t, fixture.Stale(inputs))

	found, err := kit.HybridFinder(fixture)("glorp the frobnicator")
	require.NoError(t, err)
	require.NotEmpty(t, found)
	assert.Equal(t, "list_time_off", found[0], "the fixture's vectors drive the meaning side")

	nothing, err := kit.HybridFinder(fixture)("zzzz")
	require.NoError(t, err)
	assert.Empty(t, nothing, "a query close to no tool finds nothing")
}

func TestEmbeddingFixture_NamesTheCommandWhenStale(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	inputs, axes := syntheticInputs(kit)
	embedder := &axisEmbedder{axes: axes, requests: map[serviceports.EmbeddingPurpose]int{}}
	fixture, err := agentevalgate.RecordEmbeddingFixture(
		t.Context(), embedder.embed, airetrieval.Dimensions768, inputs,
	)
	require.NoError(t, err)

	edited := inputs
	edited.Documents = append([]serviceports.EmbeddingCatalogItem(nil), inputs.Documents...)
	edited.Documents[1] = serviceports.NewEmbeddingCatalogItem(
		edited.Documents[1].Key, edited.Documents[1].Text+" Edited.",
	)
	edited.Queries = append(edited.Queries, "a new eval request")

	stale := fixture.Stale(edited)
	require.Error(t, stale)
	assert.Contains(t, stale.Error(), agentevalgate.RecordEmbeddingsCommand)
	assert.Contains(t, stale.Error(), "missing document "+edited.Documents[1].Key)
	assert.Contains(t, stale.Error(), `missing query "a new eval request"`)
	assert.True(t, strings.Contains(stale.Error(), "unused document"),
		"the edited description's old vector is no longer used")

	_, err = kit.HybridFinder(fixture)("a new eval request")
	require.Error(t, err)
	assert.Contains(t, err.Error(), agentevalgate.RecordEmbeddingsCommand)
}

func TestRecordEmbeddingFixture_RefusesAVectorOfTheWrongSize(t *testing.T) {
	t.Parallel()

	short := func(
		context.Context,
		serviceports.EmbeddingPurpose,
		[]string,
	) ([][]float32, error) {
		return [][]float32{{1, 2, 3}}, nil
	}

	_, err := agentevalgate.RecordEmbeddingFixture(t.Context(), short,
		airetrieval.Dimensions768, agentevalgate.EmbeddingInputs{Queries: []string{"x"}})
	require.Error(t, err)
}
