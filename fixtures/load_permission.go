// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
