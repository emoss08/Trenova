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
