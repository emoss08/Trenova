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
	"strconv"
	"sync"
	"time"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	batchSize           = 5000
	numWorkers          = 20
	totalEquipmentTypes = 100_000
)

func LoadEquipmentTypes(ctx context.Context, db *bun.DB, orgID, buID uuid.UUID) error {
	start := time.Now()

	count, err := db.NewSelect().Model((*models.EquipmentType)(nil)).Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count existing equipment types: %w", err)
	}

	if count > 0 {
		log.Printf("Equipment types already loaded. Count: %d", count)
		return nil
	}

	log.Println("Starting to load equipment types...")

	var wg sync.WaitGroup
	jobs := make(chan []*models.EquipmentType, numWorkers)
	errors := make(chan error, numWorkers)

	// Start worker goroutines
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go worker(ctx, db, &wg, jobs, errors)
	}

	// Start error handling goroutine
	errChan := make(chan error)
	go func() {
		for err := range errors {
			if err != nil {
				errChan <- err
				return
			}
		}
		close(errChan)
	}()

	// Generate and send jobs
	go func() {
		defer close(jobs)
		for i := 0; i < totalEquipmentTypes; i += batchSize {
			batch := generateBatch(i, batchSize, orgID, buID)
			jobs <- batch
		}
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(errors)

	// Check for any errors
	if err := <-errChan; err != nil {
		return fmt.Errorf("error occurred while loading equipment types: %w", err)
	}

	log.Printf("Finished loading %d equipment types in %v", totalEquipmentTypes, time.Since(start))
	return nil
}

func worker(ctx context.Context, db *bun.DB, wg *sync.WaitGroup, jobs <-chan []*models.EquipmentType, errors chan<- error) {
	defer wg.Done()
	for batch := range jobs {
		if _, err := db.NewInsert().Model(&batch).Exec(ctx); err != nil {
			errors <- fmt.Errorf("failed to insert batch: %w", err)
			return
		}
	}
}

func generateBatch(start, size int, orgID, buID uuid.UUID) []*models.EquipmentType {
	batch := make([]*models.EquipmentType, 0, size)
	for i := start; i < start+size && i < totalEquipmentTypes; i++ {
		equipType := &models.EquipmentType{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Code:           strconv.Itoa(i),
			Description:    fmt.Sprintf("Test Equipment Type %d", i),
		}
		batch = append(batch, equipType)
	}
	return batch
}
