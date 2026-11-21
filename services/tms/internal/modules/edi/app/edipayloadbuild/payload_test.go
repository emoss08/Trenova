package edipayloadbuild

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
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
		ProNumber:         "S2603001042",
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
	require.Equal(t, "S2603001042", payload.ProNumber)
	require.Zero(t, payload.MoveNumber)
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
	require.Equal(
		t,
		map[string]string{"BM": "BOL-1001", "CN": "PRO-1001"},
		payload.FreightInvoice.ReferenceNumbers,
		"references go out under their X12 qualifiers, never Trenova's record id",
	)
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
		Type:       shipmentevent.TypeStopDeparted,
		Summary:    "Departed stop",
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
	require.Equal(t, eventID, payload.ShipmentStatus.EventID)
	require.Equal(
		t,
		map[string]string{"BM": "BOL-214", "CN": "PRO-214"},
		payload.ShipmentStatus.References,
	)
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

func TestBuildShipmentEventStatusPayload_SendsADelayAsShipmentDelayed(t *testing.T) {
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
	require.Equal(t, "SD", payload.ShipmentStatus.StatusCode)
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
		Notes:                 "Driver says the consignee was closed",
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
	require.Equal(t, "Open", payload.ShipmentStatus.ServiceFailureStatus)
	require.Equal(
		t,
		map[string]string{"BM": "BOL-SF", "CN": "PRO-SF"},
		payload.ShipmentStatus.References,
		"internal ids, notes and statuses never reach a partner as references",
	)
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

func TestBuildFreightInvoicePayload_QuotesTheReferenceTheShipperTenderedUnder(t *testing.T) {
	t.Parallel()

	tendered := BuildFreightInvoicePayload(&invoice.Invoice{
		ID:          pulid.MustNew("inv_"),
		Number:      "INV-2001",
		ShipmentID:  pulid.MustNew("shp_"),
		ShipmentBOL: "BOL-2001",
		Shipment: &shipment.Shipment{
			BOL:               "BOL-2001",
			ExternalReference: " MFG-LOAD-88 ",
		},
	}).FreightInvoice
	require.Equal(t, "MFG-LOAD-88", tendered.ShipperReference)

	entered := BuildFreightInvoicePayload(&invoice.Invoice{
		ID:          pulid.MustNew("inv_"),
		Number:      "INV-2002",
		ShipmentBOL: "BOL-2002",
		Shipment:    &shipment.Shipment{BOL: "BOL-2002"},
	}).FreightInvoice
	require.Equal(t, "BOL-2002", entered.ShipperReference,
		"a load no shipper tendered is known by its bill of lading")

	consolidated := BuildFreightInvoicePayload(&invoice.Invoice{
		ID:     pulid.MustNew("inv_"),
		Number: "INV-2003",
	}).FreightInvoice
	require.Empty(t, consolidated.ShipperReference)
}

func TestBuildTenderPayload_CarriesPaymentEquipmentAndHazmat(t *testing.T) {
	t.Parallel()

	source := &shipment.Shipment{
		ID:           pulid.MustNew("shp_"),
		FreightTerms: shipmentvocab.FreightTermsCollect,
		TrailerType:  &equipmenttype.EquipmentType{Code: "REEFER"},
		TractorType:  &equipmenttype.EquipmentType{Code: "DAYCAB"},
		Commodities: []*shipment.ShipmentCommodity{
			{
				CommodityID: pulid.MustNew("com_"),
				Commodity: &commodity.Commodity{
					Name:                "Paint",
					HazardousMaterialID: pulid.MustNew("hm_"),
				},
			},
			{
				CommodityID: pulid.MustNew("com_"),
				Commodity:   &commodity.Commodity{Name: "Paper"},
			},
		},
	}

	payload := BuildTenderPayload(source)

	require.Equal(t, "Collect", payload.Payment)
	require.Equal(t, "REEFER", payload.Equipment)
	require.Len(t, payload.Commodities, 2)
	require.True(t, payload.Commodities[0].Hazmat)
	require.False(t, payload.Commodities[1].Hazmat)

	source.TrailerType = nil
	require.Equal(t, "DAYCAB", BuildTenderPayload(source).Equipment,
		"a load that names no trailer is known by its tractor")
	source.TractorType = nil
	require.Empty(t, BuildTenderPayload(source).Equipment)
}

func TestBuildShipmentEventStatusPayload_CodesTheStopThatMoved(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	pickupID := pulid.MustNew("stp_")
	deliveryID := pulid.MustNew("stp_")
	const occurredAt = int64(1_715_817_600)
	stops := func(pickupDeparture *int64) *shipment.Shipment {
		return &shipment.Shipment{
			ID: pulid.MustNew("shp_"),
			Moves: []*shipment.ShipmentMove{{
				ID: moveID,
				Stops: []*shipment.Stop{
					{
						ID:       deliveryID,
						Type:     shipmentvocab.StopTypeDelivery,
						Sequence: 2,
					},
					{
						ID:              pickupID,
						Type:            shipmentvocab.StopTypePickup,
						Sequence:        1,
						ActualArrival:   new(occurredAt - 600),
						ActualDeparture: pickupDeparture,
					},
				},
			}},
		}
	}

	tests := []struct {
		name      string
		event     shipmentevent.Event
		departure *int64
		code      string
		stopID    pulid.ID
	}{
		{
			name:   "arriving at the pickup",
			event:  shipmentevent.Event{Type: shipmentevent.TypeMoveDeparted, MoveID: moveID},
			code:   "X3",
			stopID: pickupID,
		},
		{
			name:      "leaving the pickup",
			event:     shipmentevent.Event{Type: shipmentevent.TypeMoveDeparted, MoveID: moveID},
			departure: new(occurredAt - 60),
			code:      "AF",
			stopID:    pickupID,
		},
		{
			name:      "a departure recorded after the event is not yet the event",
			event:     shipmentevent.Event{Type: shipmentevent.TypeMoveDeparted, MoveID: moveID},
			departure: new(occurredAt + 3600),
			code:      "X3",
			stopID:    pickupID,
		},
		{
			name:   "the move completing is the delivery",
			event:  shipmentevent.Event{Type: shipmentevent.TypeMoveArrived, MoveID: moveID},
			code:   "D1",
			stopID: deliveryID,
		},
		{
			name: "arriving at a pickup the stop names",
			event: shipmentevent.Event{
				Type:   shipmentevent.TypeStopArrived,
				MoveID: moveID,
				StopID: pickupID,
			},
			code:   "X3",
			stopID: pickupID,
		},
		{
			name: "leaving a pickup the stop names",
			event: shipmentevent.Event{
				Type:   shipmentevent.TypeStopDeparted,
				MoveID: moveID,
				StopID: pickupID,
			},
			code:   "AF",
			stopID: pickupID,
		},
		{
			name: "arriving at the delivery",
			event: shipmentevent.Event{
				Type:   shipmentevent.TypeStopArrived,
				MoveID: moveID,
				StopID: deliveryID,
			},
			code:   "X1",
			stopID: deliveryID,
		},
		{
			name: "leaving the last delivery is delivered",
			event: shipmentevent.Event{
				Type:   shipmentevent.TypeStopDeparted,
				MoveID: moveID,
				StopID: deliveryID,
			},
			code:   "D1",
			stopID: deliveryID,
		},
		{
			name:   "a driver committed to the move dispatches the load from its pickup",
			event:  shipmentevent.Event{Type: shipmentevent.TypeDriverAssigned, MoveID: moveID},
			code:   "P1",
			stopID: pickupID,
		},
		{
			name:   "a carrier covering the move dispatches the load",
			event:  shipmentevent.Event{Type: shipmentevent.TypeCarrierAssigned, MoveID: moveID},
			code:   "P1",
			stopID: pickupID,
		},
		{
			name: "a shipment saved as assigned is dispatched",
			event: shipmentevent.Event{
				Type: shipmentevent.TypeStatusChanged,
				Metadata: map[string]any{
					"newStatus": string(shipmentvocab.StatusPartiallyAssigned),
				},
			},
			code:   "P1",
			stopID: pickupID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			event := tt.event
			event.ID = pulid.MustNew("se_")
			event.OccurredAt = occurredAt
			status := BuildShipmentEventStatusPayload(&event, stops(tt.departure)).ShipmentStatus
			require.Equal(t, tt.code, status.StatusCode)
			require.Equal(t, tt.stopID, status.StopID)
		})
	}
}

func TestStopStatusCode_CodesEveryStopKind(t *testing.T) {
	t.Parallel()

	pickup := &shipment.Stop{Type: shipmentvocab.StopTypeSplitPickup}
	delivery := &shipment.Stop{Type: shipmentvocab.StopTypeSplitDelivery}

	require.Equal(t, "X3", stopStatusCode(pickup, false, false))
	require.Equal(t, "AF", stopStatusCode(pickup, true, false))
	require.Equal(t, "X1", stopStatusCode(delivery, false, false))
	require.Equal(t, "CD", stopStatusCode(delivery, true, false))
	require.Equal(t, "D1", stopStatusCode(delivery, true, true))
	require.Equal(t, "D1", stopStatusCode(nil, true, true))
	require.Equal(t, "X3", stopStatusCode(nil, false, false))
}

func TestBuildShipmentEventStatusPayload_OnlyTheShipmentsLastStopIsDelivered(t *testing.T) {
	t.Parallel()

	firstMoveID := pulid.MustNew("sm_")
	secondMoveID := pulid.MustNew("sm_")
	relayID := pulid.MustNew("stp_")
	dropID := pulid.MustNew("stp_")
	finalID := pulid.MustNew("stp_")
	canceledID := pulid.MustNew("stp_")
	source := &shipment.Shipment{
		ID: pulid.MustNew("shp_"),
		Moves: []*shipment.ShipmentMove{
			{
				ID:       secondMoveID,
				Sequence: 2,
				Stops: []*shipment.Stop{
					{ID: pulid.MustNew("stp_"), Type: shipmentvocab.StopTypePickup, Sequence: 1},
					{ID: dropID, Type: shipmentvocab.StopTypeSplitDelivery, Sequence: 2},
					{ID: finalID, Type: shipmentvocab.StopTypeDelivery, Sequence: 3},
					{
						ID:       canceledID,
						Type:     shipmentvocab.StopTypeDelivery,
						Sequence: 4,
						Status:   shipmentvocab.StopStatusCanceled,
					},
				},
			},
			{
				ID:       firstMoveID,
				Sequence: 1,
				Stops: []*shipment.Stop{
					{ID: pulid.MustNew("stp_"), Type: shipmentvocab.StopTypePickup, Sequence: 1},
					{ID: relayID, Type: shipmentvocab.StopTypeDelivery, Sequence: 2},
				},
			},
		},
	}

	tests := []struct {
		name   string
		moveID pulid.ID
		stopID pulid.ID
		code   string
	}{
		{
			name:   "a relay at the end of the first move",
			moveID: firstMoveID,
			stopID: relayID,
			code:   "CD",
		},
		{name: "a drop in the middle of a move", moveID: secondMoveID, stopID: dropID, code: "CD"},
		{
			name:   "the shipment's last active stop",
			moveID: secondMoveID,
			stopID: finalID,
			code:   "D1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			event := &shipmentevent.Event{
				ID:         pulid.MustNew("se_"),
				Type:       shipmentevent.TypeStopDeparted,
				MoveID:     tt.moveID,
				StopID:     tt.stopID,
				OccurredAt: 1_715_817_600,
			}
			status := BuildShipmentEventStatusPayload(event, source).ShipmentStatus
			require.Equal(t, tt.code, status.StatusCode)
			require.Equal(t, tt.stopID, status.StopID)
		})
	}
}

func TestBuildFreightInvoicePayload_QuotesTheOrdersPurchaseOrder(t *testing.T) {
	t.Parallel()

	single := BuildFreightInvoicePayload(&invoice.Invoice{
		ID:            pulid.MustNew("inv_"),
		Number:        "INV-3001",
		ShipmentID:    pulid.MustNew("shp_"),
		ShipmentCount: 1,
		PONumber:      " 8978300638 ",
	}).FreightInvoice
	require.Equal(t, "8978300638", single.PONumber)
	require.Equal(t, "8978300638", single.ReferenceNumbers["PO"])
	require.Empty(t, single.Shipments, "a single-shipment invoice lists no shipments")

	consolidated := BuildFreightInvoicePayload(&invoice.Invoice{
		ID:            pulid.MustNew("inv_"),
		Number:        "INV-3002",
		ShipmentCount: 2,
		PONumber:      "PO-55",
	}).FreightInvoice
	require.Equal(t, "PO-55", consolidated.PONumber)

	unordered := BuildFreightInvoicePayload(&invoice.Invoice{
		ID:     pulid.MustNew("inv_"),
		Number: "INV-3003",
	}).FreightInvoice
	require.Empty(t, unordered.PONumber)
}
