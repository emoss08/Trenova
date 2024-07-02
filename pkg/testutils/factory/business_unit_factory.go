package factory

import (
	"context"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

type BusinessUnitFactory struct {
	db *bun.DB
}

func NewBusinessUnitFactory(db *bun.DB) *BusinessUnitFactory {
	return &BusinessUnitFactory{db: db}
}

func (b *BusinessUnitFactory) CreateBusinessUnit(ctx context.Context) (*models.BusinessUnit, error) {
	bu := &models.BusinessUnit{
		Name:        "Trenova Logistics",
		PhoneNumber: "704-555-1212",
	}

	_, err := b.db.NewInsert().Model(bu).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return bu, nil
}
