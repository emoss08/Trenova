package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestBuildTenderPayload(t *testing.T) {
	source := &shipment.Shipment{
		ID:                pulid.MustNew("sp_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		OrganizationID:    pulid.MustNew("org_"),
		ServiceTypeID:     pulid.MustNew("st_"),
		ShipmentTypeID:    pulid.MustNew("sht_"),
		CustomerID:        pulid.MustNew("cus_"),
		FormulaTemplateID: pulid.MustNew("ft_"),
		BOL:               "BOL-123",
		RatingUnit:        1,
		Customer: &customer.Customer{
			Code: "ACME",
			Name: "Acme Logistics",
		},
		ServiceType: &servicetype.ServiceType{
			Code: "FTL",
		},
		ShipmentType: &shipmenttype.ShipmentType{
			Code: "DRY",
		},
		FormulaTemplate: &formulatemplate.FormulaTemplate{
			Name: "Standard Freight",
		},
		Moves: []*shipment.ShipmentMove{
			{
				Loaded:   true,
				Sequence: 0,
				Stops: []*shipment.Stop{
					{
						LocationID:           pulid.MustNew("loc_"),
						Type:                 shipment.StopTypePickup,
						ScheduleType:         shipment.StopScheduleTypeOpen,
						Sequence:             0,
						ScheduledWindowStart: 123,
						Location: &location.Location{
							Code:         "DAL",
							Name:         "Dallas Terminal",
							AddressLine1: "123 Main St",
							City:         "Dallas",
							PostalCode:   "75001",
							State:        &usstate.UsState{Abbreviation: "TX"},
						},
					},
				},
			},
		},
		Commodities: []*shipment.ShipmentCommodity{
			{
				CommodityID: pulid.MustNew("cmd_"),
				Pieces:      2,
				Weight:      300,
				Commodity: &commodity.Commodity{
					Name:        "Palletized freight",
					Description: "General freight",
				},
			},
		},
		AdditionalCharges: []*shipment.AdditionalCharge{
			{
				AccessorialChargeID: pulid.MustNew("acc_"),
				Method:              accessorialcharge.MethodFlat,
				Amount:              decimal.NewFromInt(25),
				Unit:                1,
				AccessorialCharge: &accessorialcharge.AccessorialCharge{
					Code:        "LFT",
					Description: "Liftgate",
				},
			},
		},
	}

	payload := buildTenderPayload(source)

	require.Equal(t, source.ID, payload.ShipmentID)
	require.Equal(t, source.CustomerID, payload.CustomerID)
	require.Equal(t, "ACME - Acme Logistics", payload.CustomerLabel)
	require.Equal(t, "FTL", payload.ServiceTypeLabel)
	require.Equal(t, "DRY", payload.ShipmentTypeLabel)
	require.Equal(t, "Standard Freight", payload.FormulaTemplateLabel)
	require.Len(t, payload.Moves, 1)
	require.Len(t, payload.Moves[0].Stops, 1)
	require.Equal(t, "DAL - Dallas Terminal", payload.Moves[0].Stops[0].LocationLabel)
	require.Equal(t, "Dallas Terminal", payload.Moves[0].Stops[0].LocationName)
	require.Equal(t, "DAL", payload.Moves[0].Stops[0].LocationCode)
	require.Equal(t, "123 Main St, Dallas, TX, 75001", payload.Moves[0].Stops[0].AddressLine)
	require.Len(t, payload.Commodities, 1)
	require.Equal(t, "Palletized freight", payload.Commodities[0].CommodityLabel)
	require.Len(t, payload.AdditionalCharges, 1)
	require.Equal(t, "LFT - Liftgate", payload.AdditionalCharges[0].AccessorialLabel)
	require.Contains(
		t,
		payload.RequiredMappingEntityIDs[edi.MappingEntityTypeCustomer],
		source.CustomerID,
	)
	require.Contains(
		t,
		payload.RequiredMappingEntityIDs[edi.MappingEntityTypeLocation],
		source.Moves[0].Stops[0].LocationID,
	)
	require.Contains(
		t,
		payload.RequiredMappingEntityIDs[edi.MappingEntityTypeCommodity],
		source.Commodities[0].CommodityID,
	)
	require.Contains(
		t,
		payload.RequiredMappingEntityIDs[edi.MappingEntityTypeAccessorialCharge],
		source.AdditionalCharges[0].AccessorialChargeID,
	)
}

func TestAddRequiredIDDeduplicatesAndSkipsNil(t *testing.T) {
	required := map[edi.MappingEntityType][]pulid.ID{}
	id := pulid.MustNew("cus_")

	addRequiredID(required, edi.MappingEntityTypeCustomer, pulid.Nil)
	addRequiredID(required, edi.MappingEntityTypeCustomer, id)
	addRequiredID(required, edi.MappingEntityTypeCustomer, id)

	require.Equal(t, []pulid.ID{id}, required[edi.MappingEntityTypeCustomer])
}
