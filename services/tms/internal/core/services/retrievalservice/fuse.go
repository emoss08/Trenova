package retrievalservice

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DefaultRRFK            = 60
	DefaultSimilarityFloor = 0.0
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
	k := float64(tuning.RRFK)

	hits := make(map[pulid.ID]*FusedHit, len(keyword)+len(vector))
	order := make([]pulid.ID, 0, len(keyword)+len(vector))
	for idx, hit := range keyword {
		if _, seen := hits[hit.SourceID]; seen || hit.SourceID.IsNil() {
			continue
		}
		rank := idx + 1
		hits[hit.SourceID] = &FusedHit{
			ID:          hit.SourceID,
			Score:       1 / (k + float64(rank)),
			KeywordRank: rank,
		}
		order = append(order, hit.SourceID)
	}

	for idx, hit := range vector {
		rank := idx + 1
		fused, seen := hits[hit.SourceID]
		if seen && fused.VectorRank > 0 {
			continue
		}
		if !seen {
			if hit.SourceID.IsNil() || hit.Similarity < tuning.SimilarityFloor {
				continue
			}
			fused = &FusedHit{ID: hit.SourceID}
			hits[hit.SourceID] = fused
			order = append(order, hit.SourceID)
		}
		fused.VectorRank = rank
		fused.Similarity = hit.Similarity
		fused.ChunkIndex = hit.ChunkIndex
		fused.Score += 1 / (k + float64(rank))
	}

	fused := make([]FusedHit, 0, len(order))
	for _, id := range order {
		fused = append(fused, *hits[id])
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
