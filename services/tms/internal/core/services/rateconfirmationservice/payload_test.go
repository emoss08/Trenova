package rateconfirmationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildContext(t *testing.T) {
	windowEnd := int64(1783600200)
	params := payloadParams{
		CompanyName: "Trenova Logistics",
		Revision:    2,
		Shipment: &shipment.Shipment{
			ProNumber: "PRO-1001",
			BOL:       "BOL-42",
		},
		Carrier: &carrier.Carrier{
			Name:            "Eastline Transport LLC",
			SCAC:            "ESTL",
			MCNumber:        "123456",
			DOTNumber:       "7654321",
			PaymentTermDays: 30,
		},
		Assignment: &shipment.CarrierAssignment{
			RateMethod:            shipment.CarrierRateMethodFlat,
			BaseRate:              decimal.NewFromInt(1850),
			BaseAmount:            decimal.NewFromInt(1850),
			FuelSurcharge:         decimal.RequireFromString("142.50"),
			AccessorialTotal:      decimal.NewFromInt(75),
			TotalCost:             decimal.RequireFromString("2067.50"),
			CurrencyCode:          "USD",
			ExternalDriverName:    "R. Alvarez",
			ExternalTractorNumber: "4482",
			Accessorials: []*shipment.CarrierAssignmentAccessorial{
				{Description: "Liftgate", Amount: decimal.NewFromInt(75)},
			},
		},
		Move: &shipment.ShipmentMove{
			Stops: []*shipment.Stop{
				{
					Type:                 shipment.StopTypePickup,
					ScheduledWindowStart: 1783585800,
					ScheduledWindowEnd:   &windowEnd,
					Location: &location.Location{
						Name:         "Lakeside Distribution",
						AddressLine1: "2100 Harbor Dr",
						City:         "Green Bay",
						PostalCode:   "54302",
					},
				},
				{
					Type:                 shipment.StopTypeDelivery,
					ScheduledWindowStart: 1783665000,
					AddressLine:          "880 Commerce Pkwy, Des Moines, IA 50313",
				},
			},
		},
	}

	out := buildContext(params)

	assert.Equal(t, "Trenova Logistics", out.CompanyName)
	assert.Equal(t, "Eastline Transport LLC", out.CarrierName)
	assert.Equal(t, "PRO-1001", out.ShipmentProNumber)
	assert.Equal(t, "Revision 2", out.RevisionLabel)
	assert.Equal(t, "Flat", out.RateMethodLabel)
	assert.Equal(t, "1850.00 flat", out.BaseRateLabel)
	assert.Equal(t, "1850.00", out.BaseAmount)
	assert.Equal(t, "142.50", out.FuelSurcharge)
	assert.Equal(t, "2067.50", out.TotalCost)
	assert.Equal(t, "Net 30 from clean POD", out.PaymentTermsLabel)

	require.Len(t, out.Stops, 2)
	assert.Equal(t, 1, out.Stops[0].Sequence)
	assert.Equal(t, "Pickup", out.Stops[0].Type)
	assert.Equal(t, "Lakeside Distribution", out.Stops[0].Name)
	assert.Equal(t, "2100 Harbor Dr, Green Bay 54302", out.Stops[0].Address)
	assert.NotEmpty(t, out.Stops[0].Window)
	assert.Equal(t, "Delivery", out.Stops[1].Type)
	assert.Equal(t, "880 Commerce Pkwy, Des Moines, IA 50313", out.Stops[1].Address)

	require.Len(t, out.Accessorials, 1)
	assert.Equal(t, "Liftgate", out.Accessorials[0].Description)
	assert.Equal(t, "75.00", out.Accessorials[0].Amount)
}

func TestBuildContextPerMile(t *testing.T) {
	out := buildContext(payloadParams{
		Revision: 1,
		Shipment: &shipment.Shipment{ProNumber: "PRO-2"},
		Carrier:  &carrier.Carrier{Name: "Carrier", PaymentTermDays: 0},
		Assignment: &shipment.CarrierAssignment{
			RateMethod: shipment.CarrierRateMethodPerMile,
			BaseRate:   decimal.RequireFromString("2.35"),
			BaseAmount: decimal.RequireFromString("1057.50"),
			TotalCost:  decimal.RequireFromString("1057.50"),
		},
	})

	assert.Equal(t, "Per mile", out.RateMethodLabel)
	assert.Equal(t, "2.35/mi", out.BaseRateLabel)
	assert.Equal(t, "Due on receipt of clean POD", out.PaymentTermsLabel)
	assert.Empty(t, out.Stops)
}
