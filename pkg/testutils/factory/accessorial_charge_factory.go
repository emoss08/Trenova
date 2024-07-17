package factory

import (
	"context"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
)

type AccessorialChargeFactory struct {
	db *bun.DB
}

func NewAccessorialChargeFactory(db *bun.DB) *AccessorialChargeFactory {
	return &AccessorialChargeFactory{db: db}
}

func (o *AccessorialChargeFactory) MustCreateAccessorialCharge(ctx context.Context, orgID, buID uuid.UUID) (*models.AccessorialCharge, error) {
	// Generate the random string
	randomString := lo.RandomString(10, lo.LettersCharset)

	accessorialCharge := &models.AccessorialCharge{
		Status:         property.StatusActive,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Method:         "Distance",
		Code:           randomString,
		Description:    "Test Accessorial Charge",
	}

	if _, err := o.db.NewInsert().Model(accessorialCharge).Exec(ctx); err != nil {
		return nil, err
	}

	return accessorialCharge, nil
}
