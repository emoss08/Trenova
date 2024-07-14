package fixtures

import (
	"context"

	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

func loadCustomers(ctx context.Context, db *bun.DB, gen *gen.CodeGenerator, orgID, buID uuid.UUID) error {
	count, err := db.NewSelect().Model((*models.Customer)(nil)).Count(ctx)
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
			customer := models.Customer{
				BusinessUnitID:      buID,
				OrganizationID:      orgID,
				Status:              property.StatusActive,
				Name:                "TEST",
				AddressLine1:        "123 Main St",
				City:                "Minneapolis",
				StateID:             state.ID,
				PostalCode:          "55401",
				AutoMarkReadyToBill: true,
			}

			err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				mkg, mErr := models.QueryCustomerMasterKeyGenerationByOrgID(ctx, db, orgID)
				if mErr != nil {
					return mErr
				}

				return customer.InsertCustomer(ctx, tx, gen, mkg.Pattern)
			})
			if err != nil {
				return err
			}
		}

		return nil
	}

	return nil
}
