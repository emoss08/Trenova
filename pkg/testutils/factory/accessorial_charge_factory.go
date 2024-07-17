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
