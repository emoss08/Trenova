package edipayloadbuild

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
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
						Type:                 shipmentvocab.StopTypePickup,
						ScheduleType:         shipmentvocab.StopScheduleTypeOpen,
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

	payload := BuildTenderPayload(source)

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
		Lines: []*invoice.InvoiceLine{
			{
				LineNumber:  2,
				Type:        invoice.InvoiceLineTypeAccessorial,
				Description: "Liftgate",
				ChargeCode:  "LFT",
				Amount:      decimal.NewFromInt(50),
			},
			{
				LineNumber:  3,
				Type:        invoice.InvoiceLineTypeFreight,
				Description: "Minimum charge adjustment",
				Amount:      decimal.NewFromInt(0),
			},
			{
				LineNumber:  1,
				Type:        invoice.InvoiceLineTypeFreight,
				Description: "Linehaul",
				Amount:      decimal.NewFromInt(1200),
			},
		},
	}

	payload := BuildFreightInvoicePayload(source)

	require.Equal(t, edidocument.TransactionSet210, payload.TransactionSet)
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
	require.Len(t, payload.FreightInvoice.LineCharges, 3)
	require.Equal(t, int64(1), payload.FreightInvoice.LineCharges[0].Sequence)
	require.Equal(
		t,
		"400",
		payload.FreightInvoice.LineCharges[0].Code,
		"freight is X12 special charge 400",
	)
	require.Equal(
		t,
		"LFT",
		payload.FreightInvoice.LineCharges[1].Code,
		"an accessorial sends its catalog code",
	)
	require.Equal(
		t,
		"400",
		payload.FreightInvoice.LineCharges[2].Code,
		"every freight line is freight",
	)
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
						ID:                   stopID,
						LocationID:           pulid.MustNew("loc_"),
						Type:                 shipmentvocab.StopTypeDelivery,
						Sequence:             2,
						ScheduledWindowStart: 1715814000,
						ScheduledWindowEnd:   new(int64(1715817600)),
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
	}
	event := &shipmentevent.Event{
		ID:         eventID,
		ShipmentID: shipmentID,
		MoveID:     moveID,
		StopID:     stopID,
		Type:       shipmentevent.TypeStopCompleted,
		Summary:    "Stop completed",
		OccurredAt: 1715817600,
		Metadata: map[string]any{
			"statusReasonCode":  "NS",
			"reasonDescription": "Late pickup",
			"lateMinutes":       "45",
		},
	}

	payload := BuildShipmentEventStatusPayload(event, source)

	require.Equal(t, edidocument.TransactionSet214, payload.TransactionSet)
	require.NotNil(t, payload.ShipmentStatus)
	require.Equal(t, shipmentID, payload.ShipmentStatus.ShipmentID)
	require.Equal(t, "BOL-214", payload.ShipmentStatus.BOL)
	require.Equal(t, "PRO-214", payload.ShipmentStatus.ProNumber)
	require.Equal(t, "D1", payload.ShipmentStatus.StatusCode)
	require.Equal(t, "NS", payload.ShipmentStatus.StatusReasonCode)
	require.Equal(t, "NS", payload.ShipmentStatus.ReasonCode)
	require.Equal(t, "Late pickup", payload.ShipmentStatus.ReasonDescription)
	require.Equal(t, int64(1715817600), payload.ShipmentStatus.EventDate)
	require.Equal(t, stopID, payload.ShipmentStatus.StopID)
	require.Equal(t, string(shipmentvocab.StopTypeDelivery), payload.ShipmentStatus.StopType)
	require.Equal(t, int64(2), payload.ShipmentStatus.StopSequence)
	require.Equal(t, "Dallas Terminal", payload.ShipmentStatus.LocationName)
	require.Equal(t, "DAL", payload.ShipmentStatus.LocationCode)
	require.Equal(t, "123 Main St, Dallas, TX, 75001", payload.ShipmentStatus.AddressLine)
	require.Equal(t, "Dallas", payload.ShipmentStatus.City)
	require.Equal(t, "TX", payload.ShipmentStatus.StateCode)
	require.Equal(t, "75001", payload.ShipmentStatus.PostalCode)
	require.NotNil(t, payload.ShipmentStatus.LateMinutes)
	require.Equal(t, int64(45), *payload.ShipmentStatus.LateMinutes)
	require.Equal(t, eventID.String(), payload.ShipmentStatus.References["eventId"])
	require.Equal(t, "PRO-214", payload.ShipmentStatus.References["pro"])
}

func TestBuildShipmentStatusPayloadSkipsNilMoveFallback(t *testing.T) {
	source := &shipment.Shipment{
		ID:    pulid.MustNew("sp_"),
		BOL:   "BOL-NIL-MOVE",
		Moves: []*shipment.ShipmentMove{nil},
	}

	var payload edipayload.DocumentPayload
	require.NotPanics(t, func() {
		payload = BuildShipmentStatusPayload(source)
	})

	require.Equal(t, edidocument.TransactionSet214, payload.TransactionSet)
	require.NotNil(t, payload.ShipmentStatus)
	require.Equal(t, source.ID, payload.ShipmentStatus.ShipmentID)
	require.Equal(t, "BOL-NIL-MOVE", payload.ShipmentStatus.BOL)
	require.Zero(t, payload.ShipmentStatus.EventDate)
}

func TestBuildShipmentEventStatusPayload_PreservesDelayedMappingToA3(t *testing.T) {
	t.Parallel()

	payload := BuildShipmentEventStatusPayload(&shipmentevent.Event{
		ID:         pulid.MustNew("se_"),
		ShipmentID: pulid.MustNew("shp_"),
		Type:       shipmentevent.TypeStatusChanged,
		Metadata: map[string]any{
			"newStatus": string(shipmentvocab.StatusDelayed),
		},
	}, nil)

	require.NotNil(t, payload.ShipmentStatus)
	require.Equal(t, "A3", payload.ShipmentStatus.StatusCode)
}

func TestBuildServiceFailureShipmentStatusPayload_UsesOverridesBeforeReasonDefaults(t *testing.T) {
	t.Parallel()

	reasonID := pulid.MustNew("sfrc_")
	failure := &servicefailure.ServiceFailure{
		ID:                    pulid.MustNew("sf_"),
		ShipmentID:            pulid.MustNew("sp_"),
		ShipmentMoveID:        pulid.MustNew("sm_"),
		StopID:                pulid.MustNew("stp_"),
		ReasonCodeID:          &reasonID,
		Number:                "SF-1001",
		Type:                  servicefailure.TypeLateDelivery,
		Status:                servicefailure.StatusOpen,
		X12StatusCodeOverride: "sd",
		X12ReasonCodeOverride: "zz",
		X12ExceptionCode:      "e1",
		ActualArrival:         1715817600,
		LateMinutes:           35,
		ReasonCode: &servicefailure.ReasonCode{
			ID:                   reasonID,
			Code:                 "LATE",
			Label:                "Late delivery",
			DefaultStatusCode:    "a3",
			DefaultReasonCode:    "ns",
			DefaultExceptionCode: "x2",
		},
	}

	payload := BuildServiceFailureShipmentStatusPayload(failure, &shipment.Shipment{
		ID:        failure.ShipmentID,
		BOL:       "BOL-SF",
		ProNumber: "PRO-SF",
	})

	require.NotNil(t, payload.ShipmentStatus)
	require.Equal(t, "SD", payload.ShipmentStatus.StatusCode)
	require.Equal(t, "ZZ", payload.ShipmentStatus.StatusReasonCode)
	require.Equal(t, "ZZ", payload.ShipmentStatus.ReasonCode)
	require.Equal(t, "E1", payload.ShipmentStatus.ExceptionCode)
	require.Equal(t, "Late delivery", payload.ShipmentStatus.ReasonDescription)
	require.Equal(t, reasonID, *payload.ShipmentStatus.ServiceFailureReasonCodeID)
}

func TestBuildServiceFailureShipmentStatusPayload_UsesReasonDefaultsWithoutOverrides(t *testing.T) {
	t.Parallel()

	failure := &servicefailure.ServiceFailure{
		ID:             pulid.MustNew("sf_"),
		ShipmentID:     pulid.MustNew("sp_"),
		ShipmentMoveID: pulid.MustNew("sm_"),
		StopID:         pulid.MustNew("stp_"),
		Number:         "SF-1002",
		Type:           servicefailure.TypeLatePickup,
		Status:         servicefailure.StatusOpen,
		ActualArrival:  1715817600,
		LateMinutes:    12,
		ReasonCode: &servicefailure.ReasonCode{
			Code:                 "PULATE",
			Label:                "Late pickup",
			DefaultStatusCode:    "a3",
			DefaultReasonCode:    "ns",
			DefaultExceptionCode: "x2",
		},
	}

	payload := BuildServiceFailureShipmentStatusPayload(failure, nil)

	require.NotNil(t, payload.ShipmentStatus)
	require.Equal(t, "A3", payload.ShipmentStatus.StatusCode)
	require.Equal(t, "NS", payload.ShipmentStatus.StatusReasonCode)
	require.Equal(t, "NS", payload.ShipmentStatus.ReasonCode)
	require.Equal(t, "X2", payload.ShipmentStatus.ExceptionCode)
	require.Empty(t, ServiceFailurePayloadDiagnostics(payload.ShipmentStatus))
}

func TestBuildServiceFailureShipmentStatusPayload_UsesServiceFailureTimestampPrecedence(
	t *testing.T,
) {
	t.Parallel()

	base := &servicefailure.ServiceFailure{
		ID:             pulid.MustNew("sf_"),
		ShipmentID:     pulid.MustNew("sp_"),
		ShipmentMoveID: pulid.MustNew("sm_"),
		StopID:         pulid.MustNew("stp_"),
		Number:         "SF-1003",
		Type:           servicefailure.TypeLateDelivery,
		Status:         servicefailure.StatusOpen,
		ActualArrival:  100,
		CreatedAt:      200,
		DetectedAt:     300,
		LateMinutes:    1,
	}

	payload := BuildServiceFailureShipmentStatusPayload(base, nil)
	require.Equal(t, int64(300), payload.ShipmentStatus.EventDate)
	require.Equal(t, int64(300), payload.ShipmentStatus.EventTime)

	base.DetectedAt = 0
	payload = BuildServiceFailureShipmentStatusPayload(base, nil)
	require.Equal(t, int64(200), payload.ShipmentStatus.EventDate)
	require.Equal(t, int64(200), payload.ShipmentStatus.EventTime)

	base.CreatedAt = 0
	payload = BuildServiceFailureShipmentStatusPayload(base, nil)
	require.Equal(t, int64(100), payload.ShipmentStatus.EventDate)
	require.Equal(t, int64(100), payload.ShipmentStatus.EventTime)
}

func TestServiceFailurePayloadDiagnostics_RequiresReasonForSD(t *testing.T) {
	t.Parallel()

	diagnostics := ServiceFailurePayloadDiagnostics(&edipayload.ShipmentStatusPayload{
		StatusCode: "SD",
		EventDate:  1,
		EventTime:  1,
	})

	require.Len(t, diagnostics, 1)
	require.Equal(t, "required", diagnostics[0].Code)
	require.Equal(t, "AT7", diagnostics[0].SegmentID)
	require.Equal(t, 2, diagnostics[0].ElementPosition)
	require.Equal(t, "shipmentStatus.statusReasonCode", diagnostics[0].Path)
	require.Empty(t, ServiceFailurePayloadDiagnostics(&edipayload.ShipmentStatusPayload{
		StatusCode:       "SD",
		StatusReasonCode: "NS",
		EventDate:        1,
		EventTime:        1,
	}))
	require.Len(t, ServiceFailurePayloadDiagnostics(&edipayload.ShipmentStatusPayload{
		StatusCode: " sd ",
		EventDate:  1,
		EventTime:  1,
	}), 1)
	require.Empty(t, ServiceFailurePayloadDiagnostics(&edipayload.ShipmentStatusPayload{
		StatusCode: "A3",
		EventDate:  1,
		EventTime:  1,
	}))
}

func TestServiceFailurePayloadDiagnosticsRequiresEventDateAndTime(t *testing.T) {
	t.Parallel()

	diagnostics := ServiceFailurePayloadDiagnostics(&edipayload.ShipmentStatusPayload{
		StatusCode:       "A3",
		StatusReasonCode: "NS",
	})

	require.Contains(t, diagnosticPaths(diagnostics), "shipmentStatus.eventDate")
	require.Contains(t, diagnosticPaths(diagnostics), "shipmentStatus.eventTime")
}

func diagnosticPaths(diagnostics []edidocument.Diagnostic) []string {
	paths := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		paths = append(paths, diagnostic.Path)
	}
	return paths
}

func TestAddRequiredIDDeduplicatesAndSkipsNil(t *testing.T) {
	required := map[edi.MappingEntityType][]pulid.ID{}
	id := pulid.MustNew("cus_")

	addRequiredID(required, edi.MappingEntityTypeCustomer, pulid.Nil)
	addRequiredID(required, edi.MappingEntityTypeCustomer, id)
	addRequiredID(required, edi.MappingEntityTypeCustomer, id)

	require.Equal(t, []pulid.ID{id}, required[edi.MappingEntityTypeCustomer])
}

//go:fix inline
func int64Ptr(v int64) *int64 {
	return new(v)
}

func TestShipmentStatusPayloads_CarryTheShippersReferenceAndTheWeight(t *testing.T) {
	t.Parallel()

	weight, pieces := int64(40000), int64(24)
	tendered := &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		ProNumber:         "PRO-1042",
		BOL:               "BOL-7781",
		ExternalReference: " ACME-SHP-55 ",
		Weight:            &weight,
		Pieces:            &pieces,
	}

	status := BuildShipmentStatusPayload(tendered).ShipmentStatus
	require.Equal(t, "ACME-SHP-55", status.ShipperReference)
	require.Equal(t, "PRO-1042", status.ProNumber)
	require.Equal(t, "BOL-7781", status.BOL)
	require.Equal(t, &weight, status.Weight)
	require.Equal(t, &pieces, status.Pieces)

	event := &shipmentevent.Event{ID: pulid.MustNew("sev_"), ShipmentID: tendered.ID}
	require.Equal(t, "ACME-SHP-55",
		BuildShipmentEventStatusPayload(event, tendered).ShipmentStatus.ShipperReference)

	untendered := &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: "PRO-1043", BOL: "BOL-1"}
	require.Equal(
		t,
		"BOL-1",
		BuildShipmentStatusPayload(untendered).ShipmentStatus.ShipperReference,
		"without the shipper's own reference the bill of lading stands for it",
	)
	require.Nil(t, BuildShipmentStatusPayload(untendered).ShipmentStatus.Weight)
}
