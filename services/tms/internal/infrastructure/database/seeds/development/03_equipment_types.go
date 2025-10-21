package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

// EquipmentTypesSeed Creates equipment types data
type EquipmentTypesSeed struct {
	seedhelpers.BaseSeed
}

// NewEquipmentTypesSeed creates a new equipment_types seed
func NewEquipmentTypesSeed() *EquipmentTypesSeed {
	seed := &EquipmentTypesSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"EquipmentTypes",
		"1.0.0",
		"Creates equipment types data",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	// Development seeds typically depend on base seeds
	seed.SetDependencies("USStates", "AdminAccount", "Permissions")

	return seed
}

// Run executes the seed
func (s *EquipmentTypesSeed) Run(ctx context.Context, db *bun.DB) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			var count int
			err := db.NewSelect().
				Model((*equipmenttype.EquipmentType)(nil)).
				ColumnExpr("count(*)").
				Scan(ctx, &count)
			if err != nil {
				return err
			}

			if count > 0 {
				seedhelpers.LogSuccess("Equipment types already exist, skipping")
				return nil
			}

			// Get default organization and business unit
			defaultOrg, err := seedCtx.GetDefaultOrganization()
			if err != nil {
				return fmt.Errorf("get default organization: %w", err)
			}

			defaultBU, err := seedCtx.GetDefaultBusinessUnit()
			if err != nil {
				return fmt.Errorf("get default business unit: %w", err)
			}

			return seedhelpers.RunInTransaction(
				ctx,
				db,
				s.Name(),
				func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
					equipmentTypes := []equipmenttype.EquipmentType{
						{
							BusinessUnitID: defaultBU.ID,
							OrganizationID: defaultOrg.ID,
							Code:           "TRN",
							Description:    "Tractor",
							Class:          equipmenttype.ClassTractor,
							Color:          "#C51D34",
						},
						{
							BusinessUnitID: defaultBU.ID,
							OrganizationID: defaultOrg.ID,
							Code:           "TRL",
							Description:    "Trailer",
							Class:          equipmenttype.ClassTrailer,
							Color:          "#646B63",
						},
						{
							BusinessUnitID: defaultBU.ID,
							OrganizationID: defaultOrg.ID,
							Code:           "CNT",
							Description:    "Container",
							Class:          equipmenttype.ClassContainer,
							Color:          "#5D9B9B",
						},
						{
							BusinessUnitID: defaultBU.ID,
							OrganizationID: defaultOrg.ID,
							Code:           "OTH",
							Description:    "Other",
							Class:          equipmenttype.ClassOther,
							Color:          "#474A51",
						},
					}

					if _, err := tx.NewInsert().Model(&equipmentTypes).Exec(ctx); err != nil {
						return err
					}

					seedhelpers.LogSuccess("Created equipment_types fixtures",
						"- 4 equipment types created",
					)

					return nil
				},
			)
		},
	)
}
