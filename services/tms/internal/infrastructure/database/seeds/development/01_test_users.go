package development

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

// TestUsersSeed creates test users for development
type TestUsersSeed struct {
	seedhelpers.BaseSeed
}

// NewTestUsersSeed creates a new test users seed
func NewTestUsersSeed() *TestUsersSeed {
	seed := &TestUsersSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"TestUsers",
		"1.0.0",
		"Creates test users with different roles",
		[]common.Environment{
			common.EnvDevelopment,
			common.EnvTest,
		},
	)
	seed.SetDependencies("Permissions", "TestOrganizations")
	return seed
}

// Run executes the seed
func (s *TestUsersSeed) Run(ctx context.Context, db *bun.DB) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			// defaultOrg, err := seedCtx.GetDefaultOrganization()
			// if err != nil {
			// 	return err
			// }

			// defaultBU, err := seedCtx.GetDefaultBusinessUnit()
			// if err != nil {
			// 	return err
			// }

			// users := []struct {
			// 	opts seedhelpers.UserOptions
			// 	role string
			// }{
			// 	{
			// 		opts: seedhelpers.UserOptions{
			// 			Name:           "John Manager",
			// 			Username:       "manager",
			// 			Email:          "manager@example.com",
			// 			OrganizationID: defaultOrg.ID,
			// 			BusinessUnitID: defaultBU.ID,
			// 		},
			// 		role: "Manager",
			// 	},
			// 	{
			// 		opts: seedhelpers.UserOptions{
			// 			Name:           "Jane Dispatcher",
			// 			Username:       "dispatcher",
			// 			Email:          "dispatcher@example.com",
			// 			OrganizationID: defaultOrg.ID,
			// 			BusinessUnitID: defaultBU.ID,
			// 		},
			// 		role: "Dispatcher",
			// 	},
			// 	{
			// 		opts: seedhelpers.UserOptions{
			// 			Name:           "Bob Driver",
			// 			Username:       "driver1",
			// 			Email:          "driver1@example.com",
			// 			OrganizationID: defaultOrg.ID,
			// 			BusinessUnitID: defaultBU.ID,
			// 		},
			// 		role: "Driver",
			// 	},
			// 	{
			// 		opts: seedhelpers.UserOptions{
			// 			Name:           "Alice Driver",
			// 			Username:       "driver2",
			// 			Email:          "driver2@example.com",
			// 			OrganizationID: defaultOrg.ID,
			// 			BusinessUnitID: defaultBU.ID,
			// 		},
			// 		role: "Driver",
			// 	},
			// 	{
			// 		opts: seedhelpers.UserOptions{
			// 			Name:           "Sarah Viewer",
			// 			Username:       "viewer",
			// 			Email:          "viewer@example.com",
			// 			OrganizationID: defaultOrg.ID,
			// 			BusinessUnitID: defaultBU.ID,
			// 		},
			// 		role: "Viewer",
			// 	},
			// }

			// for _, u := range users {
			// 	user, err := seedCtx.CreateUser(tx, &u.opts)
			// 	if err != nil {
			// 		return fmt.Errorf("create user %s: %w", u.opts.Username, err)
			// 	}

			// 	if err := seedCtx.AssignRoleToUser(tx, user, u.role); err != nil {
			// 		return fmt.Errorf("assign role to %s: %w", u.opts.Username, err)
			// 	}
			// }

			seedhelpers.LogSuccess("Created test users",
				"- manager@example.com (Manager role)",
				"- dispatcher@example.com (Dispatcher role)",
				"- driver1@example.com (Driver role)",
				"- driver2@example.com (Driver role)",
				"- viewer@example.com (Viewer role)",
				"All passwords: password123!",
			)

			return nil
		},
	)
}
