package rateimport

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/shopspring/decimal"
)

// ChangeKind is what committing a sheet would do to one lane.
type ChangeKind string

const (
	// ChangeKindRemoved is a lane the contract has that the sheet does not
	// mention. It would stop pricing on commit, which is the most dangerous
	// thing a rate sheet can do quietly.
	ChangeKindRemoved ChangeKind = "Removed"
	// ChangeKindAdded is a lane the sheet introduces.
	ChangeKindAdded ChangeKind = "Added"
	// ChangeKindChanged is a lane both sides carry at different terms.
	ChangeKindChanged ChangeKind = "Changed"
	// ChangeKindDuplicate is two rows in one sheet for the same lane, which
	// would leave the contract with two rules the engine has to break a tie
	// between.
	ChangeKindDuplicate ChangeKind = "Duplicate"
	// ChangeKindUnchanged is a lane the sheet restates exactly.
	ChangeKindUnchanged ChangeKind = "Unchanged"
)

// FieldChange is one term that moved, in the words the diff shows.
//
// "This went up" is not reviewable; "this went from 2.10 to 2.35" is.
type FieldChange struct {
	Field  string `json:"field"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// Change is what committing the sheet would do to one lane.
type Change struct {
	Kind    ChangeKind `json:"kind"`
	LaneKey string     `json:"laneKey"`
	Label   string     `json:"label"`

	// ExistingID names the rule this would supersede, so a commit knows what to
	// close out. Absent on an added lane.
	ExistingID string `json:"existingId,omitempty"`

	Fields []FieldChange `json:"fields,omitempty"`
}

// Diff says what committing this sheet would change.
//
// Lanes are matched by key rather than by position: a sheet is somebody's
// spreadsheet, and the order of its rows carries no meaning.
//
// The result is ordered the way somebody checks it: what disappears first,
// because that is what stops pricing; then what is new; then what moved; then
// what did not, which is most of a monthly sheet and is there for completeness.
func Diff(existing, incoming []*rateagreement.RateAgreementRule) []Change {
	byLane := make(map[string]*rateagreement.RateAgreementRule, len(existing))
	for _, rule := range existing {
		if rule != nil {
			byLane[rule.LaneKey] = rule
		}
	}

	changes := make([]Change, 0, len(existing)+len(incoming))
	seen := make(map[string]struct{}, len(incoming))

	for _, rule := range incoming {
		if rule == nil {
			continue
		}

		if _, repeated := seen[rule.LaneKey]; repeated {
			changes = append(changes, Change{
				Kind:    ChangeKindDuplicate,
				LaneKey: rule.LaneKey,
				Label:   rule.Label,
			})

			continue
		}

		seen[rule.LaneKey] = struct{}{}

		prior, known := byLane[rule.LaneKey]
		if !known {
			changes = append(changes, Change{
				Kind:    ChangeKindAdded,
				LaneKey: rule.LaneKey,
				Label:   rule.Label,
			})

			continue
		}

		changes = append(changes, compare(prior, rule))
	}

	for _, rule := range existing {
		if rule == nil {
			continue
		}

		if _, kept := seen[rule.LaneKey]; kept {
			continue
		}

		changes = append(changes, Change{
			Kind:       ChangeKindRemoved,
			LaneKey:    rule.LaneKey,
			Label:      rule.Label,
			ExistingID: rule.ID.String(),
		})
	}

	sortByRisk(changes)

	return changes
}

// compare works out which of a lane's terms moved.
func compare(prior, incoming *rateagreement.RateAgreementRule) Change {
	change := Change{
		LaneKey:    incoming.LaneKey,
		Label:      incoming.Label,
		ExistingID: prior.ID.String(),
		Fields:     movedFields(prior, incoming),
	}

	if len(change.Fields) == 0 {
		change.Kind = ChangeKindUnchanged
	} else {
		change.Kind = ChangeKindChanged
	}

	return change
}

// movedFields lists every term that differs.
//
// Every one is reported rather than stopping at the first: somebody approving a
// sheet needs all of them, and being shown one at a time is how a review takes
// four rounds.
//
// A term the contract had and the sheet drops counts. A minimum charge silently
// disappearing is exactly what a dry run exists to catch.
func movedFields(prior, incoming *rateagreement.RateAgreementRule) []FieldChange {
	moved := make([]FieldChange, 0, 4)

	compareString(&moved, "ratingBasis", string(prior.RatingBasis), string(incoming.RatingBasis))
	compareString(&moved, "currency", prior.Currency, incoming.Currency)
	// The label is deliberately not compared. A sheet without a label column
	// has one generated for it, and comparing a generated name to a stored one
	// would mark every row on every import as changed.

	compareDecimal(&moved, "rate", prior.Rate, incoming.Rate)
	compareDecimal(&moved, "minCharge", prior.MinCharge, incoming.MinCharge)
	compareDecimal(&moved, "maxCharge", prior.MaxCharge, incoming.MaxCharge)
	compareDecimal(&moved, "discountPercent", prior.DiscountPercent, incoming.DiscountPercent)

	return moved
}

func compareString(moved *[]FieldChange, field, before, after string) {
	if before == after {
		return
	}

	*moved = append(*moved, FieldChange{Field: field, Before: before, After: after})
}

// compareDecimal treats absence and zero as different, because a contract with
// no minimum charge and one with a minimum of zero are different contracts.
func compareDecimal(moved *[]FieldChange, field string, before, after decimal.NullDecimal) {
	if before.Valid == after.Valid {
		if !before.Valid || before.Decimal.Equal(after.Decimal) {
			return
		}
	}

	*moved = append(*moved, FieldChange{
		Field:  field,
		Before: decimalText(before),
		After:  decimalText(after),
	})
}

func decimalText(value decimal.NullDecimal) string {
	if !value.Valid {
		return ""
	}

	return value.Decimal.String()
}

func sortByRisk(changes []Change) {
	slices.SortStableFunc(changes, func(a, b Change) int {
		if by := cmp.Compare(risk(a.Kind), risk(b.Kind)); by != 0 {
			return by
		}

		return cmp.Compare(a.LaneKey, b.LaneKey)
	})
}

func risk(kind ChangeKind) int {
	switch kind {
	case ChangeKindDuplicate:
		return 0
	case ChangeKindRemoved:
		return 1
	case ChangeKindAdded:
		return 2
	case ChangeKindChanged:
		return 3
	case ChangeKindUnchanged:
		return 4
	default:
		return 5
	}
}

// Summary is what somebody reads before deciding whether to read the rest.
type Summary struct {
	Added     int `json:"added"`
	Changed   int `json:"changed"`
	Removed   int `json:"removed"`
	Duplicate int `json:"duplicate"`
	Unchanged int `json:"unchanged"`
}

// ChangesAnything reports whether committing this sheet would do anything.
//
// A sheet that changes nothing is worth saying out loud: it usually means
// somebody uploaded last month's file again.
func (s Summary) ChangesAnything() bool {
	return s.Added > 0 || s.Changed > 0 || s.Removed > 0
}

func Summarize(changes []Change) Summary {
	var summary Summary

	for _, change := range changes {
		switch change.Kind {
		case ChangeKindAdded:
			summary.Added++
		case ChangeKindChanged:
			summary.Changed++
		case ChangeKindRemoved:
			summary.Removed++
		case ChangeKindDuplicate:
			summary.Duplicate++
		case ChangeKindUnchanged:
			summary.Unchanged++
		}
	}

	return summary
}
