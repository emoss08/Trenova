package capture

import "github.com/emoss08/trenova/shared/pulid"

// SplitRules say where a stack of pages divides into documents.
type SplitRules struct {
	// PatchCodes splits on a patch sheet the scanner reported.
	PatchCodes bool
	// BlankPages splits on a blank sheet used as a separator.
	BlankPages bool
	// DiscardBlank drops blank pages without splitting on them: the back of
	// a one-sided sheet scanned duplex.
	DiscardBlank bool
	// FixedPageCount splits every N pages, zero for never.
	FixedPageCount int
}

// RulesFor are the rules a profile asks for. Cover sheets are not a rule: a
// person who printed one and put it in the stack meant it, whatever profile
// the scan used. A batch with no profile splits on patch codes only, because
// the device reports one only when the scanner found one.
func RulesFor(profile *CaptureProfile) SplitRules {
	if profile == nil {
		return SplitRules{PatchCodes: true}
	}

	rules := SplitRules{
		PatchCodes:   true,
		BlankPages:   profile.UsesSeparator(SeparatorBlankPage),
		DiscardBlank: profile.DiscardBlankPages,
	}
	if profile.UsesSeparator(SeparatorFixedPageCount) {
		rules.FixedPageCount = profile.FixedPageCount
	}

	return rules
}

// Proposal is one document a stack divides into.
type Proposal struct {
	PageIDs []pulid.ID
	// CoverSheetID is the verified cover sheet in front of these pages, which
	// says where they go.
	CoverSheetID *pulid.ID
}

// SplitResult is how a stack divided: the documents, and the pages that are
// not part of any because they only marked a division.
type SplitResult struct {
	Proposals  []Proposal
	Separators []pulid.ID
}

// SplitPages divides pages, already in sequence order, into documents.
//
// A cover sheet always divides and routes what follows it until the next
// division. A sheet that looked like a cover sheet but did not verify still
// divides, and routes nothing: splitting where a person clearly meant a split
// is right even when the destination cannot be trusted.
func SplitPages(pages []*CapturePage, rules SplitRules) SplitResult {
	result := SplitResult{
		Proposals:  make([]Proposal, 0, 1),
		Separators: make([]pulid.ID, 0),
	}

	var current Proposal
	// flush closes the current document. A division by page count keeps the
	// cover sheet's route; any other division ends it.
	flush := func(keepRoute bool) {
		if len(current.PageIDs) > 0 {
			result.Proposals = append(result.Proposals, current)
		}
		next := Proposal{}
		if keepRoute {
			next.CoverSheetID = current.CoverSheetID
		}
		current = next
	}

	for _, page := range pages {
		coverSheet := page.Markers.CoverSheetID != nil || page.Markers.UnrecognizedCoverSheet
		patch := rules.PatchCodes && page.Markers.PatchCode != ""
		blankSeparator := rules.BlankPages && page.IsBlank()

		if coverSheet || patch || blankSeparator {
			result.Separators = append(result.Separators, page.ID)
			flush(false)
			if coverSheet {
				current.CoverSheetID = page.Markers.CoverSheetID
			}

			continue
		}

		if rules.DiscardBlank && page.IsBlank() {
			result.Separators = append(result.Separators, page.ID)

			continue
		}

		current.PageIDs = append(current.PageIDs, page.ID)
		if rules.FixedPageCount > 0 && len(current.PageIDs) >= rules.FixedPageCount {
			flush(true)
		}
	}
	flush(false)

	return result
}
