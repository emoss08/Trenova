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
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

func LoadMasterKeyGeneration(ctx context.Context, db *bun.DB, orgID, buID uuid.UUID) (*models.MasterKeyGeneration, error) {
	masterKeyGeneration := new(models.MasterKeyGeneration)
	err := db.NewSelect().
		Model(masterKeyGeneration).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		// Create a new master key generation if it does not exist.
		masterKeyGeneration = &models.MasterKeyGeneration{
			BusinessUnitID: buID,
			OrganizationID: orgID,
		}

		_, err = db.NewInsert().Model(masterKeyGeneration).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("error creating master key generation: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error fetching master key generation: %w", err)
	}

	return masterKeyGeneration, nil
}

func LoadWorkerMasterKeyGeneration(ctx context.Context, db *bun.DB, mkg *models.MasterKeyGeneration) error {
	workerMasterKey := new(models.WorkerMasterKeyGeneration)
	err := db.NewSelect().
		Model(workerMasterKey).
		Where("master_key_id = ?", mkg.ID).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		// Create a new worker master key generation if it does not exist.
		workerMasterKey = &models.WorkerMasterKeyGeneration{
			Pattern:     "TYPE-LASTNAME-COUNTER",
			MasterKeyID: &mkg.ID,
			MasterKey:   mkg,
		}

		_, err = db.NewInsert().Model(workerMasterKey).Exec(ctx)
		if err != nil {
			return fmt.Errorf("error creating worker master key generation: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("error fetching worker master key generation: %w", err)
	}

	return nil
}

func LoadLocationMasterKeyGeneration(ctx context.Context, db *bun.DB, mkg *models.MasterKeyGeneration) error {
	locationMasterKey := new(models.LocationMasterKeyGeneration)
	err := db.NewSelect().
		Model(locationMasterKey).
		Where("master_key_id = ?", mkg.ID).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		// Create a new location master key generation if it does not exist.
		locationMasterKey = &models.LocationMasterKeyGeneration{
			Pattern:     "TYPE-COUNTER",
			MasterKeyID: &mkg.ID,
			MasterKey:   mkg,
		}

		_, err = db.NewInsert().Model(locationMasterKey).Exec(ctx)
		if err != nil {
			return fmt.Errorf("error creating location master key generation: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("error fetching location master key generation: %w", err)
	}

	return nil
}

func LoadCustomerMasterKeyGeneration(ctx context.Context, db *bun.DB, mkg *models.MasterKeyGeneration) error {
	customerMasterKey := new(models.CustomerMasterKeyGeneration)
	err := db.NewSelect().
		Model(customerMasterKey).
		Where("master_key_id = ?", mkg.ID).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		// Create a new customer master key generation if it does not exist.
		customerMasterKey = &models.CustomerMasterKeyGeneration{
			Pattern:     "TYPE-COUNTER",
			MasterKeyID: &mkg.ID,
			MasterKey:   mkg,
		}

		_, err = db.NewInsert().Model(customerMasterKey).Exec(ctx)
		if err != nil {
			return fmt.Errorf("error creating customer master key generation: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("error fetching customer master key generation: %w", err)
	}

	return nil
}
