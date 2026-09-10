package fuelimport

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/rateimport"
)

type StageOptions struct {
	Provider        fuelpurchase.CardProvider
	Mapping         Mapping
	DefaultFuelType domaintypes.IFTAFuelType
	DefaultCurrency string
	Location        *time.Location
}

type StagedRow struct {
	RowNumber   int
	Cells       []string
	Parsed      *ParsedRow
	Err         error
	DuplicateOf int
}

func (r *StagedRow) Failed() bool { return r.Err != nil }

func (r *StagedRow) IsDuplicate() bool { return r.DuplicateOf > 0 }

type StageResult struct {
	Rows     []*StagedRow
	Mapping  Mapping
	Unmapped []string
	Problems []Problem
}

func (r *StageResult) HasProblems() bool { return len(r.Problems) > 0 }

func Stage(sheet *rateimport.Sheet, opts StageOptions) *StageResult {
	result := &StageResult{}
	if sheet == nil {
		result.Problems = []Problem{{Message: "There is no sheet to stage"}}
		return result
	}

	if len(opts.Mapping) > 0 {
		result.Mapping = opts.Mapping
		result.Unmapped = UnmappedHeaders(sheet.Headers, opts.Mapping)
	} else {
		result.Mapping, result.Unmapped = GuessMapping(sheet.Headers, opts.Provider)
	}

	result.Problems = Validate(result.Mapping, opts.DefaultFuelType.IsValid())
	if len(result.Problems) > 0 {
		return result
	}

	parseOpts := ParseOptions{
		Provider:        opts.Provider,
		DefaultFuelType: opts.DefaultFuelType,
		DefaultCurrency: opts.DefaultCurrency,
		Location:        opts.Location,
	}
	if index, ok := result.Mapping[FieldQuantity]; ok && index < len(sheet.Headers) {
		parseOpts.DefaultUnit = UnitFromHeader(sheet.Headers[index])
	}

	raw := make([]RawRow, 0, len(sheet.Rows))
	for index, cells := range sheet.Rows {
		raw = append(raw, RawRow{RowNumber: sheet.FirstDataRow + index, Cells: cells})
	}
	result.Rows = StageRows(raw, result.Mapping, parseOpts)

	return result
}

// RawRow is one line of a statement and the number it is known by. Rows read
// back from a batch keep the numbers they were first given, so a row somebody
// was told about is still that row after it is staged again.
type RawRow struct {
	RowNumber int
	Cells     []string
}

// StageRows parses rows, fills in a reference for any that arrived without one,
// and marks the ones that repeat a reference already seen in the same batch.
//
// It is separated from [Stage] because a batch can be staged more than once: a
// row held for review is parsed again after somebody fixes what it was waiting
// on, and it has to be read exactly as it was the first time.
func StageRows(raw []RawRow, mapping Mapping, opts ParseOptions) []*StagedRow {
	rows := make([]*StagedRow, 0, len(raw))
	firstByReference := make(map[string]int, len(raw))

	for _, item := range raw {
		parsed, err := ParseRow(item.Cells, mapping, opts)
		if errors.Is(err, ErrBlankRow) {
			continue
		}

		row := &StagedRow{
			RowNumber: item.RowNumber,
			Cells:     item.Cells,
		}
		if err != nil {
			row.Err = err
			rows = append(rows, row)
			continue
		}

		if parsed.Reference == "" {
			parsed.Reference = SyntheticReference(
				opts.Provider,
				parsed.CardLastFour,
				parsed.Purchase.PurchasedAt,
				parsed.JurisdictionKey(),
				parsed.Purchase.Gallons,
				parsed.Purchase.TotalAmountMinor,
			)
			parsed.Purchase.TransactionReference = parsed.Reference
		}
		row.Parsed = parsed

		if first, seen := firstByReference[parsed.Reference]; seen {
			row.DuplicateOf = first
		} else {
			firstByReference[parsed.Reference] = row.RowNumber
		}

		rows = append(rows, row)
	}

	return rows
}
