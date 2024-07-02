package fixtures

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
)

func loadPermissions(ctx context.Context, db *bun.DB, org *models.Organization, bu *models.BusinessUnit) error {
	var permissions []*models.Permission

	err := db.NewSelect().Model(&permissions).Scan(ctx)
	if err != nil {
		return err
	}

	existingPermissionMap := make(map[string]bool)
	for _, permission := range permissions {
		existingPermissionMap[permission.Codename] = true
	}

	log.Println("Adding base permissions...")

	var resources []*models.Resource
	err = db.NewSelect().Model(&resources).Scan(ctx)
	if err != nil {
		return err
	}

	// Detailed permissions for each action
	actions := []struct {
		action           string
		readDescription  string
		writeDescription string
	}{
		{"view", "Can view all", "Can view all"},
		{"add", "Can view all", "Can add, edit, and delete"},
		{"edit", "Can view all", "Can add, edit, and delete"},
		{"delete", "Can view all", "Can add, edit, and delete"},
	}

	for _, resource := range resources {
		resourceTypeLower := strings.ToLower(resource.Type)
		for _, action := range actions {
			// Format codename, label, and descriptions
			codename := fmt.Sprintf("%s.%s", resourceTypeLower, action.action)
			if existingPermissionMap[codename] {
				continue
			}
			label := fmt.Sprintf("%s %s", utils.ToTitleFormat(action.action), resource.Type)
			readDescription := fmt.Sprintf("%s %s.", action.readDescription, resource.Type)
			writeDescription := fmt.Sprintf("%s %s.", action.writeDescription, resource.Type)

			permission := &models.Permission{
				Codename:         codename,
				Action:           action.action,
				Label:            label,
				ReadDescription:  readDescription,
				WriteDescription: writeDescription,
				ResourceID:       resource.ID,
				Resource:         resource,
			}

			_, err := db.NewInsert().Model(permission).Exec(ctx)
			if err != nil {
				return err
			}

			log.Printf("Added permission: %s\n", permission.Codename)
		}
	}

	return nil
}
