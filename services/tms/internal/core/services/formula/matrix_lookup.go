package formula

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/shopspring/decimal"
)

type rateBand struct {
	min   decimal.Decimal
	max   decimal.NullDecimal
	value float64
}

// exactTable is a keyed single-axis matrix. Keys are stored already
// normalised by the axis's mode, so a lookup normalises once and compares.
type exactTable struct {
	entries   map[string]float64
	normalize ratematrix.KeyNormalization
}

// rangeTable is a banded single-axis matrix together with the axis's policy
// for a quantity that no band covers.
type rangeTable struct {
	bands    []rateBand
	overflow ratematrix.RangeOverflow
}

// twoAxisCell is one intersection of a two-axis matrix, carrying whichever
// key or bounds each axis's match mode uses.
type twoAxisCell struct {
	rowKey string
	rowMin decimal.NullDecimal
	rowMax decimal.NullDecimal
	colKey string
	colMin decimal.NullDecimal
	colMax decimal.NullDecimal
	value  float64
}

// twoAxisTable indexes a two-axis matrix by however its axes match. When the
// row axis matches exactly the cells are bucketed by row key, so a lookup
// touches only its own row; a range row axis keeps the cells flat and scans,
// which stays cheap because the repository only loads matrices small enough to
// hold whole.
type twoAxisTable struct {
	rowMode      ratematrix.MatchMode
	colMode      ratematrix.MatchMode
	rowNormalize ratematrix.KeyNormalization
	colNormalize ratematrix.KeyNormalization
	rowOverflow  ratematrix.RangeOverflow
	colOverflow  ratematrix.RangeOverflow
	rowBands     []rateBand
	colBands     []rateBand
	byRow        map[string][]twoAxisCell
	cells        []twoAxisCell
}

// matrixLookup answers a formula's lookup() and lookup2() calls from rate
// matrices.
//
// The expression language keeps calling them tables — lookup("fuel_tiers", x)
// reads exactly as it always did — but the storage behind the name is a rate
// matrix. A single axis becomes a key map or sorted bands; two axes become a
// twoAxisTable addressed by lookup2's row and column keys. The value comes
// back as the same raw number a rate table entry held.
type matrixLookup struct {
	exact  map[string]exactTable
	ranges map[string]rangeTable
	two    map[string]*twoAxisTable
}

var (
	_ formulatemplatetypes.RateTableLookup = (*matrixLookup)(nil)
	_ formulatemplatetypes.LookupExplainer = (*matrixLookup)(nil)
)

// NewMatrixLookup builds the lookup provider from every one- and two-axis
// matrix the repository handed over.
//
// A matrix whose one axis matches exactly contributes a key map; one banded by
// range contributes bands sorted by their floor, matched half-open the way the
// matrix's own ContainsQuantity matches. Two-axis matrices index per axis
// mode. Codes are indexed as stored — the repository already restricts to
// active matrices.
func NewMatrixLookup(
	data []*repositories.RateMatrixLookupData,
) formulatemplatetypes.RateTableLookup {
	lookup := &matrixLookup{
		exact:  make(map[string]exactTable),
		ranges: make(map[string]rangeTable),
		two:    make(map[string]*twoAxisTable),
	}

	for _, item := range data {
		if item == nil || item.Matrix == nil {
			continue
		}

		switch len(item.Matrix.Dimensions) {
		case 1:
			indexSingleAxis(lookup, item)
		case 2:
			indexTwoAxis(lookup, item)
		}
	}

	return lookup
}

func indexSingleAxis(lookup *matrixLookup, item *repositories.RateMatrixLookupData) {
	axis := item.Matrix.Dimensions[0]
	if axis == nil {
		return
	}

	switch axis.MatchMode {
	case ratematrix.MatchModeExact:
		entries := make(map[string]float64, len(item.Cells))
		for _, cell := range item.Cells {
			if cell == nil {
				continue
			}
			entries[axis.KeyNormalization.Apply(cell.D0Key)] = cell.Value.InexactFloat64()
		}
		lookup.exact[item.Matrix.Code] = exactTable{
			entries:   entries,
			normalize: axis.KeyNormalization,
		}
	case ratematrix.MatchModeRange:
		bands := make([]rateBand, 0, len(item.Cells))
		for _, cell := range item.Cells {
			if cell == nil || !cell.D0Min.Valid {
				continue
			}
			bands = append(bands, rateBand{
				min:   cell.D0Min.Decimal,
				max:   cell.D0Max,
				value: cell.Value.InexactFloat64(),
			})
		}
		sort.Slice(bands, func(a, b int) bool {
			return bands[a].min.LessThan(bands[b].min)
		})
		lookup.ranges[item.Matrix.Code] = rangeTable{
			bands:    bands,
			overflow: axis.RangeOverflow,
		}
	}
}

func indexTwoAxis(lookup *matrixLookup, item *repositories.RateMatrixLookupData) {
	var rowAxis, colAxis *ratematrix.RateMatrixDimension
	for _, dim := range item.Matrix.Dimensions {
		if dim == nil {
			continue
		}
		switch dim.Position {
		case 0:
			rowAxis = dim
		case 1:
			colAxis = dim
		}
	}
	if rowAxis == nil || colAxis == nil {
		return
	}

	table := &twoAxisTable{
		rowMode:      rowAxis.MatchMode,
		colMode:      colAxis.MatchMode,
		rowNormalize: rowAxis.KeyNormalization,
		colNormalize: colAxis.KeyNormalization,
		rowOverflow:  rowAxis.RangeOverflow,
		colOverflow:  colAxis.RangeOverflow,
	}

	cells := make([]twoAxisCell, 0, len(item.Cells))
	for _, cell := range item.Cells {
		if cell == nil {
			continue
		}
		if table.rowMode == ratematrix.MatchModeRange && !cell.D0Min.Valid {
			continue
		}
		if table.colMode == ratematrix.MatchModeRange && !cell.D1Min.Valid {
			continue
		}
		cells = append(cells, twoAxisCell{
			rowKey: rowAxis.KeyNormalization.Apply(cell.D0Key),
			rowMin: cell.D0Min,
			rowMax: cell.D0Max,
			colKey: colAxis.KeyNormalization.Apply(cell.D1Key),
			colMin: cell.D1Min,
			colMax: cell.D1Max,
			value:  cell.Value.InexactFloat64(),
		})
	}

	if table.rowMode == ratematrix.MatchModeRange {
		table.rowBands = distinctBands(
			cells,
			func(cell twoAxisCell) (decimal.NullDecimal, decimal.NullDecimal) {
				return cell.rowMin, cell.rowMax
			},
		)
	}
	if table.colMode == ratematrix.MatchModeRange {
		table.colBands = distinctBands(
			cells,
			func(cell twoAxisCell) (decimal.NullDecimal, decimal.NullDecimal) {
				return cell.colMin, cell.colMax
			},
		)
	}

	if table.rowMode == ratematrix.MatchModeExact {
		table.byRow = make(map[string][]twoAxisCell)
		for _, cell := range cells {
			table.byRow[cell.rowKey] = append(table.byRow[cell.rowKey], cell)
		}
	} else {
		table.cells = cells
	}

	lookup.two[item.Matrix.Code] = table
}

func (l *matrixLookup) Has(table string) bool {
	if _, ok := l.exact[table]; ok {
		return true
	}
	_, ok := l.ranges[table]
	return ok
}

func (l *matrixLookup) Has2(table string) bool {
	_, ok := l.two[table]
	return ok
}

func (l *matrixLookup) Lookup(table string, key any) (float64, error) {
	if t, ok := l.exact[table]; ok {
		return lookupExact(table, t, key)
	}

	if t, ok := l.ranges[table]; ok {
		band, _, err := resolveBand(table, t, key)
		if err != nil {
			return 0, err
		}
		return band.value, nil
	}

	if _, ok := l.two[table]; ok {
		return 0, fmt.Errorf(
			"rate table %q has two axes — use lookup2(table, rowKey, colKey)",
			table,
		)
	}

	return 0, fmt.Errorf("rate table %q not found", table)
}

func (l *matrixLookup) Lookup2(table string, rowKey, colKey any) (float64, error) {
	t, ok := l.two[table]
	if !ok {
		if l.Has(table) {
			return 0, fmt.Errorf(
				"rate table %q has a single axis — use lookup(table, key)",
				table,
			)
		}
		return 0, fmt.Errorf("two-axis rate table %q not found", table)
	}

	cell, _, err := t.match(table, rowKey, colKey)
	if err != nil {
		return 0, err
	}

	return cell.value, nil
}

// match resolves the one cell a two-axis lookup prices from, reporting
// whether either axis's overflow policy had to move the key into a band.
func (t *twoAxisTable) match(table string, rowKey, colKey any) (*twoAxisCell, bool, error) {
	candidates, rowAdjusted, err := t.rowCandidates(table, rowKey)
	if err != nil {
		return nil, false, err
	}

	cell, err := t.firstColumnMatch(table, candidates, colKey)
	if err != nil {
		return nil, false, err
	}

	colAdjusted := false
	if cell == nil && t.colMode == ratematrix.MatchModeRange && len(candidates) > 0 {
		numericKey, convErr := keyToDecimal(colKey)
		if convErr != nil {
			return nil, false, fmt.Errorf("rate table %q column key: %w", table, convErr)
		}
		if band, ok := overflowBand(t.colBands, numericKey, t.colOverflow); ok {
			cell, err = t.firstColumnMatch(table, candidates, band.min.InexactFloat64())
			if err != nil {
				return nil, false, err
			}
			colAdjusted = cell != nil
		}
	}

	if cell == nil {
		return nil, false, fmt.Errorf(
			"%w: rate table %q has no cell matching row %v and column %v",
			formulatemplatetypes.ErrRateTableMiss, table, rowKey, colKey,
		)
	}

	return cell, rowAdjusted || colAdjusted, nil
}

func (t *twoAxisTable) firstColumnMatch(
	table string,
	candidates []twoAxisCell,
	colKey any,
) (*twoAxisCell, error) {
	for i := range candidates {
		matched, err := t.colMatches(table, &candidates[i], colKey)
		if err != nil {
			return nil, err
		}
		if matched {
			return &candidates[i], nil
		}
	}
	return nil, nil //nolint:nilnil // no match is an outcome, not a failure
}

// rowCandidates narrows a two-axis lookup to the cells whose row axis matches.
// Exact rows resolve through the per-row index; banded rows filter the flat
// cell list half-open on the upper bound, the same way single-axis bands match.
func (t *twoAxisTable) rowCandidates(table string, rowKey any) ([]twoAxisCell, bool, error) {
	if t.rowMode == ratematrix.MatchModeExact {
		matchKey, err := keyToString(rowKey)
		if err != nil {
			return nil, false, fmt.Errorf("rate table %q row key: %w", table, err)
		}
		return t.byRow[t.rowNormalize.Apply(matchKey)], false, nil
	}

	numericKey, err := keyToDecimal(rowKey)
	if err != nil {
		return nil, false, fmt.Errorf("rate table %q row key: %w", table, err)
	}

	candidates := t.rowCellsContaining(numericKey)
	if len(candidates) == 0 {
		if band, ok := overflowBand(t.rowBands, numericKey, t.rowOverflow); ok {
			return t.rowCellsContaining(band.min), true, nil
		}
	}

	return candidates, false, nil
}

func (t *twoAxisTable) rowCellsContaining(quantity decimal.Decimal) []twoAxisCell {
	candidates := make([]twoAxisCell, 0, 4)
	for _, cell := range t.cells {
		if bandContains(cell.rowMin, cell.rowMax, quantity) {
			candidates = append(candidates, cell)
		}
	}
	return candidates
}

func (t *twoAxisTable) colMatches(
	table string,
	cell *twoAxisCell,
	colKey any,
) (bool, error) {
	if t.colMode == ratematrix.MatchModeExact {
		matchKey, err := keyToString(colKey)
		if err != nil {
			return false, fmt.Errorf("rate table %q column key: %w", table, err)
		}
		return cell.colKey == t.colNormalize.Apply(matchKey), nil
	}

	numericKey, err := keyToDecimal(colKey)
	if err != nil {
		return false, fmt.Errorf("rate table %q column key: %w", table, err)
	}

	return bandContains(cell.colMin, cell.colMax, numericKey), nil
}

func bandContains(minimum, maximum decimal.NullDecimal, key decimal.Decimal) bool {
	if !minimum.Valid || key.LessThan(minimum.Decimal) {
		return false
	}
	if maximum.Valid && key.GreaterThanOrEqual(maximum.Decimal) {
		return false
	}
	return true
}

func lookupExact(table string, t exactTable, key any) (float64, error) {
	matchKey, err := keyToString(key)
	if err != nil {
		return 0, fmt.Errorf("rate table %q: %w", table, err)
	}
	matchKey = t.normalize.Apply(matchKey)

	value, ok := t.entries[matchKey]
	if !ok {
		return 0, fmt.Errorf(
			"%w: rate table %q has no entry for key %q",
			formulatemplatetypes.ErrRateTableMiss, table, matchKey,
		)
	}

	return value, nil
}

// resolveBand finds the band that prices a quantity, falling back to the
// axis's overflow policy when none covers it. The bool reports that fallback
// so a receipt can say the key was moved.
func resolveBand(table string, t rangeTable, key any) (rateBand, bool, error) {
	numericKey, err := keyToDecimal(key)
	if err != nil {
		return rateBand{}, false, fmt.Errorf("rate table %q: %w", table, err)
	}

	if band, ok := findBand(t.bands, numericKey); ok {
		return band, false, nil
	}

	if band, ok := overflowBand(t.bands, numericKey, t.overflow); ok {
		return band, true, nil
	}

	return rateBand{}, false, fmt.Errorf(
		"%w: rate table %q has no band matching %s",
		formulatemplatetypes.ErrRateTableMiss, table, numericKey.String(),
	)
}

// overflowBand picks the band an out-of-range quantity prices at, if the
// policy allows one. Bands arrive sorted by floor. A quantity that reached
// here matched no band, so one at or past the top floor is beyond the top
// band's ceiling.
func overflowBand(
	bands []rateBand,
	key decimal.Decimal,
	overflow ratematrix.RangeOverflow,
) (rateBand, bool) {
	if len(bands) == 0 {
		return rateBand{}, false
	}

	bottom := bands[0]
	top := bands[len(bands)-1]

	switch overflow {
	case ratematrix.RangeOverflowClampToTopBand:
		if key.GreaterThanOrEqual(top.min) {
			return top, true
		}
		return rateBand{}, false
	case ratematrix.RangeOverflowNearest:
		if key.LessThan(bottom.min) {
			return bottom, true
		}
		if key.GreaterThanOrEqual(top.min) {
			return top, true
		}
		best, bestDistance := rateBand{}, decimal.Decimal{}
		found := false
		for _, band := range bands {
			distance := distanceToBand(band, key)
			if !found || distance.LessThan(bestDistance) {
				best, bestDistance, found = band, distance, true
			}
		}
		return best, found
	default:
		return rateBand{}, false
	}
}

func distanceToBand(band rateBand, key decimal.Decimal) decimal.Decimal {
	if key.LessThan(band.min) {
		return band.min.Sub(key)
	}
	if band.max.Valid && key.GreaterThanOrEqual(band.max.Decimal) {
		return key.Sub(band.max.Decimal)
	}
	return decimal.Zero
}

// distinctBands lists the bands one axis of a two-axis matrix is cut into,
// so its overflow policy can pick one the same way a single-axis table does.
func distinctBands(
	cells []twoAxisCell,
	bounds func(twoAxisCell) (decimal.NullDecimal, decimal.NullDecimal),
) []rateBand {
	seen := make(map[string]struct{}, len(cells))
	bands := make([]rateBand, 0, len(cells))
	for _, cell := range cells {
		low, high := bounds(cell)
		if !low.Valid {
			continue
		}
		label := bandLabel(low.Decimal, high)
		if _, duplicate := seen[label]; duplicate {
			continue
		}
		seen[label] = struct{}{}
		bands = append(bands, rateBand{min: low.Decimal, max: high})
	}
	sort.Slice(bands, func(a, b int) bool {
		return bands[a].min.LessThan(bands[b].min)
	})
	return bands
}

func keyToString(key any) (string, error) {
	switch v := key.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return "", fmt.Errorf("unsupported lookup key type %T", key)
	}
}

func keyToDecimal(key any) (decimal.Decimal, error) {
	switch v := key.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return decimal.Zero, fmt.Errorf("lookup key %v is not a finite number", v)
		}
		return decimal.NewFromFloat(v), nil
	case float32:
		return keyToDecimal(float64(v))
	case int:
		return decimal.NewFromInt(int64(v)), nil
	case int64:
		return decimal.NewFromInt(v), nil
	case int32:
		return decimal.NewFromInt(int64(v)), nil
	case decimal.Decimal:
		return v, nil
	case string:
		parsed, err := decimal.NewFromString(v)
		if err != nil {
			return decimal.Zero, fmt.Errorf("lookup key %q is not numeric", v)
		}
		return parsed, nil
	default:
		return decimal.Zero, fmt.Errorf("unsupported lookup key type %T", key)
	}
}

// findBand returns the first band containing the key; bands are sorted by
// floor, so the first hit is the lowest band that applies.
func findBand(bands []rateBand, key decimal.Decimal) (rateBand, bool) {
	for _, band := range bands {
		if key.LessThan(band.min) {
			continue
		}
		if band.max.Valid && key.GreaterThanOrEqual(band.max.Decimal) {
			continue
		}
		return band, true
	}
	return rateBand{}, false
}

func bandMatch(band rateBand) formulatypes.LookupMatch {
	low := band.min
	match := formulatypes.LookupMatch{BandMin: &low}
	if band.max.Valid {
		high := band.max.Decimal
		match.BandMax = &high
	}
	return match
}

// ExplainLookup reports which key or band a single-axis lookup would resolve
// to, without evaluating anything.
func (l *matrixLookup) ExplainLookup(table string, key any) (formulatypes.LookupMatch, bool) {
	if t, ok := l.exact[table]; ok {
		matchKey, err := keyToString(key)
		if err != nil {
			return formulatypes.LookupMatch{}, false
		}
		matchKey = t.normalize.Apply(matchKey)
		if _, found := t.entries[matchKey]; !found {
			return formulatypes.LookupMatch{}, false
		}
		return formulatypes.LookupMatch{MatchedKey: matchKey}, true
	}

	if t, ok := l.ranges[table]; ok {
		band, adjusted, err := resolveBand(table, t, key)
		if err != nil {
			return formulatypes.LookupMatch{}, false
		}
		match := bandMatch(band)
		match.Adjusted = adjusted
		return match, true
	}

	return formulatypes.LookupMatch{}, false
}

// ExplainLookup2 reports the cell a two-axis lookup resolves to, described by
// the row and column keys or bounds that matched.
func (l *matrixLookup) ExplainLookup2(
	table string,
	rowKey, colKey any,
) (formulatypes.LookupMatch, bool) {
	t, ok := l.two[table]
	if !ok {
		return formulatypes.LookupMatch{}, false
	}

	cell, adjusted, err := t.match(table, rowKey, colKey)
	if err != nil {
		return formulatypes.LookupMatch{}, false
	}

	return formulatypes.LookupMatch{MatchedKey: describeCell(cell), Adjusted: adjusted}, true
}

func describeCell(cell *twoAxisCell) string {
	row := cell.rowKey
	if row == "" && cell.rowMin.Valid {
		row = bandLabel(cell.rowMin.Decimal, cell.rowMax)
	}
	col := cell.colKey
	if col == "" && cell.colMin.Valid {
		col = bandLabel(cell.colMin.Decimal, cell.colMax)
	}
	return row + " × " + col
}

func bandLabel(low decimal.Decimal, high decimal.NullDecimal) string {
	if high.Valid {
		return low.String() + "–" + high.Decimal.String()
	}
	return low.String() + "+"
}
