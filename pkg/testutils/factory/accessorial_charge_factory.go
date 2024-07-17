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

package factory

import (
	"context"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
)

type AccessorialChargeFactory struct {
	db *bun.DB
}

func NewAccessorialChargeFactory(db *bun.DB) *AccessorialChargeFactory {
	return &AccessorialChargeFactory{db: db}
}

func (o *AccessorialChargeFactory) MustCreateAccessorialCharge(ctx context.Context, orgID, buID uuid.UUID) (*models.AccessorialCharge, error) {
	// Generate the random string
	randomString := lo.RandomString(10, lo.LettersCharset)

	accessorialCharge := &models.AccessorialCharge{
		Status:         property.StatusActive,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Method:         "Distance",
		Code:           randomString,
		Description:    "Test Accessorial Charge",
	}

	if _, err := o.db.NewInsert().Model(accessorialCharge).Exec(ctx); err != nil {
		return nil, err
	}

	return accessorialCharge, nil
}
