package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
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

func TestBuildFreightInvoicePayload(t *testing.T) {
	shipmentID := pulid.MustNew("sp_")
	source := &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		Number:             "INV-1001",
		InvoiceDate:        1715817600,
		ShipmentID:         shipmentID,
		ShipmentBOL:        "BOL-1001",
		ShipmentProNumber:  "PRO-1001",
		BillToName:         "Acme Logistics",
		BillToAddressLine1: "123 Main St",
		BillToCity:         "Dallas",
		BillToState:        "TX",
		BillToPostalCode:   "75001",
		BillToCountry:      "US",
		CurrencyCode:       "USD",
		TotalAmount:        decimal.NewFromInt(1250),
		Lines: []*invoice.InoviceLine{
			{
				LineNumber:  2,
				Type:        invoice.InvoiceLineTypeAccessorial,
				Description: "Liftgate",
				Amount:      decimal.NewFromInt(50),
			},
			{
				LineNumber:  1,
				Type:        invoice.InvoiceLineTypeFreight,
				Description: "Linehaul",
				Amount:      decimal.NewFromInt(1200),
			},
		},
	}

	payload := buildFreightInvoicePayload(source)

	require.Equal(t, edi.TransactionSet210, payload.TransactionSet)
	require.NotNil(t, payload.FreightInvoice)
	require.Equal(t, source.ID, payload.FreightInvoice.InvoiceID)
	require.Equal(t, "INV-1001", payload.FreightInvoice.InvoiceNumber)
	require.Equal(t, shipmentID, payload.FreightInvoice.ShipmentID)
	require.Equal(t, "BOL-1001", payload.FreightInvoice.BOL)
	require.Equal(t, "PRO-1001", payload.FreightInvoice.ProNumber)
	require.Equal(t, "Acme Logistics", payload.FreightInvoice.BillToName)
	require.Equal(t, "TX", payload.FreightInvoice.BillToStateCode)
	require.True(t, payload.FreightInvoice.TotalAmount.Valid)
	require.True(t, decimal.NewFromInt(1250).Equal(payload.FreightInvoice.TotalAmount.Decimal))
	require.Equal(t, "PRO-1001", payload.FreightInvoice.ReferenceNumbers["pro"])
	require.Len(t, payload.FreightInvoice.LineCharges, 2)
	require.Equal(t, int64(1), payload.FreightInvoice.LineCharges[0].Sequence)
	require.Equal(t, "Freight", payload.FreightInvoice.LineCharges[0].Code)
}

func TestBuildShipmentEventStatusPayload(t *testing.T) {
	shipmentID := pulid.MustNew("sp_")
	moveID := pulid.MustNew("sm_")
	stopID := pulid.MustNew("stp_")
	eventID := pulid.MustNew("se_")
	source := &shipment.Shipment{
		ID:        shipmentID,
		BOL:       "BOL-214",
		ProNumber: "PRO-214",
		Moves: []*shipment.ShipmentMove{
			{
				ID: moveID,
				Stops: []*shipment.Stop{
					{
						ID: stopID,
						Location: &location.Location{
							City:  "Dallas",
							State: &usstate.UsState{Abbreviation: "TX"},
						},
					},
				},
			},
		},
	}
	event := &shipmentevent.Event{
		ID:         eventID,
		ShipmentID: shipmentID,
		MoveID:     moveID,
		StopID:     stopID,
		Type:       shipmentevent.TypeStopCompleted,
		Summary:    "Stop completed",
		OccurredAt: 1715817600,
	}

	payload := buildShipmentEventStatusPayload(event, source)

	require.Equal(t, edi.TransactionSet214, payload.TransactionSet)
	require.NotNil(t, payload.ShipmentStatus)
	require.Equal(t, shipmentID, payload.ShipmentStatus.ShipmentID)
	require.Equal(t, "BOL-214", payload.ShipmentStatus.BOL)
	require.Equal(t, "D1", payload.ShipmentStatus.StatusCode)
	require.Equal(t, int64(1715817600), payload.ShipmentStatus.EventDate)
	require.Equal(t, "Dallas", payload.ShipmentStatus.City)
	require.Equal(t, "TX", payload.ShipmentStatus.StateCode)
	require.Equal(t, eventID.String(), payload.ShipmentStatus.References["eventId"])
	require.Equal(t, "PRO-214", payload.ShipmentStatus.References["pro"])
}

func TestBuildShipmentStatusPayloadSkipsNilMoveFallback(t *testing.T) {
	source := &shipment.Shipment{
		ID:    pulid.MustNew("sp_"),
		BOL:   "BOL-NIL-MOVE",
		Moves: []*shipment.ShipmentMove{nil},
	}

	var payload edi.DocumentPayload
	require.NotPanics(t, func() {
		payload = buildShipmentStatusPayload(source)
	})

	require.Equal(t, edi.TransactionSet214, payload.TransactionSet)
	require.NotNil(t, payload.ShipmentStatus)
	require.Equal(t, source.ID, payload.ShipmentStatus.ShipmentID)
	require.Equal(t, "BOL-NIL-MOVE", payload.ShipmentStatus.BOL)
	require.Zero(t, payload.ShipmentStatus.EventDate)
}

func TestAddRequiredIDDeduplicatesAndSkipsNil(t *testing.T) {
	required := map[edi.MappingEntityType][]pulid.ID{}
	id := pulid.MustNew("cus_")

	addRequiredID(required, edi.MappingEntityTypeCustomer, pulid.Nil)
	addRequiredID(required, edi.MappingEntityTypeCustomer, id)
	addRequiredID(required, edi.MappingEntityTypeCustomer, id)

	require.Equal(t, []pulid.ID{id}, required[edi.MappingEntityTypeCustomer])
}
