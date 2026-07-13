package formula

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/ratetable"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/shopspring/decimal"
)

type rateBand struct {
	min   decimal.Decimal
	max   decimal.NullDecimal
	value float64
}

type rateTableLookup struct {
	exact  map[string]map[string]float64
	ranges map[string][]rateBand
}

var _ formulatemplatetypes.RateTableLookup = (*rateTableLookup)(nil)

func NewRateTableLookup(tables []*ratetable.RateTable) formulatemplatetypes.RateTableLookup {
	lookup := &rateTableLookup{
		exact:  make(map[string]map[string]float64),
		ranges: make(map[string][]rateBand),
	}

	for _, table := range tables {
		if table == nil || !table.Active {
			continue
		}

		switch table.LookupType {
		case ratetable.LookupTypeExact:
			entries := make(map[string]float64, len(table.Entries))
			for _, entry := range table.Entries {
				if entry == nil || entry.MatchKey == nil {
					continue
				}
				entries[*entry.MatchKey] = entry.Value.InexactFloat64()
			}
			lookup.exact[table.Key] = entries
		case ratetable.LookupTypeRange:
			bands := make([]rateBand, 0, len(table.Entries))
			for _, entry := range table.Entries {
				if entry == nil || !entry.RangeMin.Valid {
					continue
				}
				bands = append(bands, rateBand{
					min:   entry.RangeMin.Decimal,
					max:   entry.RangeMax,
					value: entry.Value.InexactFloat64(),
				})
			}
			sort.Slice(bands, func(a, b int) bool {
				return bands[a].min.LessThan(bands[b].min)
			})
			lookup.ranges[table.Key] = bands
		}
	}

	return lookup
}

func (l *rateTableLookup) Has(table string) bool {
	if _, ok := l.exact[table]; ok {
		return true
	}
	_, ok := l.ranges[table]
	return ok
}

func (l *rateTableLookup) Lookup(table string, key any) (float64, error) {
	if entries, ok := l.exact[table]; ok {
		return lookupExact(table, entries, key)
	}

	if bands, ok := l.ranges[table]; ok {
		return lookupRange(table, bands, key)
	}

	return 0, fmt.Errorf("rate table %q not found", table)
}

func lookupExact(table string, entries map[string]float64, key any) (float64, error) {
	matchKey, err := keyToString(key)
	if err != nil {
		return 0, fmt.Errorf("rate table %q: %w", table, err)
	}

	value, ok := entries[matchKey]
	if !ok {
		return 0, fmt.Errorf("rate table %q has no entry for key %q", table, matchKey)
	}

	return value, nil
}

func lookupRange(table string, bands []rateBand, key any) (float64, error) {
	numericKey, err := keyToDecimal(key)
	if err != nil {
		return 0, fmt.Errorf("rate table %q: %w", table, err)
	}

	for _, band := range bands {
		if numericKey.LessThan(band.min) {
			continue
		}
		if band.max.Valid && numericKey.GreaterThanOrEqual(band.max.Decimal) {
			continue
		}
		return band.value, nil
	}

	return 0, fmt.Errorf("rate table %q has no band matching %s", table, numericKey.String())
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
