package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// LookupMatrixSeed creates the single-axis matrices a development formula's
// lookup() calls resolve against. These carry the same codes and numbers the
// old rate table seed created, so a dev expression written before the merge
// keeps producing the same rate after it.
type LookupMatrixSeed struct {
	seedhelpers.BaseSeed
}

func NewLookupMatrixSeed() *LookupMatrixSeed {
	seed := &LookupMatrixSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"LookupMatrix",
		"1.0.0",
		"Creates sample single-axis lookup matrices for development",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(seedhelpers.SeedTestOrganizations, seedhelpers.SeedFormulaTemplate)

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

			// A matrix names the template that says what its cells mean. These
			// matrices exist for lookup() alone, whose reads bypass the
			// template, so they carry the standard Flat Rate template — the
			// same answer the merge migration's backfill gives a matrix with
			// no better signal.
			var flatRate formulatemplate.FormulaTemplate
			if err = tx.NewSelect().
				Model(&flatRate).
				Column("id").
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Where("status = ?", formulatemplate.StatusActive).
				Where("name = ?", "Flat Rate").
				Order("created_at ASC").
				Limit(1).
				Scan(ctx); err != nil {
				return fmt.Errorf("resolve Flat Rate formula template: %w", err)
			}

			defs := []lookupMatrixDef{
				{
					name:        "Fuel Surcharge",
					code:        "fuel_surcharge",
					description: "Fuel surcharge percentage by national average diesel price",
					matchMode:   ratematrix.MatchModeRange,
					cells: []lookupCellDef{
						{rangeMin: "0", rangeMax: "3", value: "0"},
						{rangeMin: "3", rangeMax: "3.5", value: "0.12"},
						{rangeMin: "3.5", rangeMax: "4", value: "0.18"},
						{rangeMin: "4", value: "0.25"},
					},
				},
				{
					name:        "Lane Rates",
					code:        "lane_rate",
					description: "Flat lane rates by origin-destination pair",
					matchMode:   ratematrix.MatchModeExact,
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
					sc:         sc,
					orgID:      org.ID,
					buID:       org.BusinessUnitID,
					templateID: flatRate.ID,
					def:        def,
				}); err != nil {
					return err
				}
			}

			return nil
		},
	)
}

type createLookupMatrixParams struct {
	sc         *seedhelpers.SeedContext
	orgID      pulid.ID
	buID       pulid.ID
	templateID pulid.ID
	def        lookupMatrixDef
}

func (s *LookupMatrixSeed) createMatrix(
	ctx context.Context,
	tx bun.Tx,
	params createLookupMatrixParams,
) error {
	sc, orgID, buID, def := params.sc, params.orgID, params.buID, params.def

	matrix := &ratematrix.RateMatrix{
		ID:                pulid.MustNew("rmx_"),
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		Code:              def.code,
		Name:              def.name,
		Description:       def.description,
		Status:            domaintypes.StatusActive,
		Currency:          "USD",
		FormulaTemplateID: params.templateID,
	}

	if _, err := tx.NewInsert().Model(matrix).Exec(ctx); err != nil {
		return fmt.Errorf("insert lookup matrix %s: %w", def.code, err)
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

	if _, err := tx.NewInsert().Model(dimension).Exec(ctx); err != nil {
		return fmt.Errorf("insert lookup matrix dimension %s: %w", def.code, err)
	}

	cells := make([]*ratematrix.RateMatrixCell, 0, len(def.cells))
	for _, cellDef := range def.cells {
		cells = append(cells, &ratematrix.RateMatrixCell{
			ID:             pulid.MustNew("rmc_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			RateMatrixID:   matrix.ID,
			D0Key:          cellDef.matchKey,
			D0Min:          nullDecimalFromString(cellDef.rangeMin),
			D0Max:          nullDecimalFromString(cellDef.rangeMax),
			Value:          decimal.RequireFromString(cellDef.value),
		})
	}

	if _, err := tx.NewInsert().Model(&cells).Exec(ctx); err != nil {
		return fmt.Errorf("insert lookup matrix cells %s: %w", def.code, err)
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
