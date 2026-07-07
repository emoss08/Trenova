package ediinboundservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	editemplates "github.com/emoss08/trenova/internal/core/domain/edi/templates"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func renderBase204(t *testing.T, payload edi.LoadTenderPayload) string {
	t.Helper()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	versionID := pulid.MustNew("editv_")
	profile := &edi.EDIPartnerDocumentProfile{
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionOutbound,
		ValidationMode: edi.ValidationModeWarnOnly,
		Envelope: edi.X12EnvelopeSettings{
			InterchangeSenderID:   "SENDERID",
			InterchangeReceiverID: "RECEIVERID",
			ElementSeparator:      "*",
			SegmentTerminator:     "~",
			ComponentSeparator:    ">",
			RepetitionSeparator:   "^",
		},
	}
	version := &edi.EDITemplateVersion{
		ID:         versionID,
		X12Version: edi.DefaultX12204Version,
		Segments:   editemplates.Base204Segments(tenantInfo, versionID),
	}
	runtime := edix12.RuntimeValues(profile, edi.DefaultX12204Version)
	runtime["isaControlNumber"] = "000000042"
	runtime["groupControlNumber"] = "42"
	runtime["transactionControlNumber"] = "0042"

	documentPayload := edi.NewLoadTenderDocumentPayload(payload)
	result, err := edix12.RenderX12(&edix12.RenderInput{
		Context:         t.Context(),
		Profile:         profile,
		TemplateVersion: version,
		DocumentPayload: documentPayload,
		X12Version:      edi.DefaultX12204Version,
		Runtime:         runtime,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.RawX12)
	return result.RawX12
}

func sampleTenderPayload() edi.LoadTenderPayload {
	weight := int64(42000)
	pieces := int64(12)
	stopWeight := int64(21000)
	windowEnd := int64(1_767_207_600)
	return edi.LoadTenderPayload{
		PurposeCode:       edi.LoadTenderPurposeOriginal,
		ShipmentID:        pulid.MustNew("shp_"),
		BOL:               "BOL-778899",
		Weight:            &weight,
		Pieces:            &pieces,
		ServiceTypeID:     pulid.MustNew("st_"),
		CustomerID:        pulid.MustNew("cust_"),
		FormulaTemplateID: pulid.MustNew("ft_"),
		Moves: []edi.LoadTenderMove{{
			Loaded:   true,
			Sequence: 0,
			Stops: []edi.LoadTenderStop{
				{
					LocationID:           pulid.MustNew("loc_"),
					LocationName:         "Chicago Warehouse",
					LocationAddressLine1: "100 Dock St",
					LocationCity:         "Chicago",
					LocationStateCode:    "IL",
					LocationPostalCode:   "60601",
					Type:                 "Pickup",
					ScheduleType:         "Open",
					Sequence:             1,
					Weight:               &stopWeight,
					Pieces:               &pieces,
					ScheduledWindowStart: 1_767_193_200,
				},
				{
					LocationID:           pulid.MustNew("loc_"),
					LocationName:         "Dallas DC",
					LocationAddressLine1: "500 Freight Ave",
					LocationCity:         "Dallas",
					LocationStateCode:    "TX",
					LocationPostalCode:   "75201",
					Type:                 "Delivery",
					ScheduleType:         "Open",
					Sequence:             2,
					ScheduledWindowStart: 1_767_204_000,
					ScheduledWindowEnd:   &windowEnd,
				},
			},
		}},
		Commodities: []edi.LoadTenderCommodity{{
			CommodityID:          pulid.MustNew("com_"),
			CommodityDescription: "Frozen Poultry",
			Weight:               42000,
			Pieces:               12,
		}},
		RequiredMappingEntityIDs: map[edi.MappingEntityType][]pulid.ID{},
	}
}

func TestParseInterchangeRoundTripsRendered204(t *testing.T) {
	t.Parallel()

	raw := renderBase204(t, sampleTenderPayload())
	interchange, err := parseInterchange(raw)
	require.NoError(t, err)

	require.Equal(t, "000000042", interchange.controlNumber)
	require.Equal(t, "SENDERID", interchange.senderID)
	require.Equal(t, "RECEIVERID", interchange.receiverID)
	require.Len(t, interchange.transactions, 1)

	transaction := interchange.transactions[0]
	require.Equal(t, edi.TransactionSet204, transaction.set)
	require.Equal(t, "0042", transaction.controlNumber)
	require.Equal(t, "42", transaction.groupControlNumber)
	require.NotEmpty(t, transaction.raw)

	payload := transaction.documentPayload()
	require.NotNil(t, payload.LoadTender)
	tender := payload.LoadTender
	require.Equal(t, edi.LoadTenderPurposeOriginal, tender.PurposeCode)
	require.Equal(t, "BOL-778899", tender.BOL)
	require.NotNil(t, tender.Weight)
	require.EqualValues(t, 42000, *tender.Weight)
	require.NotNil(t, tender.Pieces)
	require.EqualValues(t, 12, *tender.Pieces)

	require.Len(t, tender.Moves, 1)
	stops := tender.Moves[0].Stops
	require.Len(t, stops, 2)
	require.Equal(t, "Pickup", stops[0].Type)
	require.EqualValues(t, 1, stops[0].Sequence)
	require.Equal(t, "Chicago Warehouse", stops[0].LocationName)
	require.Equal(t, "Chicago", stops[0].LocationCity)
	require.Equal(t, "IL", stops[0].LocationStateCode)
	require.NotNil(t, stops[0].Weight)
	require.EqualValues(t, 21000, *stops[0].Weight)
	require.Equal(t, "Delivery", stops[1].Type)
	require.EqualValues(t, 2, stops[1].Sequence)
	require.Equal(t, "Dallas DC", stops[1].LocationName)
	require.NotZero(t, stops[0].ScheduledWindowStart)

	require.Len(t, tender.Commodities, 1)
	require.Equal(t, "FROZEN POULTRY", string(tender.Commodities[0].CommodityID))

	require.Equal(
		t,
		pulid.ID(inboundDefaultMappingKey),
		tender.CustomerID,
	)
	require.NotEmpty(t, tender.RequiredMappingEntityIDs[edi.MappingEntityTypeLocation])
	require.NotEmpty(t, tender.RequiredMappingEntityIDs[edi.MappingEntityTypeCustomer])
	require.NotEmpty(t, tender.RequiredMappingEntityIDs[edi.MappingEntityTypeServiceType])
}

func TestParseAcknowledgments997(t *testing.T) {
	t.Parallel()

	raw := "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000101*0*P*>~" +
		"GS*FA*PARTNER*TRENOVA*20260107*1200*101*X*004010~" +
		"ST*997*0001~" +
		"AK1*SM*42~" +
		"AK2*204*0042~" +
		"AK5*A~" +
		"AK9*A*1*1*1~" +
		"SE*6*0001~" +
		"GE*1*101~" +
		"IEA*1*000000101~"

	interchange, err := parseInterchange(raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	transaction := interchange.transactions[0]
	require.Equal(t, edi.TransactionSet997, transaction.set)

	entries := parseAcknowledgments(&transaction)
	require.Len(t, entries, 1)
	entry := entries[0]
	require.Equal(t, "204", entry.originalTransactionSet)
	require.Equal(t, "0042", entry.originalControlNumber)
	require.Equal(t, "A", entry.acknowledgmentCode)
	require.Equal(t, "SM", entry.originalFunctionalGroupID)
	require.Equal(t, "42", entry.originalGroupControl)
	require.EqualValues(t, 1, entry.acceptedCount)

	status, ackErr := acknowledgmentResolution(&entry)
	require.Equal(t, edi.MessageAcknowledgmentStatusAccepted, status)
	require.Empty(t, ackErr)
}

func TestParseAcknowledgmentsRejected999(t *testing.T) {
	t.Parallel()

	raw := "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00501*000000102*0*P*>~" +
		"GS*FA*PARTNER*TRENOVA*20260107*1200*102*X*005010~" +
		"ST*999*0001~" +
		"AK1*SM*43~" +
		"IK2*204*0043~" +
		"IK3*S5*5**8~" +
		"IK5*R*5~" +
		"AK9*R*1*1*0~" +
		"SE*7*0001~" +
		"GE*1*102~" +
		"IEA*1*000000102~"

	interchange, err := parseInterchange(raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	entries := parseAcknowledgments(&interchange.transactions[0])
	require.Len(t, entries, 1)
	require.Equal(t, "R", entries[0].acknowledgmentCode)
	require.Len(t, entries[0].diagnostics, 1)

	status, ackErr := acknowledgmentResolution(&entries[0])
	require.Equal(t, edi.MessageAcknowledgmentStatusRejected, status)
	require.Contains(t, ackErr, "acknowledgment code R")
}

func TestParseTenderResponse990(t *testing.T) {
	t.Parallel()

	raw := "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000103*0*P*>~" +
		"GS*GF*PARTNER*TRENOVA*20260107*1200*103*X*004010~" +
		"ST*990*0001~" +
		"B1*SCAC*SHIP-1001*20260107*A~" +
		"SE*3*0001~" +
		"GE*1*103~" +
		"IEA*1*000000103~"

	interchange, err := parseInterchange(raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	details := parseTenderResponse(&interchange.transactions[0])
	require.Equal(t, "SHIP-1001", details.shipmentRef)
	require.Equal(t, "A", details.reservationCode)
}

func TestParseFreightInvoice210(t *testing.T) {
	t.Parallel()

	raw := "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000105*0*P*>~" +
		"GS*IM*PARTNER*TRENOVA*20260107*1200*105*X*004010~" +
		"ST*210*0001~" +
		"B3*B*INV-4501*SHIP-1001*PP**1250.50**20260105*35*SCAC~" +
		"C3*USD~" +
		"N9*BM*BOL-778899~" +
		"N9*CN*PRO-555~" +
		"G62*86*20260106~" +
		"N1*BT*ACME LOGISTICS~" +
		"N3*100 MAIN ST*SUITE 4~" +
		"N4*CHICAGO*IL*60601*US~" +
		"LX*1~" +
		"L5*1*LINEHAUL~" +
		"L0*1***42000*G~" +
		"L1*1*2.5*PM*1100.00****LHS~" +
		"LX*2~" +
		"L5*2*FUEL SURCHARGE~" +
		"L1*2**FR*150.50****FUE~" +
		"L3*42000*G***1250.50~" +
		"SE*17*0001~" +
		"GE*1*105~" +
		"IEA*1*000000105~"

	interchange, err := parseInterchange(raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	transaction := interchange.transactions[0]
	require.Equal(t, edi.TransactionSet210, transaction.set)

	payload := transaction.documentPayload()
	require.Equal(t, edi.TransactionSet210, payload.TransactionSet)
	require.NotNil(t, payload.FreightInvoice)
	invoice := payload.FreightInvoice

	require.Equal(t, "INV-4501", invoice.InvoiceNumber)
	require.Equal(t, "SHIP-1001", invoice.ReferenceNumbers["shipmentId"])
	require.Equal(t, "PP", invoice.ReferenceNumbers["paymentMethod"])
	require.Equal(t, "SCAC", invoice.ReferenceNumbers["scac"])
	require.Equal(t, "USD", invoice.CurrencyCode)
	require.True(t, invoice.TotalAmount.Valid)
	require.Equal(t, "1250.5", invoice.TotalAmount.Decimal.String())
	require.NotZero(t, invoice.DeliveryDate)
	require.NotZero(t, invoice.InvoiceDate)

	require.Equal(t, "BOL-778899", invoice.BOL)
	require.Equal(t, "PRO-555", invoice.ProNumber)
	require.Equal(t, "BOL-778899", invoice.ReferenceNumbers["BM"])
	require.Equal(t, "PRO-555", invoice.ReferenceNumbers["CN"])

	require.Equal(t, "ACME LOGISTICS", invoice.BillToName)
	require.Equal(t, "100 MAIN ST", invoice.BillToAddressLine1)
	require.Equal(t, "SUITE 4", invoice.BillToAddressLine2)
	require.Equal(t, "CHICAGO", invoice.BillToCity)
	require.Equal(t, "IL", invoice.BillToStateCode)
	require.Equal(t, "60601", invoice.BillToPostalCode)
	require.Equal(t, "US", invoice.BillToCountry)

	require.Len(t, invoice.LineCharges, 2)
	linehaul := invoice.LineCharges[0]
	require.EqualValues(t, 1, linehaul.Sequence)
	require.Equal(t, "LINEHAUL", linehaul.Description)
	require.Equal(t, "LHS", linehaul.Code)
	require.Equal(t, "1100", linehaul.Amount.String())
	require.True(t, linehaul.Rate.Valid)
	require.Equal(t, "2.5", linehaul.Rate.Decimal.String())
	require.NotNil(t, linehaul.Weight)
	require.EqualValues(t, 42000, *linehaul.Weight)
	fuel := invoice.LineCharges[1]
	require.EqualValues(t, 2, fuel.Sequence)
	require.Equal(t, "FUEL SURCHARGE", fuel.Description)
	require.Equal(t, "FUE", fuel.Code)
	require.Equal(t, "150.5", fuel.Amount.String())

	require.NotNil(t, invoice.Weight)
	require.EqualValues(t, 42000, *invoice.Weight)
}

func TestParseFreightInvoice210FallbacksToL3AndL11(t *testing.T) {
	t.Parallel()

	raw := "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000106*0*P*>~" +
		"GS*IM*PARTNER*TRENOVA*20260107*1200*106*X*004010~" +
		"ST*210*0001~" +
		"B3**INV-9001**TP~" +
		"L11*BOL-4455*BM~" +
		"L3*18000*G***875.25~" +
		"SE*5*0001~" +
		"GE*1*106~" +
		"IEA*1*000000106~"

	interchange, err := parseInterchange(raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	invoice := parseFreightInvoice(&interchange.transactions[0])

	require.Equal(t, "INV-9001", invoice.InvoiceNumber)
	require.Equal(t, "BOL-4455", invoice.BOL)
	require.True(t, invoice.TotalAmount.Valid)
	require.Equal(t, "875.25", invoice.TotalAmount.Decimal.String())
	require.Empty(t, invoice.LineCharges)
	require.Empty(t, invoice.CurrencyCode)
}

func renderBase210(t *testing.T, payload edi.FreightInvoicePayload) string {
	t.Helper()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	versionID := pulid.MustNew("editv_")
	profile := &edi.EDIPartnerDocumentProfile{
		TransactionSet: edi.TransactionSet210,
		Direction:      edi.DocumentDirectionOutbound,
		ValidationMode: edi.ValidationModeWarnOnly,
		Envelope: edi.X12EnvelopeSettings{
			InterchangeSenderID:   "SENDERID",
			InterchangeReceiverID: "RECEIVERID",
			ElementSeparator:      "*",
			SegmentTerminator:     "~",
			ComponentSeparator:    ">",
			RepetitionSeparator:   "^",
		},
	}
	version := &edi.EDITemplateVersion{
		ID:         versionID,
		X12Version: edi.DefaultX12204Version,
		Segments:   editemplates.Base210Segments(tenantInfo, versionID),
	}
	runtime := edix12.RuntimeValues(profile, edi.DefaultX12204Version)
	runtime["isaControlNumber"] = "000000043"
	runtime["groupControlNumber"] = "43"
	runtime["transactionControlNumber"] = "0043"

	result, err := edix12.RenderX12(&edix12.RenderInput{
		Context:         t.Context(),
		Profile:         profile,
		TemplateVersion: version,
		DocumentPayload: edi.DocumentPayload{
			TransactionSet: edi.TransactionSet210,
			FreightInvoice: &payload,
		},
		X12Version: edi.DefaultX12204Version,
		Runtime:    runtime,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.RawX12)
	return result.RawX12
}

func TestParseInterchangeRoundTripsRendered210(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	rendered := renderBase210(t, edi.FreightInvoicePayload{
		InvoiceNumber:      "INV-210-77",
		ShipmentID:         shipmentID,
		BOL:                "BOL-210-77",
		BillToName:         "ACME LOGISTICS",
		BillToAddressLine1: "100 MAIN ST",
		BillToCity:         "CHICAGO",
		BillToStateCode:    "IL",
		BillToPostalCode:   "60601",
		CurrencyCode:       "USD",
		TotalAmount: decimal.NullDecimal{
			Decimal: decimal.RequireFromString("1250.50"),
			Valid:   true,
		},
		LineCharges: []edi.FreightInvoiceCharge{
			{Sequence: 1, Description: "LINEHAUL", Amount: decimal.RequireFromString("1100.00")},
			{Sequence: 2, Description: "FUEL", Amount: decimal.RequireFromString("150.50")},
		},
	})

	interchange, err := parseInterchange(rendered)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	transaction := interchange.transactions[0]
	require.Equal(t, edi.TransactionSet210, transaction.set)

	payload := transaction.documentPayload()
	require.NotNil(t, payload.FreightInvoice)
	invoice := payload.FreightInvoice

	require.Equal(t, "INV-210-77", invoice.InvoiceNumber)
	require.Equal(t, shipmentID.String(), invoice.ReferenceNumbers["shipmentId"])
	require.Equal(t, "BOL-210-77", invoice.BOL)
	require.Equal(t, "USD", invoice.CurrencyCode)
	require.True(t, invoice.TotalAmount.Valid)
	require.Equal(t, "1250.5", invoice.TotalAmount.Decimal.String())
	require.Equal(t, "ACME LOGISTICS", invoice.BillToName)
	require.Equal(t, "100 MAIN ST", invoice.BillToAddressLine1)
	require.Equal(t, "CHICAGO", invoice.BillToCity)
	require.Equal(t, "IL", invoice.BillToStateCode)
	require.Equal(t, "60601", invoice.BillToPostalCode)
	require.Len(t, invoice.LineCharges, 2)
	require.Equal(t, "LINEHAUL", invoice.LineCharges[0].Description)
	require.Equal(t, "1100", invoice.LineCharges[0].Amount.String())
	require.Equal(t, "FUEL", invoice.LineCharges[1].Description)
	require.Equal(t, "150.5", invoice.LineCharges[1].Amount.String())
}

func TestParseShipmentStatus214(t *testing.T) {
	t.Parallel()

	raw := "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000104*0*P*>~" +
		"GS*QM*PARTNER*TRENOVA*20260107*1200*104*X*004010~" +
		"ST*214*0001~" +
		"B10*PRO123*SHIP-1001*SCAC~" +
		"AT7*AF*NS***20260107*1315~" +
		"SE*4*0001~" +
		"GE*1*104~" +
		"IEA*1*000000104~"

	interchange, err := parseInterchange(raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	details := parseShipmentStatus(&interchange.transactions[0])
	require.Equal(t, "PRO123", details.referenceID)
	require.Equal(t, "SHIP-1001", details.shipmentRef)
	require.Equal(t, "AF", details.statusCode)
	require.NotZero(t, details.eventAt)
}
