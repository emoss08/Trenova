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
	"log"

	"github.com/casbin/casbin/v2"
	"github.com/uptrace/bun"
)

func loadRoles(_ context.Context, _ *bun.DB, enforcer *casbin.Enforcer) error {
	roles := []string{"Admin", "Dispatcher", "Billing", "Safety", "Maintenance"}

	for _, role := range roles {
		// We don't need to add grouping policies here
		log.Printf("Role defined: %s\n", role)
	}

	return nil
}
