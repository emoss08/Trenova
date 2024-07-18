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

	// Basic CRUD actions
	basicActions := []string{"view", "create", "update", "import", "export"}

	// Resources with limited actions
	limitedActionResources := map[string][]string{
		"admin_dashboard": {"view", "export"},
		"billing_client":  {"view", "run_job"},
	}

	// Additional system-defined actions for specific resources
	additionalActions := map[string][]string{
		"shipment":     {"assign_tractor"},
		"tractor":      {"assign_driver"},
		"organization": {"change_logo"},
		"role":         {"view", "update", "assign_permissions"},
	}

	for _, resource := range resources {
		resourceTypeLower := lo.SnakeCase(resource.Type)

		// Check if the resource should have limited actions
		if limitedActions, exists := limitedActionResources[resourceTypeLower]; exists {
			for _, action := range limitedActions {
				if err = addPermissionPolicy(enforcer, resourceTypeLower, action); err != nil {
					return err
				}
			}
		} else {
			// Add basic CRUD permissions
			for _, action := range basicActions {
				if err = addPermissionPolicy(enforcer, resourceTypeLower, action); err != nil {
					return err
				}
			}
		}

		// Add additional system-defined permissions
		if specificActions, exists := additionalActions[resourceTypeLower]; exists {
			for _, action := range specificActions {
				if err = addPermissionPolicy(enforcer, resourceTypeLower, action); err != nil {
					return err
				}
			}
		}
	}

	return enforcer.SavePolicy()
}

func addPermissionPolicy(enforcer *casbin.Enforcer, resource, action string) error {
	codename := fmt.Sprintf("%s:%s", resource, action)
	_, err := enforcer.AddPolicy("Admin", codename, "allow")
	if err != nil {
		return fmt.Errorf("failed to add policy: %w", err)
	}
	log.Printf("Added permission to Casbin: Admin, %s, allow\n", codename)
	return nil
}
