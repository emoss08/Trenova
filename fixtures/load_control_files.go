package fixtures

import (
	"context"
	"database/sql"
	"errors"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

func loadShipmentControl(ctx context.Context, db *bun.DB, orgID, buID uuid.UUID) error {
	shipmentControl := new(models.ShipmentControl)
	err := db.NewSelect().
		Model(shipmentControl).
		Where("sc.organization_id = ?", orgID).
		Where("sc.business_unit_id = ?", buID).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		shipControl := &models.ShipmentControl{
			OrganizationID: orgID,
			BusinessUnitID: buID,
		}

		if _, err = db.NewInsert().Model(shipControl).Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}
