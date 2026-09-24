package retrievalservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuseScoresByReciprocalRank(t *testing.T) {
	t.Parallel()

	a, b, c, d := pulid.ID("doc_a"), pulid.ID("doc_b"), pulid.ID("doc_c"), pulid.ID("doc_d")
	fused := Fuse(
		[]repositories.RetrievalKeywordHit{{SourceID: a}, {SourceID: b}, {SourceID: a}},
		[]repositories.VectorSearchHit{
			{SourceID: c, Similarity: 0.9, ChunkIndex: 4},
			{SourceID: b, Similarity: 0.3, ChunkIndex: 1},
			{SourceID: d, Similarity: 0.2},
		},
		DefaultSearchTuning(),
	)
	require.Len(t, fused, 3, "a vector-only hit under the floor is dropped")

	assert.Equal(t, b, fused[0].ID)
	assert.InDelta(t, 1.0/62+1.0/62, fused[0].Score, 1e-12)
	assert.Equal(t, serviceports.RetrievalMatchBoth, fused[0].Match())
	assert.Equal(t, 1, fused[0].ChunkIndex)

	assert.Equal(t, a, fused[1].ID, "a keyword first place beats a vector first place on a tie")
	assert.Equal(t, serviceports.RetrievalMatchWords, fused[1].Match())
	assert.Equal(t, c, fused[2].ID)
	assert.Equal(t, serviceports.RetrievalMatchMeaning, fused[2].Match())
	assert.InDelta(t, 0.9, fused[2].Similarity, 1e-12)
}

func TestFuseWithoutAVectorLegKeepsTheKeywordOrder(t *testing.T) {
	t.Parallel()

	ids := []pulid.ID{"doc_3", "doc_1", "doc_2"}
	keyword := make([]repositories.RetrievalKeywordHit, 0, len(ids))
	for _, id := range ids {
		keyword = append(keyword, repositories.RetrievalKeywordHit{SourceID: id})
	}

	fused := Fuse(keyword, nil, SearchTuning{})
	got := make([]pulid.ID, 0, len(fused))
	for _, hit := range fused {
		got = append(got, hit.ID)
	}
	assert.Equal(t, ids, got)
}

func TestFuseHonoursATunedFloor(t *testing.T) {
	t.Parallel()

	vector := []repositories.VectorSearchHit{{SourceID: "doc_a", Similarity: 0.31}}
	assert.Empty(t, Fuse(nil, vector, SearchTuning{SimilarityFloor: 0.4}))
	assert.Len(t, Fuse(nil, vector, SearchTuning{SimilarityFloor: 0.3}), 1)
}
