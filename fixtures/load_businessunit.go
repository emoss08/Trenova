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
