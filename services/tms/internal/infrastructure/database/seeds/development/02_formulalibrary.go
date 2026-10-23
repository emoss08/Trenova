package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/buncolgen/formulalibrarycol"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const accessorialsLibraryName = "accessorials"

const accessorialsLibrarySource = `library "org.accessorials"

/// Accessorial charges this organisation bills from its tariffs.

/// The hazmat handling fee: a flat fee plus a surcharge for every mile.
fn hazmat(miles: Distance, fee: Money = $175, perMile: Money/mi = $0.10/mi) -> Money = fee + perMile * miles

/// The residential delivery fee, waived for freight at or above a weight.
fn residential(weight: Weight, fee: Money = $95, waivedFrom: Weight = 10_000 lb) -> Money {
  if weight >= waivedFrom {
    return $0
  }
  return fee
}

test "hazmat adds the per-mile surcharge to the fee" {
  expect hazmat(100 mi) == $185
}

test "residential delivery is waived for heavy freight" {
  expect residential(12_000 lb) == $0
  expect residential(2_000 lb) == $95
}
`

type seededLibrary struct {
	name        string
	description string
	source      string
}

func seededLibraries() []seededLibrary {
	return []seededLibrary{{
		name:        accessorialsLibraryName,
		description: "Hazmat and residential delivery fees the organisation's tariffs import.",
		source:      accessorialsLibrarySource,
	}}
}

type FormulaLibrarySeed struct {
	seedhelpers.BaseSeed
}

func NewFormulaLibrarySeed() *FormulaLibrarySeed {
	seed := &FormulaLibrarySeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"FormulaLibrary",
		"1.0.0",
		"Creates the organisation's shared Tariff libraries, approved",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(seedhelpers.SeedTestOrganizations, seedhelpers.SeedAdminAccount)

	return seed
}

func (s *FormulaLibrarySeed) Run(ctx context.Context, tx bun.Tx) error {
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

			admin, err := sc.GetUserByUsername(ctx, "admin")
			if err != nil {
				return fmt.Errorf("get admin user: %w", err)
			}

			var existing []formulalibrary.FormulaLibrary
			cols := formulalibrarycol.FormulaLibraryColumns
			if err = tx.NewSelect().
				Model(&existing).
				Column(cols.Name.String()).
				Where(cols.OrganizationID.Eq(), org.ID).
				Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
				Scan(ctx); err != nil {
				return fmt.Errorf("load existing formula libraries: %w", err)
			}
			existingNames := make(map[string]struct{}, len(existing))
			for i := range existing {
				existingNames[existing[i].Name] = struct{}{}
			}

			for _, entry := range seededLibraries() {
				if _, ok := existingNames[entry.name]; ok {
					continue
				}

				checked, err := checkSeedSource(
					ctx, rules.TariffKindLibrary, formulalibrary.Path(entry.name), entry.source, nil,
				)
				if err != nil {
					return err
				}

				now := timeutils.NowUnix()
				library := &formulalibrary.FormulaLibrary{
					OrganizationID:       org.ID,
					BusinessUnitID:       org.BusinessUnitID,
					Name:                 entry.name,
					Description:          entry.description,
					Source:               entry.source,
					Imports:              []tarifftypes.Import{},
					TestCount:            checked.testCount,
					Status:               formulatemplate.StatusActive,
					ApprovedByID:         &admin.ID,
					ApprovedAt:           &now,
					CurrentVersionNumber: 1,
				}
				if _, err = tx.NewInsert().Model(library).Exec(ctx); err != nil {
					return fmt.Errorf("insert formula library %s: %w", entry.name, err)
				}

				version := formulalibrary.NewVersionFromLibrary(
					library, 1, admin.ID, "Seeded library", nil,
				)
				if _, err = tx.NewInsert().Model(version).Exec(ctx); err != nil {
					return fmt.Errorf("insert formula library version %s: %w", entry.name, err)
				}

				if err = sc.TrackCreated(ctx, "formula_libraries", library.ID, s.Name()); err != nil {
					return fmt.Errorf("track formula library: %w", err)
				}
				if err = sc.TrackCreated(ctx, "formula_library_versions", version.ID, s.Name()); err != nil {
					return fmt.Errorf("track formula library version: %w", err)
				}
			}

			return nil
		},
	)
}

func (s *FormulaLibrarySeed) Down(ctx context.Context, tx bun.Tx) error {
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

func (s *FormulaLibrarySeed) CanRollback() bool {
	return true
}
