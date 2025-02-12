package fixtures

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/fatih/color"
	"github.com/jaswdr/faker/v2"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
)

var ErrAdminAccountAlreadyExists = eris.New("admin account already exists")

func LoadAdminAccount(ctx context.Context, db *bun.DB, fixture *dbfixture.Fixture) (*user.User, error) {
	org := fixture.MustRow("Organization.trenova").(*organization.Organization)
	org2 := fixture.MustRow("Organization.trenova_2").(*organization.Organization)
	bu := fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)

	exists, err := db.NewSelect().Model((*user.User)(nil)).Where("email_address = ?", "admin@trenova.com").Exists(ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		usr := &user.User{
			CurrentOrganizationID: org.ID,
			CurrentOrganization:   org,
			BusinessUnit:          bu,
			BusinessUnitID:        bu.ID,
			Status:                domain.StatusActive,
			Username:              "admin",
			EmailAddress:          "admin@trenova.app",
			Timezone:              "America/New_York",
			Name:                  "System Administrator",
		}

		password, pErr := usr.GeneratePassword("admin")
		if pErr != nil {
			return nil, pErr
		}

		usr.Password = password

		_, err = db.NewInsert().Model(usr).Exec(ctx)
		if err != nil {
			return nil, err
		}

		uo := make([]*user.UserOrganization, 0, 2)

		uo = append(uo, &user.UserOrganization{
			UserID:         usr.ID,
			OrganizationID: org.ID,
		}, &user.UserOrganization{
			UserID:         usr.ID,
			OrganizationID: org2.ID,
		})

		_, err = db.NewInsert().Model(&uo).Exec(ctx)
		if err != nil {
			return nil, err
		}

		// Print out the admin account credentials
		color.Magenta("-----------------------------")
		color.Magenta("Admin account credentials:")
		color.Magenta("Email: admin@trenova.app")
		color.Magenta("Password: admin")
		color.Magenta("-----------------------------")

		return usr, nil
	}

	return nil, ErrAdminAccountAlreadyExists
}

// LoadFakeAccounts generates 50 fake accounts
func LoadFakeAccounts(ctx context.Context, db *bun.DB, fixture *dbfixture.Fixture) error {
	org := fixture.MustRow("Organization.trenova").(*organization.Organization)
	org2 := fixture.MustRow("Organization.trenova_2").(*organization.Organization)
	bu := fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)

	const numAccounts = 50
	users := make([]*user.User, 0, numAccounts)
	uo := make([]*user.UserOrganization, 0, numAccounts)

	fake := faker.New()

	for i := 0; i < numAccounts; i++ {
		email := fake.Internet().Email()
		name := fake.Person().Name()
		username := fake.RandomStringWithLength(19) // This sometimes will generate the same username for multiple users so re-generate if it already exists
		timezone := "America/Los_Angeles"

		usr := &user.User{
			CurrentOrganizationID: org.ID,
			BusinessUnitID:        bu.ID,
			Status:                domain.StatusActive,
			Username:              username, // ensure the user name is no longer than 20 characters
			EmailAddress:          email,
			Timezone:              timezone,
			Name:                  name,
		}

		password, err := usr.GeneratePassword("password123")
		if err != nil {
			return eris.Wrap(err, "failed to generate password")
		}
		usr.Password = password

		users = append(users, usr)

		uo = append(uo, &user.UserOrganization{
			UserID:         usr.ID,
			OrganizationID: org.ID,
		}, &user.UserOrganization{
			UserID:         usr.ID,
			OrganizationID: org2.ID,
		})
	}

	// Bulk insert users
	if _, err := db.NewInsert().Model(&users).Exec(ctx); err != nil {
		return eris.Wrap(err, "failed to bulk insert users")
	}

	// Update user IDs in userOrganizations after user insertion
	for i, user := range users {
		uo[i*2].UserID = user.ID
		uo[i*2+1].UserID = user.ID
	}

	// Bulk insert user organizations
	if _, err := db.NewInsert().Model(&uo).Exec(ctx); err != nil {
		return eris.Wrap(err, "failed to bulk insert user organizations")
	}

	return nil
}
