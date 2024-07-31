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

func loadEquipmentTypes(ctx context.Context, db *bun.DB, orgID, buID uuid.UUID) error {
	start := time.Now()

	count, err := db.NewSelect().Model((*models.EquipmentType)(nil)).Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count existing equipment types: %w", err)
	}

	if count > 10 {
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
