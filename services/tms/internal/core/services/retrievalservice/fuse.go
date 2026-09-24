package retrievalservice

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/rankfusion"
)

const (
	DefaultRRFK            = rankfusion.DefaultK
	DefaultSimilarityFloor = serviceports.DefaultCatalogSimilarityFloor
	DefaultCandidateFactor = 4
	DefaultMinCandidates   = 20
)

type SearchTuning struct {
	RRFK            int
	SimilarityFloor float64
	CandidateFactor int
	MinCandidates   int
}

func DefaultSearchTuning() SearchTuning {
	return SearchTuning{
		RRFK:            DefaultRRFK,
		SimilarityFloor: DefaultSimilarityFloor,
		CandidateFactor: DefaultCandidateFactor,
		MinCandidates:   DefaultMinCandidates,
	}
}

func (t SearchTuning) normalized() SearchTuning {
	defaults := DefaultSearchTuning()
	if t.RRFK <= 0 {
		t.RRFK = defaults.RRFK
	}
	if t.CandidateFactor <= 0 {
		t.CandidateFactor = defaults.CandidateFactor
	}
	if t.MinCandidates <= 0 {
		t.MinCandidates = defaults.MinCandidates
	}

	return t
}

func (t SearchTuning) candidates(limit int) int {
	return max(limit*t.CandidateFactor, t.MinCandidates)
}

type FusedHit struct {
	ID          pulid.ID
	Score       float64
	KeywordRank int
	VectorRank  int
	Similarity  float64
	ChunkIndex  int
}

func (h FusedHit) Match() serviceports.RetrievalMatch {
	switch {
	case h.KeywordRank > 0 && h.VectorRank > 0:
		return serviceports.RetrievalMatchBoth
	case h.VectorRank > 0:
		return serviceports.RetrievalMatchMeaning
	default:
		return serviceports.RetrievalMatchWords
	}
}

func Fuse(
	keyword []repositories.RetrievalKeywordHit,
	vector []repositories.VectorSearchHit,
	tuning SearchTuning,
) []FusedHit {
	tuning = tuning.normalized()

	hits := make(map[pulid.ID]*FusedHit, len(keyword)+len(vector))
	order := make([]pulid.ID, 0, len(keyword)+len(vector))
	keywordIDs := make([]pulid.ID, 0, len(keyword))
	for _, hit := range keyword {
		if _, seen := hits[hit.SourceID]; seen || hit.SourceID.IsNil() {
			continue
		}
		keywordIDs = append(keywordIDs, hit.SourceID)
		hits[hit.SourceID] = &FusedHit{ID: hit.SourceID, KeywordRank: len(keywordIDs)}
		order = append(order, hit.SourceID)
	}

	vectorIDs := make([]pulid.ID, 0, len(vector))
	for _, hit := range vector {
		fused, seen := hits[hit.SourceID]
		switch {
		case hit.SourceID.IsNil(), seen && fused.VectorRank > 0:
			continue
		case !seen && hit.Similarity < tuning.SimilarityFloor:
			continue
		case !seen:
			fused = &FusedHit{ID: hit.SourceID}
			hits[hit.SourceID] = fused
			order = append(order, hit.SourceID)
		}
		vectorIDs = append(vectorIDs, hit.SourceID)
		fused.VectorRank = len(vectorIDs)
		fused.Similarity = hit.Similarity
		fused.ChunkIndex = hit.ChunkIndex
	}

	scores := rankfusion.Reciprocal(tuning.RRFK, keywordIDs, vectorIDs)
	fused := make([]FusedHit, 0, len(order))
	for _, id := range order {
		hit := *hits[id]
		hit.Score = scores[id]
		fused = append(fused, hit)
	}

	slices.SortStableFunc(fused, compareFused)

	return fused
}

func compareFused(a, b FusedHit) int {
	switch {
	case a.Score > b.Score:
		return -1
	case a.Score < b.Score:
		return 1
	}
	if c := compareRank(a.KeywordRank, b.KeywordRank); c != 0 {
		return c
	}
	if c := compareRank(a.VectorRank, b.VectorRank); c != 0 {
		return c
	}

	return strings.Compare(a.ID.String(), b.ID.String())
}

func compareRank(a, b int) int {
	switch {
	case a == b:
		return 0
	case a == 0:
		return 1
	case b == 0:
		return -1
	case a < b:
		return -1
	default:
		return 1
	}
}
