package development

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

// TestOrganizationsSeed creates test organizations
type TestOrganizationsSeed struct {
	seedhelpers.BaseSeed
}

// NewTestOrganizationsSeed creates a new test organizations seed
func NewTestOrganizationsSeed() *TestOrganizationsSeed {
	seed := &TestOrganizationsSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"TestOrganizations",
		"1.0.0",
		"Creates test organizations for development",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(seedhelpers.SeedUSStates)
	return seed
}

func (s *TestOrganizationsSeed) Run(ctx context.Context, tx bun.Tx) error {
	// buCount, err := tx.NewSelect().
	// 	Model((*tenant.BusinessUnit)(nil)).
	// 	Where("name = ?", "Default Business Unit").
	// 	Count(ctx)
	// if err != nil {
	// 	return err
	// }

	// if buCount > 0 {
	// 	return nil
	// }

	// bu := &tenant.BusinessUnit{
	// 	Name: "Default Business Unit",
	// 	Code: "DEFAULT",
	// }

	// if _, err = tx.NewInsert().Model(bu).Exec(ctx); err != nil {
	// 	return fmt.Errorf("create business unit: %w", err)
	// }

	// var caState usstate.UsState
	// err = tx.NewSelect().Model(&caState).Where("abbreviation = ?", "CA").Scan(ctx)
	// if err != nil {
	// 	return fmt.Errorf("get california state: %w", err)
	// }

	// var txState usstate.UsState
	// err = tx.NewSelect().Model(&txState).Where("abbreviation = ?", "TX").Scan(ctx)
	// if err != nil {
	// 	return fmt.Errorf("get texas state: %w", err)
	// }

	// count, err := tx.NewSelect().Model((*tenant.Organization)(nil)).Count(ctx)
	// if err != nil {
	// 	return err
	// }

	// if count > 0 {
	// 	return nil
	// }

	// org1 := &tenant.Organization{
	// 	BusinessUnitID: bu.ID,
	// 	Name:           "Trenova Logistics",
	// 	AddressLine1:   "1 Market Street",
	// 	City:           "Los Angeles",
	// 	ScacCode:       "TRNV",
	// 	PostalCode:     "90001",
	// 	TaxID:          "12-3456789",
	// 	Timezone:       "America/Los_Angeles",
	// 	BucketName:     "trenova-logistics",
	// 	DOTNumber:      "1234567",
	// 	StateID:        caState.ID,
	// 	State:          &caState,
	// }

	// if _, err = tx.NewInsert().Model(org1).Exec(ctx); err != nil {
	// 	return fmt.Errorf("create organization 1: %w", err)
	// }

	// org2 := &tenant.Organization{
	// 	BusinessUnitID: bu.ID,
	// 	Name:           "Swift Transport Co",
	// 	AddressLine1:   "500 Commerce Street",
	// 	City:           "Dallas",
	// 	ScacCode:       "SWFT",
	// 	PostalCode:     "75201",
	// 	TaxID:          "98-7654321",
	// 	Timezone:       "America/Chicago",
	// 	BucketName:     "swift-transport",
	// 	DOTNumber:      "7654321",
	// 	StateID:        txState.ID,
	// 	State:          &txState,
	// }

	// if _, err = tx.NewInsert().Model(org2).Exec(ctx); err != nil {
	// 	return fmt.Errorf("create organization 2: %w", err)
	// }

	return nil
}
