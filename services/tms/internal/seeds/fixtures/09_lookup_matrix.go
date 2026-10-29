package fixtures

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixcell"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// LookupMatrixSeed creates single-axis tables a development tariff can look
// values up in.
type LookupMatrixSeed struct {
}

var LookupMatrix = Seeds.Add(seed.Seed{
	Name:        "lookup_matrix",
	Kind:        seed.Fixture,
	Description: "Creates sample single-axis rate tables for development",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&LookupMatrixSeed{}).Run,
})

type lookupCellDef struct {
	matchKey string
	rangeMin string
	rangeMax string
	value    string
}

type lookupMatrixDef struct {
	name        string
	code        string
	description string
	matchMode   ratematrix.MatchMode
	valueType   string
	cells       []lookupCellDef
}

func nullDecimalFromString(s string) decimal.NullDecimal {
	if s == "" {
		return decimal.NullDecimal{}
	}

	return decimal.NullDecimal{Decimal: decimal.RequireFromString(s), Valid: true}
}

func (s *LookupMatrixSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	defs := []lookupMatrixDef{
		{
			name:        "Fuel Surcharge",
			code:        "fuel_surcharge",
			description: "Fuel surcharge percentage by national average diesel price",
			matchMode:   ratematrix.MatchModeRange,
			valueType:   ratematrix.ValueTypePercent,
			cells: []lookupCellDef{
				{rangeMin: "0", rangeMax: "3", value: "0"},
				{rangeMin: "3", rangeMax: "3.5", value: "12"},
				{rangeMin: "3.5", rangeMax: "4", value: "18"},
				{rangeMin: "4", value: "25"},
			},
		},
		{
			name:        "Lane Rates",
			code:        "lane_rate",
			description: "Flat lane rates by origin-destination pair",
			matchMode:   ratematrix.MatchModeExact,
			valueType:   ratematrix.ValueTypeMoney,
			cells: []lookupCellDef{
				{matchKey: "ATL-MIA", value: "1450"},
				{matchKey: "ATL-JAX", value: "980"},
			},
		},
	}

	for _, def := range defs {

		if err = s.createMatrix(ctx, db, createLookupMatrixParams{
			orgID: org.ID,
			buID:  org.BusinessUnitID,
			def:   def,
		}); err != nil {
			return err
		}
	}

	return nil
}

type createLookupMatrixParams struct {
	orgID pulid.ID
	buID  pulid.ID
	def   lookupMatrixDef
}

func (s *LookupMatrixSeed) createMatrix(
	ctx context.Context,
	db *keel.DB,
	params createLookupMatrixParams,
) error {
	orgID, buID, def := params.orgID, params.buID, params.def

	matrix := &ratematrix.RateMatrix{
		ID:             pulid.MustNew("rmx_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           def.code,
		Name:           def.name,
		Description:    def.description,
		ValueType:      def.valueType,
		Status:         formulatemplate.StatusDraft,
		DraftEditorIDs: []pulid.ID{},
	}

	kind := ratematrix.DimensionKindCustom
	if def.matchMode == ratematrix.MatchModeRange {
		kind = ratematrix.DimensionKindQuantity
	}

	dimension := &ratematrix.RateMatrixDimension{
		ID:               pulid.MustNew("rmd_"),
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		RateMatrixID:     matrix.ID,
		Position:         0,
		Kind:             kind,
		MatchMode:        def.matchMode,
		Label:            def.name,
		KeyNormalization: ratematrix.KeyNormalizationNone,
		RangeOverflow:    ratematrix.RangeOverflowError,
	}

	cells := make([]*ratematrixcell.RateMatrixCell, 0, len(def.cells))
	for _, cellDef := range def.cells {
		cells = append(cells, &ratematrixcell.RateMatrixCell{
			ID:              pulid.MustNew("rmc_"),
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			RateMatrixID:    matrix.ID,
			D0Key:           cellDef.matchKey,
			D0Min:           nullDecimalFromString(cellDef.rangeMin),
			D0Max:           nullDecimalFromString(cellDef.rangeMax),
			DeficitEligible: true,
			Value:           decimal.RequireFromString(cellDef.value),
		})
	}

	if err := insertPublishedMatrix(ctx, db, &seededMatrix{
		matrix:     matrix,
		dimensions: []*ratematrix.RateMatrixDimension{dimension},
		cells:      cells,
	}); err != nil {
		return err
	}

	return nil
}
