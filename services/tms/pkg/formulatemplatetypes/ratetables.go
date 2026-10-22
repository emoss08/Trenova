package formulatemplatetypes

import (
	"strings"

	"github.com/emoss08/trenova/pkg/rules"
)

var tableFunctionMarkers = []string{"lookup", "deficitWeight"}

type RateTable struct {
	Name    string
	Axes    int
	Banded  bool
	Encoded []byte
}

type RateTables struct {
	Stamp  string
	Tables []RateTable
}

type TableSelection struct {
	Stamp   string
	Encoded rules.RawJSON
	Shapes  []rules.FormulaTable
}

func (t *RateTables) Select(sources ...string) *TableSelection {
	if t == nil {
		return nil
	}

	selection := &TableSelection{Stamp: t.Stamp, Shapes: make([]rules.FormulaTable, 0, 2)}
	if !ReadsTables(sources...) {
		selection.Encoded = rules.RawJSON("[]")
		return selection
	}

	everything := escaped(sources)
	encoded := make([]byte, 0, 1024)
	encoded = append(encoded, '[')
	for index := range t.Tables {
		table := &t.Tables[index]
		if !everything && !named(sources, table.Name) {
			continue
		}
		if len(selection.Shapes) > 0 {
			encoded = append(encoded, ',')
		}
		encoded = append(encoded, table.Encoded...)
		selection.Shapes = append(selection.Shapes, rules.FormulaTable{
			Name:   table.Name,
			Axes:   table.Axes,
			Banded: table.Banded,
		})
	}
	encoded = append(encoded, ']')
	selection.Encoded = encoded

	return selection
}

func ReadsTables(sources ...string) bool {
	for _, source := range sources {
		for _, marker := range tableFunctionMarkers {
			if strings.Contains(source, marker) {
				return true
			}
		}
	}
	return false
}

func escaped(sources []string) bool {
	for _, source := range sources {
		if strings.ContainsRune(source, '\\') {
			return true
		}
	}
	return false
}

func named(sources []string, name string) bool {
	for _, source := range sources {
		for _, quote := range []string{`"`, `'`, "`"} {
			if strings.Contains(source, quote+name+quote) {
				return true
			}
		}
	}
	return false
}
