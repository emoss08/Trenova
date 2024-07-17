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

	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

func loadBusinessUnit(ctx context.Context, db *bun.DB) (*models.BusinessUnit, error) {
	bu := new(models.BusinessUnit)

	exists, err := db.NewSelect().Model(bu).Where("name = ?", "Trenova Logistics").Exists(ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		bu = &models.BusinessUnit{
			Name:        "Trenova Logistics",
			PhoneNumber: "704-555-1212",
		}

		_, err = db.NewInsert().Model(bu).Exec(ctx)
		if err != nil {
			return nil, err
		}

		return bu, nil
	}

	err = db.NewSelect().Model(bu).Where("name = ?", "Trenova Logistics").Scan(ctx)
	if err != nil {
		return nil, err
	}

	return bu, nil
}
