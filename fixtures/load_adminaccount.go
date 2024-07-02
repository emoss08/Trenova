package fixtures

import (
	"context"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/fatih/color"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

func LoadAdminAccount(ctx context.Context, db *bun.DB, org *models.Organization, bu *models.BusinessUnit) error {
	exists, err := db.NewSelect().Model((*models.User)(nil)).Where("username = ?", "admin").Exists(ctx)
	if err != nil {
		return err
	}

	if !exists {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// Find or create the Admin role
		role := new(models.Role)
		err = db.NewSelect().Model(role).Where("name = ?", "Admin").Scan(ctx)
		if err != nil {
			return err
		}

		user := &models.User{
			OrganizationID: org.ID,
			Organization:   org,
			BusinessUnitID: bu.ID,
			BusinessUnit:   bu,
			Status:         "Active",
			Username:       "admin",
			Password:       string(hashedPassword),
			Email:          "admin@trenova.app",
			Name:           "System Administrator",
			IsAdmin:        true,
			Timezone:       "America/New_York",
		}

		_, err = db.NewInsert().Model(user).Exec(ctx)
		if err != nil {
			return err
		}

		// Associate the user with the Admin role
		userRole := &models.UserRole{
			UserID: user.ID,
			RoleID: role.ID,
		}

		_, err = db.NewInsert().Model(userRole).Exec(ctx)
		if err != nil {
			return err
		}

		// Print out the admin account credentials
		color.Yellow("✅ Admin account created successfully")
		color.Yellow("-----------------------------")
		color.Yellow("Admin account credentials:")
		color.Yellow("Email: admin@trenova.app")
		color.Yellow("Password: admin")
		color.Yellow("-----------------------------")

	}

	return nil
}
