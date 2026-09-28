package capture_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pageSpec struct {
	blank      bool
	patch      string
	cover      bool
	unverified bool
}

func pagesFrom(specs ...pageSpec) ([]*capture.CapturePage, []pulid.ID) {
	pages := make([]*capture.CapturePage, 0, len(specs))
	ids := make([]pulid.ID, 0, len(specs))
	for i, spec := range specs {
		score := 0.2
		if spec.blank {
			score = 0.0001
		}
		page := &capture.CapturePage{
			ID:         pulid.MustNew("cpg_"),
			Sequence:   i + 1,
			BlankScore: &score,
			Markers:    capture.PageMarkers{PatchCode: spec.patch},
		}
		if spec.cover {
			id := pulid.MustNew("ccs_")
			page.Markers.CoverSheetID = &id
		}
		page.Markers.UnrecognizedCoverSheet = spec.unverified
		pages = append(pages, page)
		ids = append(ids, page.ID)
	}

	return pages, ids
}

func TestSplitWithoutSeparatorsIsOneDocument(t *testing.T) {
	t.Parallel()

	pages, ids := pagesFrom(pageSpec{}, pageSpec{}, pageSpec{})
	result := capture.SplitPages(pages, capture.RulesFor(nil))

	require.Len(t, result.Proposals, 1)
	assert.Equal(t, ids, result.Proposals[0].PageIDs)
	assert.Empty(t, result.Separators)
}

func TestSplitOnPatchCodesDropsTheSheet(t *testing.T) {
	t.Parallel()

	pages, ids := pagesFrom(pageSpec{}, pageSpec{patch: "T"}, pageSpec{}, pageSpec{})
	result := capture.SplitPages(pages, capture.RulesFor(nil))

	require.Len(t, result.Proposals, 2)
	assert.Equal(t, ids[:1], result.Proposals[0].PageIDs)
	assert.Equal(t, ids[2:], result.Proposals[1].PageIDs)
	assert.Equal(t, []pulid.ID{ids[1]}, result.Separators)
}

func TestCoverSheetRoutesUntilTheNextDivision(t *testing.T) {
	t.Parallel()

	pages, ids := pagesFrom(
		pageSpec{cover: true}, pageSpec{}, pageSpec{},
		pageSpec{cover: true}, pageSpec{},
		pageSpec{patch: "2"}, pageSpec{},
	)
	result := capture.SplitPages(pages, capture.RulesFor(nil))

	require.Len(t, result.Proposals, 3)
	assert.Equal(t, pages[0].Markers.CoverSheetID, result.Proposals[0].CoverSheetID)
	assert.Equal(t, ids[1:3], result.Proposals[0].PageIDs)
	assert.Equal(t, pages[3].Markers.CoverSheetID, result.Proposals[1].CoverSheetID)
	assert.Nil(t, result.Proposals[2].CoverSheetID, "a patch sheet ends a cover sheet's reach")
}

func TestUnverifiedCoverSheetDividesButRoutesNothing(t *testing.T) {
	t.Parallel()

	pages, _ := pagesFrom(pageSpec{cover: true}, pageSpec{}, pageSpec{unverified: true}, pageSpec{})
	result := capture.SplitPages(pages, capture.RulesFor(nil))

	require.Len(t, result.Proposals, 2)
	assert.NotNil(t, result.Proposals[0].CoverSheetID)
	assert.Nil(t, result.Proposals[1].CoverSheetID)
}

func TestBlankPagesAsSeparatorsAndDiscards(t *testing.T) {
	t.Parallel()

	pages, ids := pagesFrom(pageSpec{}, pageSpec{blank: true}, pageSpec{}, pageSpec{blank: true})

	discard := capture.SplitPages(pages, capture.SplitRules{DiscardBlank: true})
	require.Len(t, discard.Proposals, 1)
	assert.Equal(t, []pulid.ID{ids[0], ids[2]}, discard.Proposals[0].PageIDs)

	separate := capture.SplitPages(pages, capture.SplitRules{BlankPages: true})
	require.Len(t, separate.Proposals, 2)
	assert.Equal(t, []pulid.ID{ids[0]}, separate.Proposals[0].PageIDs)
	assert.Equal(t, []pulid.ID{ids[2]}, separate.Proposals[1].PageIDs)
}

func TestFixedPageCountKeepsTheCoverSheetRoute(t *testing.T) {
	t.Parallel()

	pages, _ := pagesFrom(
		pageSpec{cover: true},
		pageSpec{}, pageSpec{}, pageSpec{}, pageSpec{}, pageSpec{}, pageSpec{},
	)
	result := capture.SplitPages(pages, capture.SplitRules{FixedPageCount: 2})

	require.Len(t, result.Proposals, 3)
	for _, proposal := range result.Proposals {
		assert.Len(t, proposal.PageIDs, 2)
		assert.Equal(t, pages[0].Markers.CoverSheetID, proposal.CoverSheetID)
	}
}

func TestAllBlankStackHasNoDocuments(t *testing.T) {
	t.Parallel()

	pages, _ := pagesFrom(pageSpec{blank: true}, pageSpec{blank: true})
	result := capture.SplitPages(pages, capture.SplitRules{DiscardBlank: true})

	assert.Empty(t, result.Proposals)
	assert.Len(t, result.Separators, 2)
}

func TestRulesForProfile(t *testing.T) {
	t.Parallel()

	profile := &capture.CaptureProfile{
		DiscardBlankPages: true,
		SeparatorStrategies: []capture.SeparatorStrategy{
			capture.SeparatorBlankPage,
			capture.SeparatorFixedPageCount,
		},
		FixedPageCount: 3,
	}

	assert.Equal(t, capture.SplitRules{
		PatchCodes:     true,
		BlankPages:     true,
		DiscardBlank:   true,
		FixedPageCount: 3,
	}, capture.RulesFor(profile))
}
