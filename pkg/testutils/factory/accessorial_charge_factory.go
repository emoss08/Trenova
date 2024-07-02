package factory

import (
	"context"
	"log"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AccessorialChargeFactory struct {
	db *bun.DB
}

func NewAccessorialChargeFactory(db *bun.DB) *AccessorialChargeFactory {
	return &AccessorialChargeFactory{db: db}
}

func (o *AccessorialChargeFactory) MustCreateAccessorialCharge(ctx context.Context, orgID, buID uuid.UUID) (*models.AccessorialCharge, error) {
	// Define the length of the random string
	length := 10

	// Define the character ranges to include (numeric and lowercase alphabet)
	charRanges := []utils.CharRange{utils.CharRangeNumeric, utils.CharRangeAlphaLowerCase}

	// Define any extra characters to include
	extraChars := "!@#$"

	// Generate the random string
	randomString, err := utils.GenerateRandomString(length, charRanges, extraChars)
	if err != nil {
		log.Fatalf("Error generating random string: %v", err)
	}

	accessorialCharge := &models.AccessorialCharge{
		Status:         property.StatusActive,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Method:         "Distance",
		Code:           randomString,
		Description:    "Test Accessorial Charge",
	}

	_, err = o.db.NewInsert().Model(accessorialCharge).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return accessorialCharge, nil
}
