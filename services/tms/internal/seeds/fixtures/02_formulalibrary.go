package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary/formulalibrarydb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/timeutils"
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

var FormulaLibraries = Seeds.Add(seed.Seed{
	Name:        "formula_libraries",
	Kind:        seed.Fixture,
	Description: "The organisation's shared Tariff libraries, approved",
	After:       []*seed.Seed{AdminAccount},
	Run:         runFormulaLibraries,
})

func runFormulaLibraries(ctx context.Context, r *seed.Run) error {
	org, err := defaultOrganization(ctx, r)
	if err != nil {
		return err
	}

	admin, err := adminUser(ctx, r.DB)
	if err != nil {
		return err
	}

	for _, entry := range seededLibraries() {
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
		if err = formulalibrarydb.FormulaLibraries.Create(ctx, r.DB, library); err != nil {
			return fmt.Errorf("insert formula library %s: %w", entry.name, err)
		}

		version := formulalibrary.NewVersionFromLibrary(library, 1, admin.ID, "Seeded library", nil)
		if err = formulalibrarydb.FormulaLibraryVersions.Create(ctx, r.DB, version); err != nil {
			return fmt.Errorf("insert formula library version %s: %w", entry.name, err)
		}
		r.Count(1)
	}

	return nil
}
