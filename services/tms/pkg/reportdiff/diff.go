// Package reportdiff compares two runs of the same report.
//
// "Did this change?" is the question a person actually has about a report they
// run every week, and it is the one thing a report tool cannot answer by
// looking at a single run. The comparison is over stored rows rather than over
// the rendered artifact, so a currency setting or a column width can never read
// as a change in the numbers.
package reportdiff

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/shopspring/decimal"
)

// ChangeKind is what happened to one row between the two runs.
type ChangeKind string

const (
	// ChangeKindRemoved is a row the earlier run had and the later one does
	// not. A row disappearing is the change most likely to be missed by
	// reading the new report on its own, so it sorts first.
	ChangeKindRemoved ChangeKind = "Removed"
	// ChangeKindAdded is a row only the later run has.
	ChangeKindAdded ChangeKind = "Added"
	// ChangeKindChanged is a row both runs carry with different measures.
	ChangeKindChanged ChangeKind = "Changed"
	// ChangeKindDuplicate is two rows in one run sharing a key, which means
	// the chosen keys do not identify a row and every number attached to that
	// key is unreliable.
	ChangeKindDuplicate ChangeKind = "Duplicate"
	// ChangeKindUnchanged is a row that restates itself exactly.
	ChangeKindUnchanged ChangeKind = "Unchanged"
)

// MeasureChange is one number that moved, in the words the diff shows.
//
// Delta is carried separately from Before and After because "this went up" is
// not reviewable and "this went from 2,840.00 to 3,102.50, up 262.50" is.
type MeasureChange struct {
	Column string `json:"column"`
	Label  string `json:"label"`
	Before string `json:"before"`
	After  string `json:"after"`
	Delta  string `json:"delta"`
}

// Change is what happened to one row.
type Change struct {
	Kind ChangeKind `json:"kind"`
	// Key is the row's identity, as the chosen key columns spell it.
	Key string `json:"key"`
	// KeyValues is the same identity column by column, so a caller can render
	// it as cells rather than re-splitting a joined string.
	KeyValues []string        `json:"keyValues"`
	Measures  []MeasureChange `json:"measures,omitempty"`
	// AbsDelta is the largest absolute measure move on this row, and is what
	// orders the changed rows: the biggest movement is what a person wants to
	// see first.
	AbsDelta decimal.Decimal `json:"-"`
}

// MeasureTotal is one measure summed over each side.
type MeasureTotal struct {
	Column string `json:"column"`
	Label  string `json:"label"`
	Before string `json:"before"`
	After  string `json:"after"`
	Delta  string `json:"delta"`
}

// Summary counts the rows by what happened to them.
type Summary struct {
	Added     int `json:"added"`
	Removed   int `json:"removed"`
	Changed   int `json:"changed"`
	Unchanged int `json:"unchanged"`
	Duplicate int `json:"duplicate"`
}

// Result is the whole comparison.
type Result struct {
	Keys     []string       `json:"keys"`
	Measures []string       `json:"measures"`
	Summary  Summary        `json:"summary"`
	Changes  []Change       `json:"changes"`
	Totals   []MeasureTotal `json:"totals"`
	// Truncated marks that one side hit its row cap, so a row reported
	// removed may simply be past the cap. It is stated rather than silently
	// folded in, because a truncated comparison is still useful and a
	// comparison that hides it is not.
	Truncated bool `json:"truncated"`
}

var (
	ErrSchemaMismatch = errors.New("the two runs do not have the same columns")
	ErrNoKeys         = errors.New("the two runs have no column that identifies a row")
	ErrNoMeasures     = errors.New("the two runs have no numeric column to compare")
	ErrUnknownColumn  = errors.New("no such column in these runs")
)

// Options narrows what is compared. Both lists are column ids; empty means the
// schema decides — dimensions key the rows, numbers are the measures.
type Options struct {
	Keys     []string
	Measures []string
	// IncludeUnchanged keeps every matching row in the result. A weekly report
	// is mostly unchanged rows, so they are dropped by default and the count
	// in Summary carries them.
	IncludeUnchanged bool
	// MaxChanges bounds the rows returned, after sorting, so a comparison of
	// two large reports still fits in an answer. Zero means DefaultMaxChanges.
	MaxChanges int
}

// DefaultMaxChanges is what a person can actually read in one answer. The
// counts in Summary always cover every row, so a bounded list understates the
// rows shown, never the change.
const DefaultMaxChanges = 200

// Compare says what moved between two runs of the same report.
//
// Rows are matched by key rather than by position: two runs order their rows
// however the query's sort left them, and a row that moved from line 40 to
// line 12 has not changed.
func Compare(before, after *reportrows.Envelope, opts Options) (*Result, error) {
	if err := assertSameShape(before, after); err != nil {
		return nil, err
	}

	keys, err := resolveColumns(after, opts.Keys, reportrows.Column.IsDimension)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, ErrNoKeys
	}

	measures, err := resolveColumns(after, opts.Measures, reportrows.Column.IsMeasure)
	if err != nil {
		return nil, err
	}
	if len(measures) == 0 {
		return nil, ErrNoMeasures
	}

	result := &Result{
		Keys:      columnIDs(after.Schema, keys),
		Measures:  columnIDs(after.Schema, measures),
		Truncated: before.Summary.Truncated || after.Summary.Truncated,
	}

	priorRows, priorDuplicates := indexByKey(before.Rows, keys)
	_, currentDuplicates := indexByKey(after.Rows, keys)

	// A key repeated on either side taints that key on both. The run that
	// repeated it never had one row's numbers for it, so comparing its numbers
	// to the other run's would be arithmetic on a figure nobody can stand
	// behind. Such a key is reported once, as a duplicate, and nothing else.
	tainted := union(priorDuplicates, currentDuplicates)

	changes := make([]Change, 0, len(after.Rows)+len(priorRows))
	for key := range tainted {
		changes = append(changes, Change{
			Kind: ChangeKindDuplicate, Key: key, KeyValues: splitKey(key),
		})
	}

	matched := make(map[string]struct{}, len(after.Rows))
	for _, row := range after.Rows {
		key, values := rowKey(row, keys)
		if _, spoiled := tainted[key]; spoiled {
			continue
		}
		matched[key] = struct{}{}

		prior, known := priorRows[key]
		if !known {
			changes = append(changes, Change{
				Kind: ChangeKindAdded, Key: key, KeyValues: values,
				Measures: measuresOf(after.Schema, row, measures),
			})

			continue
		}

		changes = append(changes, compareRow(after.Schema, prior, row, key, values, measures))
	}

	for key, row := range priorRows {
		if _, spoiled := tainted[key]; spoiled {
			continue
		}
		if _, kept := matched[key]; kept {
			continue
		}
		changes = append(changes, Change{
			Kind: ChangeKindRemoved, Key: key, KeyValues: splitKey(key),
			Measures: measuresOf(before.Schema, row, measures),
		})
	}

	result.Summary = summarize(changes)
	result.Totals = totals(before, after, measures)

	if !opts.IncludeUnchanged {
		changes = slices.DeleteFunc(changes, func(c Change) bool {
			return c.Kind == ChangeKindUnchanged
		})
	}
	sortByRisk(changes)

	limit := opts.MaxChanges
	if limit <= 0 {
		limit = DefaultMaxChanges
	}
	if len(changes) > limit {
		changes = changes[:limit]
	}
	result.Changes = changes

	return result, nil
}

// assertSameShape refuses two runs whose columns differ.
//
// Comparing different shapes is not a comparison: a column added between the
// two runs shifts every position after it, and a diff that quietly lined those
// up would report changes in columns nobody touched.
func assertSameShape(before, after *reportrows.Envelope) error {
	if len(before.Schema) != len(after.Schema) {
		return fmt.Errorf(
			"%w: the earlier run has %d columns and the later one has %d",
			ErrSchemaMismatch, len(before.Schema), len(after.Schema),
		)
	}

	for i := range after.Schema {
		if before.Schema[i].ID != after.Schema[i].ID {
			return fmt.Errorf(
				"%w: column %d is %q in the earlier run and %q in the later one",
				ErrSchemaMismatch, i+1, before.Schema[i].ID, after.Schema[i].ID,
			)
		}
		if before.Schema[i].Type != after.Schema[i].Type {
			return fmt.Errorf(
				"%w: column %q is %s in the earlier run and %s in the later one",
				ErrSchemaMismatch, after.Schema[i].ID,
				before.Schema[i].Type, after.Schema[i].Type,
			)
		}
	}

	return nil
}

// resolveColumns turns a caller's column ids into positions, or picks the
// columns that fit when the caller named none.
func resolveColumns(
	envelope *reportrows.Envelope,
	chosen []string,
	fits func(reportrows.Column) bool,
) ([]int, error) {
	if len(chosen) == 0 {
		positions := make([]int, 0, len(envelope.Schema))
		for i := range envelope.Schema {
			if fits(envelope.Schema[i]) {
				positions = append(positions, i)
			}
		}

		return positions, nil
	}

	index := envelope.ColumnIndex()
	positions := make([]int, 0, len(chosen))
	for _, id := range chosen {
		at, ok := index[id]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownColumn, id)
		}
		positions = append(positions, at)
	}

	return positions, nil
}

func columnIDs(schema []reportrows.Column, positions []int) []string {
	ids := make([]string, len(positions))
	for i, at := range positions {
		ids[i] = schema[at].ID
	}

	return ids
}

// keySeparator is a unit separator rather than a printable character, so a key
// column whose value contains a comma or a dash cannot collide with the
// boundary between two columns.
const keySeparator = "\x1f"

func rowKey(row []any, keys []int) (string, []string) {
	values := make([]string, len(keys))
	for i, at := range keys {
		values[i] = cellText(row[at])
	}

	return strings.Join(values, keySeparator), values
}

func splitKey(key string) []string { return strings.Split(key, keySeparator) }

func union(sets ...map[string]struct{}) map[string]struct{} {
	merged := make(map[string]struct{})
	for _, set := range sets {
		for key := range set {
			merged[key] = struct{}{}
		}
	}

	return merged
}

func indexByKey(rows [][]any, keys []int) (map[string][]any, map[string]struct{}) {
	byKey := make(map[string][]any, len(rows))
	duplicates := make(map[string]struct{})

	for _, row := range rows {
		key, _ := rowKey(row, keys)
		if _, repeated := byKey[key]; repeated {
			duplicates[key] = struct{}{}

			continue
		}
		byKey[key] = row
	}

	return byKey, duplicates
}

func compareRow(
	schema []reportrows.Column,
	prior, current []any,
	key string,
	values []string,
	measures []int,
) Change {
	change := Change{Key: key, KeyValues: values, AbsDelta: decimal.Zero}

	for _, at := range measures {
		before := cellDecimal(prior[at])
		after := cellDecimal(current[at])
		if before.Equal(after) {
			continue
		}

		delta := after.Sub(before)
		change.Measures = append(change.Measures, MeasureChange{
			Column: schema[at].ID,
			Label:  schema[at].Label,
			Before: before.String(),
			After:  after.String(),
			Delta:  delta.String(),
		})
		if abs := delta.Abs(); abs.GreaterThan(change.AbsDelta) {
			change.AbsDelta = abs
		}
	}

	if len(change.Measures) == 0 {
		change.Kind = ChangeKindUnchanged
	} else {
		change.Kind = ChangeKindChanged
	}

	return change
}

// measuresOf states an added or removed row's numbers against zero, so the
// delta on a row that appeared reads the same way as one that grew.
func measuresOf(schema []reportrows.Column, row []any, measures []int) []MeasureChange {
	changes := make([]MeasureChange, 0, len(measures))
	for _, at := range measures {
		value := cellDecimal(row[at])
		changes = append(changes, MeasureChange{
			Column: schema[at].ID,
			Label:  schema[at].Label,
			Before: value.String(),
			After:  value.String(),
			Delta:  decimal.Zero.String(),
		})
	}

	return changes
}

// totals sums each measure over every row on each side.
//
// The envelope's own totals row is not used: it is the aggregate over
// everything the filters matched, which on a truncated run is more than the
// rows present — and a total that does not add up from the rows shown beside
// it is worse than no total.
func totals(before, after *reportrows.Envelope, measures []int) []MeasureTotal {
	sums := make([]MeasureTotal, 0, len(measures))
	for _, at := range measures {
		beforeSum := sumColumn(before.Rows, at)
		afterSum := sumColumn(after.Rows, at)
		sums = append(sums, MeasureTotal{
			Column: after.Schema[at].ID,
			Label:  after.Schema[at].Label,
			Before: beforeSum.String(),
			After:  afterSum.String(),
			Delta:  afterSum.Sub(beforeSum).String(),
		})
	}

	return sums
}

func sumColumn(rows [][]any, at int) decimal.Decimal {
	sum := decimal.Zero
	for _, row := range rows {
		sum = sum.Add(cellDecimal(row[at]))
	}

	return sum
}

func summarize(changes []Change) Summary {
	summary := Summary{}
	for i := range changes {
		switch changes[i].Kind {
		case ChangeKindAdded:
			summary.Added++
		case ChangeKindRemoved:
			summary.Removed++
		case ChangeKindChanged:
			summary.Changed++
		case ChangeKindUnchanged:
			summary.Unchanged++
		case ChangeKindDuplicate:
			summary.Duplicate++
		}
	}

	return summary
}

// sortByRisk orders the result the way somebody checks it: what cannot be
// trusted first, then what disappeared, then what is new, then what moved —
// biggest movement first — and lastly what did not move.
func sortByRisk(changes []Change) {
	slices.SortStableFunc(changes, func(a, b Change) int {
		if by := cmp.Compare(risk(a.Kind), risk(b.Kind)); by != 0 {
			return by
		}
		if by := b.AbsDelta.Cmp(a.AbsDelta); by != 0 {
			return by
		}

		return cmp.Compare(a.Key, b.Key)
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

// cellText renders a key cell. JSON writes integers back as float64, so an id
// that went out as 12 must not come back as "12".
func cellText(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}

		return strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprint(v)
	}
}

// cellDecimal reads a measure exactly.
//
// Decimals cross the wire as strings precisely so money never passes through a
// float, and parsing them back through one here would undo that. Counts cross
// as JSON numbers and come back as float64, which is exact for every integer a
// report can hold. A cell that is not a number at all counts as zero: a null
// measure and a measure of zero contribute the same amount to a sum.
func cellDecimal(value any) decimal.Decimal {
	switch v := value.(type) {
	case nil:
		return decimal.Zero
	case string:
		parsed, err := decimal.NewFromString(v)
		if err != nil {
			return decimal.Zero
		}

		return parsed
	case float64:
		return decimal.NewFromFloat(v)
	case int64:
		return decimal.NewFromInt(v)
	case bool:
		return decimal.Zero
	default:
		return decimal.Zero
	}
}
