package development

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
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
			common.EnvTest,
		},
	)
	seed.SetDependencies("USStates", "AdminAccount")
	return seed
}

// Run executes the seed
func (s *TestOrganizationsSeed) Run(ctx context.Context, db *bun.DB) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			// Get default business unit
			defaultBU, err := seedCtx.GetDefaultBusinessUnit()
			if err != nil {
				return err
			}

			// Get states for different organizations
			txStateID, err := seedCtx.GetStateByAbbreviation("TX")
			if err != nil {
				return err
			}

			nyStateID, err := seedCtx.GetStateByAbbreviation("NY")
			if err != nil {
				return err
			}

			ilStateID, err := seedCtx.GetStateByAbbreviation("IL")
			if err != nil {
				return err
			}

			// Create test organizations
			orgs := []seedhelpers.OrgOptions{
				{
					Name:           "Texas Transport LLC",
					ScacCode:       "TXTX",
					DOTNumber:      "1234567",
					BusinessUnitID: defaultBU.ID,
					StateID:        txStateID,
					City:           "Austin",
					PostalCode:     "78701",
					AddressLine1:   "123 Congress Ave",
					OrgType:        tenant.TypeCarrier,
					Timezone:       "America/Chicago",
					BucketName:     "texas-transport",
				},
				{
					Name:           "Empire Logistics",
					ScacCode:       "EMPL",
					DOTNumber:      "2345678",
					BusinessUnitID: defaultBU.ID,
					StateID:        nyStateID,
					City:           "New York",
					PostalCode:     "10001",
					AddressLine1:   "456 Broadway",
					OrgType:        tenant.TypeBrokerage,
					Timezone:       "America/New_York",
					BucketName:     "empire-logistics",
				},
				{
					Name:           "Midwest Freight Co",
					ScacCode:       "MWFC",
					DOTNumber:      "3456789",
					BusinessUnitID: defaultBU.ID,
					StateID:        ilStateID,
					City:           "Chicago",
					PostalCode:     "60601",
					AddressLine1:   "789 Michigan Ave",
					OrgType:        tenant.TypeCarrier,
					Timezone:       "America/Chicago",
					BucketName:     "midwest-freight",
				},
			}

			for _, orgOpts := range orgs {
				if _, err := seedCtx.CreateOrganization(tx, &orgOpts); err != nil {
					return err
				}
			}

			seedhelpers.LogSuccess("Created test organizations",
				"- Texas Transport LLC (Carrier)",
				"- Empire Logistics (Broker)",
				"- Midwest Freight Co (Carrier)",
			)

			return nil
		},
	)
}
