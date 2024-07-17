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

package models_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/testutils"
	"github.com/emoss08/trenova/pkg/testutils/factory"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestShipment_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		s := &models.Shipment{
			BusinessUnitID:      uuid.New(),
			OrganizationID:      uuid.New(),
			Status:              property.ShipmentStatusCompleted,
			ProNumber:           "PRO",
			RatingMethod:        property.ShipmentRatingMethodFlatRate,
			RatingUnit:          1,
			FreightChargeAmount: decimal.NewFromInt(100),
			OtherChargeAmount:   decimal.NewFromInt(10),
			TotalChargeAmount:   decimal.NewFromInt(110),
			BillOfLading:        "BOL",
		}

		err := s.Validate()
		require.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {
		s := &models.Shipment{
			BusinessUnitID:      uuid.New(),
			Status:              property.ShipmentStatusCompleted,
			ProNumber:           "PRO",
			RatingMethod:        property.ShipmentRatingMethodFlatRate,
			RatingUnit:          1,
			FreightChargeAmount: decimal.NewFromInt(100),
			OtherChargeAmount:   decimal.NewFromInt(10),
			TotalChargeAmount:   decimal.NewFromInt(110),
			BillOfLading:        "BOL",
		}

		err := s.Validate()
		require.NoError(t, err)
	})
}

func TestShipment_DBValidate(t *testing.T) {
	ctx := context.Background()
	server := testutils.SetupTestServer(t)

	t.Run("valid", func(t *testing.T) {
		s := &models.Shipment{
			BusinessUnitID:      uuid.New(),
			OrganizationID:      uuid.New(),
			Status:              property.ShipmentStatusNew,
			ProNumber:           "PRO",
			RatingMethod:        property.ShipmentRatingMethodFlatRate,
			RatingUnit:          1,
			FreightChargeAmount: decimal.NewFromInt(100),
			OtherChargeAmount:   decimal.NewFromInt(10),
			TotalChargeAmount:   decimal.NewFromInt(110),
			BillOfLading:        "BOL",
		}

		err := s.DBValidate(ctx, server.DB)
		require.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {
		s := &models.Shipment{
			BusinessUnitID:      uuid.New(),
			OrganizationID:      uuid.New(),
			Status:              property.ShipmentStatusCompleted,
			ProNumber:           "PRO",
			RatingMethod:        property.ShipmentRatingMethodFlatRate,
			RatingUnit:          1,
			FreightChargeAmount: decimal.NewFromInt(100),
			TotalChargeAmount:   decimal.NewFromInt(110),
			BillOfLading:        "BOL",
		}

		err := s.DBValidate(ctx, server.DB)
		require.Error(t, err)
	})
}

func TestShipment_CalculateTotalChargeAmount(t *testing.T) {
	t.Run("calculate", func(t *testing.T) {
		s := &models.Shipment{
			BusinessUnitID:      uuid.New(),
			OrganizationID:      uuid.New(),
			Status:              property.ShipmentStatusCompleted,
			ProNumber:           "PRO",
			RatingMethod:        property.ShipmentRatingMethodFlatRate,
			RatingUnit:          1,
			OtherChargeAmount:   decimal.NewFromInt(10),
			FreightChargeAmount: decimal.NewFromInt(100),
			BillOfLading:        "BOL",
		}
		// Calculate the total charge amount
		s.CalculateTotalChargeAmount()

		require.Equal(t, decimal.NewFromInt(110), s.TotalChargeAmount)
	})
}

func TestShipment_MarkReadyToBill(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		s := &models.Shipment{
			BusinessUnitID:      uuid.New(),
			OrganizationID:      uuid.New(),
			Status:              property.ShipmentStatusCompleted,
			ProNumber:           "PRO",
			RatingMethod:        property.ShipmentRatingMethodFlatRate,
			RatingUnit:          1,
			OtherChargeAmount:   decimal.NewFromInt(10),
			FreightChargeAmount: decimal.NewFromInt(100),
			BillOfLading:        "BOL",
		}

		err := s.MarkReadyToBill()
		require.NoError(t, err)
		require.True(t, s.ReadyToBill)
	})

	t.Run("invalid", func(t *testing.T) {
		s := &models.Shipment{
			BusinessUnitID:      uuid.New(),
			OrganizationID:      uuid.New(),
			Status:              property.ShipmentStatusNew,
			ProNumber:           "PRO",
			RatingMethod:        property.ShipmentRatingMethodFlatRate,
			RatingUnit:          1,
			OtherChargeAmount:   decimal.NewFromInt(10),
			FreightChargeAmount: decimal.NewFromInt(100),
			BillOfLading:        "BOL",
		}

		err := s.MarkReadyToBill()
		require.Error(t, err)
	})
}

func TestShipment_GenerateProNumber(t *testing.T) {
	ctx := context.Background()
	server := testutils.SetupTestServer(t)
	org, err := factory.NewOrganizationFactory(server.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)

	t.Run("generate single pro number", func(t *testing.T) {
		proNumber, err := models.GenerateProNumber(ctx, server.DB, org.ID)
		require.NoError(t, err)
		require.NotEmpty(t, proNumber)
		require.Regexp(t, `^S\d{4}-\d{6}$`, proNumber)
	})

	t.Run("generate multiple pro numbers", func(t *testing.T) {
		proNumbers := make(map[string]struct{})
		for i := 0; i < 100; i++ {
			proNumber, err := models.GenerateProNumber(ctx, server.DB, org.ID)
			require.NoError(t, err)
			require.NotEmpty(t, proNumber)
			require.Regexp(t, `^S\d{4}-\d{6}$`, proNumber)
			proNumbers[proNumber] = struct{}{}
		}
		require.Len(t, proNumbers, 100, "All generated pro numbers should be unique")
	})

	t.Run("generate pro numbers for multiple organizations", func(t *testing.T) {
		org2, err := factory.NewOrganizationFactory(server.DB).MustCreateOrganization(ctx)
		require.NoError(t, err)

		proNumber1, err := models.GenerateProNumber(ctx, server.DB, org.ID)
		require.NoError(t, err)
		proNumber2, err := models.GenerateProNumber(ctx, server.DB, org2.ID)
		require.NoError(t, err)

		require.NotEqual(t, proNumber1, proNumber2)
	})

	t.Run("year transition", func(t *testing.T) {
		// Mock time to simulate year transition
		mockTime := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)
		models.SetNow(func() time.Time { return mockTime })
		defer models.SetNow(time.Now)

		proNumber1, err := models.GenerateProNumber(ctx, server.DB, org.ID)
		require.NoError(t, err)

		// Simulate transition to next year
		mockTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		models.SetNow(func() time.Time { return mockTime })

		proNumber2, err := models.GenerateProNumber(ctx, server.DB, org.ID)
		require.NoError(t, err)

		year1 := proNumber1[1:5]
		year2 := proNumber2[1:5]
		require.NotEqual(t, year1, year2, "Pro numbers should have different years")

		number1, err := strconv.Atoi(proNumber1[6:])
		require.NoError(t, err)
		number2, err := strconv.Atoi(proNumber2[6:])
		require.NoError(t, err)
		require.Equal(t, number1+1, number2, "Sequential numbers should increment across years")

		// Generate another number to ensure it continues to increment
		proNumber3, err := models.GenerateProNumber(ctx, server.DB, org.ID)
		require.NoError(t, err)
		number3, err := strconv.Atoi(proNumber3[6:])
		require.NoError(t, err)
		require.Equal(t, number2+1, number3, "Sequential numbers should continue to increment")
	})

	t.Run("concurrent generation", func(t *testing.T) {
		var wg sync.WaitGroup
		proNumbers := make(chan string, 100)
		errors := make(chan error, 100)
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				proNumber, err := models.GenerateProNumber(ctx, server.DB, org.ID)
				if err != nil {
					errors <- err
					return
				}
				proNumbers <- proNumber
			}()
		}
		wg.Wait()
		close(proNumbers)
		close(errors)

		for err := range errors {
			require.NoError(t, err)
		}

		uniqueProNumbers := make(map[string]struct{})
		for proNumber := range proNumbers {
			uniqueProNumbers[proNumber] = struct{}{}
		}
		require.Len(t, uniqueProNumbers, 100, "All concurrently generated pro numbers should be unique")
	})
}
