package fixtures

import (
	"context"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

func loadBusinessUnit(ctx context.Context, db *bun.DB) (*models.BusinessUnit, error) {
	bu := new(models.BusinessUnit)

	exists, err := db.NewSelect().Model(bu).Where("name = ?", "Trenova Logistics").Exists(ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		bu = &models.BusinessUnit{
			Name:        "Trenova Logistics",
			PhoneNumber: "704-555-1212",
		}

		_, err = db.NewInsert().Model(bu).Exec(ctx)
		if err != nil {
			return nil, err
		}

		return bu, nil
	}

	err = db.NewSelect().Model(bu).Where("name = ?", "Trenova Logistics").Scan(ctx)
	if err != nil {
		return nil, err
	}

	return bu, nil
}
