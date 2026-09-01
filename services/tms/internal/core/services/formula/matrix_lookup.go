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
	rowMode ratematrix.MatchMode
	colMode ratematrix.MatchMode
	byRow   map[string][]twoAxisCell
	cells   []twoAxisCell
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
	exact  map[string]map[string]float64
	ranges map[string][]rateBand
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
		exact:  make(map[string]map[string]float64),
		ranges: make(map[string][]rateBand),
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
			entries[cell.D0Key] = cell.Value.InexactFloat64()
		}
		lookup.exact[item.Matrix.Code] = entries
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
		lookup.ranges[item.Matrix.Code] = bands
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
		rowMode: rowAxis.MatchMode,
		colMode: colAxis.MatchMode,
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
			rowKey: cell.D0Key,
			rowMin: cell.D0Min,
			rowMax: cell.D0Max,
			colKey: cell.D1Key,
			colMin: cell.D1Min,
			colMax: cell.D1Max,
			value:  cell.Value.InexactFloat64(),
		})
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
	if entries, ok := l.exact[table]; ok {
		return lookupExact(table, entries, key)
	}

	if bands, ok := l.ranges[table]; ok {
		return lookupRange(table, bands, key)
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

	candidates, err := t.rowCandidates(table, rowKey)
	if err != nil {
		return 0, err
	}

	for i := range candidates {
		matched, mErr := t.colMatches(table, &candidates[i], colKey)
		if mErr != nil {
			return 0, mErr
		}
		if matched {
			return candidates[i].value, nil
		}
	}

	return 0, fmt.Errorf(
		"%w: rate table %q has no cell matching row %v and column %v",
		formulatemplatetypes.ErrRateTableMiss, table, rowKey, colKey,
	)
}

// rowCandidates narrows a two-axis lookup to the cells whose row axis matches.
// Exact rows resolve through the per-row index; banded rows filter the flat
// cell list half-open on the upper bound, the same way single-axis bands match.
func (t *twoAxisTable) rowCandidates(table string, rowKey any) ([]twoAxisCell, error) {
	if t.rowMode == ratematrix.MatchModeExact {
		matchKey, err := keyToString(rowKey)
		if err != nil {
			return nil, fmt.Errorf("rate table %q row key: %w", table, err)
		}
		return t.byRow[matchKey], nil
	}

	numericKey, err := keyToDecimal(rowKey)
	if err != nil {
		return nil, fmt.Errorf("rate table %q row key: %w", table, err)
	}

	candidates := make([]twoAxisCell, 0, 4)
	for _, cell := range t.cells {
		if bandContains(cell.rowMin, cell.rowMax, numericKey) {
			candidates = append(candidates, cell)
		}
	}

	return candidates, nil
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
		return cell.colKey == matchKey, nil
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

func lookupExact(table string, entries map[string]float64, key any) (float64, error) {
	matchKey, err := keyToString(key)
	if err != nil {
		return 0, fmt.Errorf("rate table %q: %w", table, err)
	}

	value, ok := entries[matchKey]
	if !ok {
		return 0, fmt.Errorf(
			"%w: rate table %q has no entry for key %q",
			formulatemplatetypes.ErrRateTableMiss, table, matchKey,
		)
	}

	return value, nil
}

func lookupRange(table string, bands []rateBand, key any) (float64, error) {
	numericKey, err := keyToDecimal(key)
	if err != nil {
		return 0, fmt.Errorf("rate table %q: %w", table, err)
	}

	if band, ok := findBand(bands, numericKey); ok {
		return band.value, nil
	}

	return 0, fmt.Errorf(
		"%w: rate table %q has no band matching %s",
		formulatemplatetypes.ErrRateTableMiss, table, numericKey.String(),
	)
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
	if entries, ok := l.exact[table]; ok {
		matchKey, err := keyToString(key)
		if err != nil {
			return formulatypes.LookupMatch{}, false
		}
		if _, found := entries[matchKey]; !found {
			return formulatypes.LookupMatch{}, false
		}
		return formulatypes.LookupMatch{MatchedKey: matchKey}, true
	}

	if bands, ok := l.ranges[table]; ok {
		numericKey, err := keyToDecimal(key)
		if err != nil {
			return formulatypes.LookupMatch{}, false
		}
		band, found := findBand(bands, numericKey)
		if !found {
			return formulatypes.LookupMatch{}, false
		}
		return bandMatch(band), true
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

	candidates, err := t.rowCandidates(table, rowKey)
	if err != nil {
		return formulatypes.LookupMatch{}, false
	}

	for i := range candidates {
		matched, mErr := t.colMatches(table, &candidates[i], colKey)
		if mErr != nil {
			return formulatypes.LookupMatch{}, false
		}
		if matched {
			return formulatypes.LookupMatch{MatchedKey: describeCell(&candidates[i])}, true
		}
	}

	return formulatypes.LookupMatch{}, false
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
