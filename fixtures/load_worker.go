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
	"time"

	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/uptrace/bun"
)

func loadWorkers(ctx context.Context, db *bun.DB, gen *gen.CodeGenerator, orgID, buID uuid.UUID) error {
	count, err := db.NewSelect().Model((*models.Worker)(nil)).Count(ctx)
	if err != nil {
		return err
	}

	state := new(models.UsState)
	err = db.NewSelect().Model(state).Where("abbreviation = ?", "AL").Scan(ctx)
	if err != nil {
		return err
	}

	if count < 20 {
		for i := 0; i < 20; i++ {
			worker := models.Worker{
				BusinessUnitID: buID,
				OrganizationID: orgID,
				Status:         property.StatusActive,
				FirstName:      "TEST",
				LastName:       fmt.Sprintf("WORKER-%d", i),
				WorkerType:     property.WorkerTypeEmployee,
				WorkerProfile: &models.WorkerProfile{
					BusinessUnitID: buID,
					OrganizationID: orgID,
					LicenseNumber:  fmt.Sprintf("TEST-%d", i),
					StateID:        &state.ID,
					Endorsements:   property.WorkerEndorsementNone,
					DateOfBirth:    &pgtype.Date{Valid: true, Time: time.Now()},
				},
			}

			err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				mkg, mErr := models.QueryWorkerMasterKeyGenerationByOrgID(ctx, db, orgID)
				if mErr != nil {
					return mErr
				}

				return worker.InsertWorker(ctx, tx, gen, mkg.Pattern)
			})
			if err != nil {
				return err
			}
		}

		return nil
	}

	return nil
}
