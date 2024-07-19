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
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

func LoadMasterKeyGeneration(ctx context.Context, db *bun.DB, orgID, buID uuid.UUID) (*models.MasterKeyGeneration, error) {
	// Check if the organization has a master key generation entity.
	masterKeyGeneration := new(models.MasterKeyGeneration)
	_, err := db.NewSelect().
		Model(masterKeyGeneration).
		Where("organization_id = ?", orgID).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	if masterKeyGeneration.ID != uuid.Nil {
		// Return the existing master key generation.
		return masterKeyGeneration, nil
	}

	// Create a new master key generation if it does not exist.
	masterKeyGeneration = &models.MasterKeyGeneration{
		BusinessUnitID: buID,
		OrganizationID: orgID,
	}

	_, kErr := db.NewInsert().Model(masterKeyGeneration).Exec(ctx)
	if kErr != nil {
		return nil, kErr
	}

	return masterKeyGeneration, nil
}

func LoadWorkerMasterKeyGeneration(ctx context.Context, db *bun.DB, mkg *models.MasterKeyGeneration) error {
	// Check if the master key generation has a worker master key generation entity.
	workerMasterKey := new(models.WorkerMasterKeyGeneration)
	_, err := db.NewSelect().
		Model(workerMasterKey).
		Where("master_key_id = ?", mkg.ID).
		Exec(ctx)
	if err != nil {
		return err
	}

	if workerMasterKey.ID != uuid.Nil {
		// Return nil if the worker master key generation already exists.
		return nil
	}

	// Create a new worker master key generation if it does not exist.
	workerMasterKey = &models.WorkerMasterKeyGeneration{
		Pattern:     "TYPE-LASTNAME-COUNTER",
		MasterKeyID: &mkg.ID,
		MasterKey:   mkg,
	}

	_, err = db.NewInsert().Model(workerMasterKey).Exec(ctx)
	return err
}

func LoadLocationMasterKeyGeneration(ctx context.Context, db *bun.DB, mkg *models.MasterKeyGeneration) error {
	// Check if the master key generation has a location master key generation entity.
	locationMasterKey := new(models.LocationMasterKeyGeneration)
	_, err := db.NewSelect().
		Model(locationMasterKey).
		Where("master_key_id = ?", mkg.ID).
		Exec(ctx)
	if err != nil {
		return err
	}

	if locationMasterKey.ID != uuid.Nil {
		// Return nil if the location master key generation already exists.
		return nil
	}

	// Create a new location master key generation if it does not exist.
	locationMasterKey = &models.LocationMasterKeyGeneration{
		Pattern:     "CITY-STATE-COUNTER",
		MasterKeyID: &mkg.ID,
		MasterKey:   mkg,
	}

	_, err = db.NewInsert().Model(locationMasterKey).Exec(ctx)
	return err
}

func LoadCustomerMasterKeyGeneration(ctx context.Context, db *bun.DB, mkg *models.MasterKeyGeneration) error {
	// Check if the master key generation has a customer master key generation entity.
	customerMasterKey := new(models.CustomerMasterKeyGeneration)
	_, err := db.NewSelect().
		Model(customerMasterKey).
		Where("master_key_id = ?", mkg.ID).
		Exec(ctx)
	if err != nil {
		return err
	}

	if customerMasterKey.ID != uuid.Nil {
		// Return nil if the customer master key generation already exists.
		return nil
	}

	// Create a new customer master key generation if it does not exist.
	customerMasterKey = &models.CustomerMasterKeyGeneration{
		Pattern:     "NAME-COUNTER",
		MasterKeyID: &mkg.ID,
		MasterKey:   mkg,
	}

	_, err = db.NewInsert().Model(customerMasterKey).Exec(ctx)
	return err
}
