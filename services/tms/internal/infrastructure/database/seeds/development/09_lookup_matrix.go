package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// LookupMatrixSeed creates single-axis tables a development tariff can look
// values up in.
type LookupMatrixSeed struct {
	seedhelpers.BaseSeed
}

func NewLookupMatrixSeed() *LookupMatrixSeed {
	seed := &LookupMatrixSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"LookupMatrix",
		"2.0.0",
		"Creates sample single-axis rate tables for development",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(seedhelpers.SeedTestOrganizations)

	return seed
}

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

func (s *LookupMatrixSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetOrganization("default_org")
			if err != nil {
				org, err = sc.GetDefaultOrganization(ctx)
				if err != nil {
					return fmt.Errorf("get organization: %w", err)
				}
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
				exists, cErr := tx.NewSelect().
					Model((*ratematrix.RateMatrix)(nil)).
					Where("organization_id = ?", org.ID).
					Where("business_unit_id = ?", org.BusinessUnitID).
					Where("lower(code) = lower(?)", def.code).
					Exists(ctx)
				if cErr != nil {
					return fmt.Errorf("check existing lookup matrix %s: %w", def.code, cErr)
				}
				if exists {
					continue
				}

				if err = s.createMatrix(ctx, tx, createLookupMatrixParams{
					sc:    sc,
					orgID: org.ID,
					buID:  org.BusinessUnitID,
					def:   def,
				}); err != nil {
					return err
				}
			}

			return nil
		},
	)
}

type createLookupMatrixParams struct {
	sc    *seedhelpers.SeedContext
	orgID pulid.ID
	buID  pulid.ID
	def   lookupMatrixDef
}

func (s *LookupMatrixSeed) createMatrix(
	ctx context.Context,
	tx bun.Tx,
	params createLookupMatrixParams,
) error {
	sc, orgID, buID, def := params.sc, params.orgID, params.buID, params.def

	matrix := &ratematrix.RateMatrix{
		ID:             pulid.MustNew("rmx_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           def.code,
		Name:           def.name,
		Description:    def.description,
		ValueType:      def.valueType,
	}

	kind := ratematrix.DimensionKindCustom
	if def.matchMode == ratematrix.MatchModeRange {
		kind = ratematrix.DimensionKindQuantity
	}

	dimension := &ratematrix.RateMatrixDimension{
		ID:             pulid.MustNew("rmd_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		RateMatrixID:   matrix.ID,
		Position:       0,
		Kind:           kind,
		MatchMode:      def.matchMode,
		Label:          def.name,
	}

	cells := make([]*ratematrix.RateMatrixCell, 0, len(def.cells))
	for _, cellDef := range def.cells {
		cells = append(cells, &ratematrix.RateMatrixCell{
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

	if err := insertPublishedMatrix(ctx, tx, &seededMatrix{
		matrix:     matrix,
		dimensions: []*ratematrix.RateMatrixDimension{dimension},
		cells:      cells,
	}); err != nil {
		return err
	}

	if err := sc.TrackCreated(ctx, "rate_matrices", matrix.ID, s.Name()); err != nil {
		return fmt.Errorf("track lookup matrix: %w", err)
	}

	return nil
}

func (s *LookupMatrixSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *LookupMatrixSeed) CanRollback() bool {
	return true
}
