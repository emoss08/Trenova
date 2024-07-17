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
