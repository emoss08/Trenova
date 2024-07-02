package fixtures

import (
	"context"
	"log"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

func loadRoles(ctx context.Context, db *bun.DB, org *models.Organization, bu *models.BusinessUnit) error {
	cnt, err := db.NewSelect().Model((*models.Role)(nil)).Count(ctx)
	if err != nil {
		return err
	}

	if cnt == 0 {
		log.Println("Loading roles...")

		roles := []*models.Role{
			{
				Name:           "Admin",
				Description:    "Base role for administrators",
				Organization:   org,
				OrganizationID: org.ID,
				BusinessUnit:   bu,
				BusinessUnitID: bu.ID,
			},
			{
				Name:           "Dispatcher",
				Description:    "Base role for dispatchers",
				Organization:   org,
				OrganizationID: org.ID,
				BusinessUnit:   bu,
				BusinessUnitID: bu.ID,
			},
			{
				Name:           "Billing",
				Description:    "Base role for billing",
				Organization:   org,
				OrganizationID: org.ID,
				BusinessUnit:   bu,
				BusinessUnitID: bu.ID,
			},
			{
				Name:           "Safety",
				Description:    "Base role for safety",
				Organization:   org,
				OrganizationID: org.ID,
				BusinessUnit:   bu,
				BusinessUnitID: bu.ID,
			},
			{
				Name:           "Maintenance",
				Description:    "Base role for maintenance",
				Organization:   org,
				OrganizationID: org.ID,
				BusinessUnit:   bu,
				BusinessUnitID: bu.ID,
			},
		}

		_, err = db.NewInsert().Model(&roles).Exec(ctx)
		if err != nil {
			return err
		}

		// Add all permissions to the admin roles.
		admin := new(models.Role)
		err = db.NewSelect().Model(admin).Where("name = ?", "Admin").Scan(ctx)
		if err != nil {
			return err
		}

		var permissions []*models.Permission

		err = db.NewSelect().Model(&permissions).Scan(ctx)
		if err != nil {
			return err
		}

		for _, permission := range permissions {
			rolePermission := &models.RolePermission{
				RoleID:       admin.ID,
				PermissionID: permission.ID,
			}

			_, err = db.NewInsert().Model(rolePermission).Exec(ctx)
			if err != nil {
				return err
			}
		}

	}

	return nil
}
