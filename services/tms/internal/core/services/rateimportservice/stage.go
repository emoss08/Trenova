package rateimportservice

import (
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rateimport"
	pkgrateimport "github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
)

// stagedBatch is a sheet read but not yet applied.
type stagedBatch struct {
	Mapping         pkgrateimport.Mapping
	UnmappedHeaders []string

	Rows       []*rateimport.RateImportRow
	ErrorCount int

	Changes []pkgrateimport.Change
	Summary pkgrateimport.Summary

	// existingByLane is what the contract said before, kept so a commit knows
	// which rules to close out.
	existingByLane map[string]*rateagreement.RateAgreementRule
}

// stage reads a sheet and works out what committing it would do.
//
// Every row is read even after one fails. A sheet that stopped at the first bad
// row would take as many uploads to fix as it has mistakes, which is how bulk
// import ends up slower than typing the lanes in.
//
// A sheet whose columns cannot be placed at all is refused before any row is
// read: staging thousands of rows against a mapping that cannot work fills the
// table with rows nobody will look at.
func stage(
	sheet *pkgrateimport.Sheet,
	existing []*rateagreement.RateAgreementRule,
	templates pkgrateimport.TemplateResolver,
) (*stagedBatch, error) {
	mapping, unmapped := pkgrateimport.GuessMapping(sheet.Headers)

	if err := mapping.Validate(); err != nil {
		return nil, err
	}

	staged := &stagedBatch{
		Mapping:         mapping,
		UnmappedHeaders: unmapped,
		Rows:            make([]*rateimport.RateImportRow, 0, len(sheet.Rows)),
		existingByLane:  make(map[string]*rateagreement.RateAgreementRule, len(existing)),
	}

	for _, rule := range existing {
		if rule != nil {
			staged.existingByLane[rule.LaneKey] = rule
		}
	}

	incoming := make([]*rateagreement.RateAgreementRule, 0, len(sheet.Rows))

	for index, cells := range sheet.Rows {
		rule, err := pkgrateimport.ParseRow(mapping, cells, templates)

		if errors.Is(err, pkgrateimport.ErrBlankRow) {
			// A blank row is padding, not a failure. Staging it would report a
			// clean sheet as full of errors.
			continue
		}

		row := &rateimport.RateImportRow{
			RowNumber: sheet.FirstDataRow + index,
			Cells:     cells,
		}

		if err != nil {
			row.Error = err.Error()
			staged.ErrorCount++
		} else {
			row.Rule = rule
			row.LaneKey = rule.LaneKey
			incoming = append(incoming, rule)
		}

		staged.Rows = append(staged.Rows, row)
	}

	staged.Changes = pkgrateimport.Diff(existing, incoming)

	// A lane only looks withdrawn because no row named it, and a row that would
	// not read named nothing. Withdrawing a lane because one line of somebody's
	// sheet has a typo in it is the worst thing this could do quietly, so while
	// any row is unreadable no lane is reported as removed at all.
	//
	// Fixing the rows and re-uploading is what makes the removals appear, which
	// is the right order to do those in anyway.
	if staged.ErrorCount > 0 {
		staged.Changes = withoutRemovals(staged.Changes)
	}

	staged.Summary = pkgrateimport.Summarize(staged.Changes)

	return staged, nil
}

// withoutRemovals drops the withdrawals from a diff.
func withoutRemovals(changes []pkgrateimport.Change) []pkgrateimport.Change {
	kept := make([]pkgrateimport.Change, 0, len(changes))

	for _, change := range changes {
		if change.Kind == pkgrateimport.ChangeKindRemoved {
			continue
		}

		kept = append(kept, change)
	}

	return kept
}

// plan is what committing a staged batch would do to the agreement.
type plan struct {
	SupersededIDs []pulid.ID
	Rules         []*rateagreement.RateAgreementRule
}

// commitPlan turns a reviewed dry run into the amendment that applies it.
//
// It supersedes exactly what the diff named — the lanes that changed and the
// lanes the sheet dropped — and leaves the ones it restated identically alone.
// Rewriting an unchanged lane would give it a new effective window for no
// reason, and a lane's history is what a dispute reads.
func commitPlan(staged *stagedBatch) plan {
	result := plan{
		SupersededIDs: make([]pulid.ID, 0, len(staged.Changes)),
		Rules:         make([]*rateagreement.RateAgreementRule, 0, len(staged.Rows)),
	}

	supersede := make(map[string]struct{}, len(staged.Changes))
	carry := make(map[string]struct{}, len(staged.Changes))

	for _, change := range staged.Changes {
		switch change.Kind {
		case pkgrateimport.ChangeKindChanged, pkgrateimport.ChangeKindRemoved:
			supersede[change.LaneKey] = struct{}{}
		case pkgrateimport.ChangeKindAdded:
			carry[change.LaneKey] = struct{}{}
		case pkgrateimport.ChangeKindDuplicate, pkgrateimport.ChangeKindUnchanged:
			// A duplicate is dropped: two rules for one lane is a coin flip
			// somebody did not ask for. An unchanged lane is left as it is.
		}
	}

	for laneKey := range supersede {
		if prior, ok := staged.existingByLane[laneKey]; ok {
			result.SupersededIDs = append(result.SupersededIDs, prior.ID)
		}
	}

	seen := make(map[string]struct{}, len(staged.Rows))

	for _, row := range staged.Rows {
		if row.Failed() || row.Rule == nil {
			continue
		}

		if _, repeated := seen[row.LaneKey]; repeated {
			continue
		}

		seen[row.LaneKey] = struct{}{}

		_, changed := supersede[row.LaneKey]
		_, added := carry[row.LaneKey]

		if !changed && !added {
			continue
		}

		result.Rules = append(result.Rules, row.Rule)
	}

	return result
}

var errNothingToCommit = errors.New("this sheet would not change anything in the agreement")
