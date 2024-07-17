// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package fixtures

import (
	"context"
	"fmt"
	"log"

	"github.com/casbin/casbin/v2"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
)

func loadPermissions(ctx context.Context, db *bun.DB, enforcer *casbin.Enforcer) error {
	var resources []*models.Resource
	err := db.NewSelect().Model(&resources).Scan(ctx)
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
		{"create", "Can view all", "Can create, update, and delete"},
		{"update", "Can view all", "Can create, update, and delete"},
		{"delete", "Can view all", "Can create, update, and delete"},
	}

	for _, resource := range resources {
		resourceTypeLower := lo.SnakeCase(resource.Type)
		for _, action := range actions {
			codename := fmt.Sprintf("%s:%s", resourceTypeLower, action.action)

			// Add policy for the Admin role instead of "admin" subject
			_, err = enforcer.AddPolicy("Admin", codename, "allow")
			if err != nil {
				return fmt.Errorf("failed to add policy: %w", err)
			}

			log.Printf("Added permission to Casbin: Admin, %s, allow\n", codename)
		}
	}

	return enforcer.SavePolicy()
}
