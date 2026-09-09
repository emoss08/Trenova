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

	result.Rows = make([]*StagedRow, 0, len(sheet.Rows))
	firstByReference := make(map[string]int, len(sheet.Rows))

	for index, cells := range sheet.Rows {
		parsed, err := ParseRow(cells, result.Mapping, parseOpts)
		if errors.Is(err, ErrBlankRow) {
			continue
		}

		row := &StagedRow{
			RowNumber: sheet.FirstDataRow + index,
			Cells:     cells,
		}
		if err != nil {
			row.Err = err
			result.Rows = append(result.Rows, row)
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

		result.Rows = append(result.Rows, row)
	}

	return result
}
