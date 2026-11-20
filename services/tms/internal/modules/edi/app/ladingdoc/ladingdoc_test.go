package ladingdoc_test

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edipayloadbuild"
	"github.com/emoss08/trenova/internal/modules/edi/app/ladingdoc"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func project(t *testing.T, role string) (*lading.Encoded, *lading.Starter) {
	t.Helper()
	starter, err := lading.Default().Starter(context.Background(), &lading.StarterRequest{
		Name:  "acme",
		Label: "ACME",
		Role:  role,
	})
	require.NoError(t, err)
	encoded, err := lading.Encode(
		&lading.Project{Files: starter.Files},
		&lading.Context{Org: lading.Org{Name: "Trenova", SCAC: "TRNV", Zone: "America/Chicago"}},
	)
	require.NoError(t, err)
	return encoded, starter
}

func sample(t *testing.T, starter *lading.Starter, path string) string {
	t.Helper()
	for _, item := range starter.Samples {
		if item.Path == path {
			return item.Content
		}
	}
	t.Fatalf("no sample %s", path)
	return ""
}

func TestAReadTenderBecomesATenderPayload(t *testing.T) {
	t.Parallel()

	encoded, starter := project(t, lading.RoleCarrier)
	result, err := lading.Default().Run(context.Background(), encoded, &lading.RunRequest{
		Map:   "tenderIn",
		Input: lading.X12Input(sample(t, starter, "samples/tender-original.x12")),
		Lookups: []lading.LookupAnswer{
			{Kind: "Customer", Code: "BT-1", Found: &lading.Found{ID: "cus_1", Label: "Shipper"}},
			{Kind: "Location", Code: "DC-1"},
			{Kind: "Location", Code: "ST-7"},
		},
		Rules: true,
	})
	require.NoError(t, err)
	require.Nil(t, result.Error)
	require.Len(t, result.Transactions, 1)
	transaction := result.Transactions[0]
	require.False(t, transaction.Rejected(), "%+v", transaction.Outcomes)

	payload, key, err := ladingdoc.ToLoadTender(
		transaction.Document,
		ladingdoc.UnmappedOf(result.Unmapped),
	)
	require.NoError(t, err)
	assert.Equal(t, "L-1001", key)
	assert.Equal(t, editransfer.LoadTenderPurposeOriginal, payload.PurposeCode)
	assert.Equal(t, "BOL-55", payload.BOL)
	assert.Equal(t, edi.MappingSourceID("BT-1"), payload.CustomerID)
	require.Len(t, payload.Moves, 1)
	require.Len(t, payload.Moves[0].Stops, 2)
	pickup := payload.Moves[0].Stops[0]
	assert.Equal(t, "Pickup", pickup.Type)
	assert.Equal(t, edi.MappingSourceID("DC-1"), pickup.LocationID)
	assert.Equal(t, "DALLAS", pickup.LocationCity)
	assert.NotZero(t, pickup.ScheduledWindowStart)
	assert.Equal(t, "Delivery", payload.Moves[0].Stops[1].Type)
	assert.Contains(
		t,
		payload.RequiredMappingEntityIDs[edi.MappingEntityTypeLocation],
		pickup.LocationID,
	)
	require.True(t, payload.TotalChargeAmount.Valid)
	assert.Equal(t, "1850", payload.TotalChargeAmount.Decimal.String())
	require.NotNil(t, payload.Weight)
	assert.EqualValues(t, 42000, *payload.Weight)
}

func TestAStatusPayloadWritesA214(t *testing.T) {
	t.Parallel()

	encoded, _ := project(t, lading.RoleCarrier)
	stop := int64(2)
	document, err := ladingdoc.FromShipmentStatus(&edipayload.ShipmentStatusPayload{
		ShipperReference: "L-1001",
		ProNumber:        "PRO-9",
		Events: []edipayload.ShipmentStatusEvent{{
			StatusCode:   "X1",
			EventAt:      1_768_055_400,
			StopSequence: &stop,
			City:         "Chicago",
			StateCode:    "IL",
		}},
	}, "L-1001", ladingdoc.Zones{Default: "America/Chicago"}, nil)
	require.NoError(t, err)

	result, err := lading.Default().Run(context.Background(), encoded, &lading.RunRequest{
		Map:   "statusOut",
		Input: lading.DocumentInput(document),
		Rules: true,
	})
	require.NoError(t, err)
	require.Nil(t, result.Error)
	require.NotNil(t, result.Output)
	assert.False(t, result.Output.Blocked(), "%+v", result.Output.Findings)
	assert.Contains(t, result.Output.Text, "B10*PRO-9*L-1001*TRNV~")
	assert.Contains(t, result.Output.Text, "AT7*X1*NS***20260110*0830*CT~")
}

func TestAnInvoicePayloadWritesA210(t *testing.T) {
	t.Parallel()

	encoded, _ := project(t, lading.RoleCarrier)
	document, err := ladingdoc.FromFreightInvoice(&edipayload.FreightInvoicePayload{
		InvoiceNumber: "INV-1",
		InvoiceDate:   1_768_219_200,
		Payment:       "Collect",
		TotalAmount:   decimal.NewNullDecimal(decimal.RequireFromString("1250.5")),
		LineCharges: []edipayload.FreightInvoiceCharge{
			{Sequence: 1, Amount: decimal.RequireFromString("1250.5")},
		},
	}, "L-1001", "America/Chicago", nil)
	require.NoError(t, err)

	result, err := lading.Default().Run(context.Background(), encoded, &lading.RunRequest{
		Map:   "invoiceOut",
		Input: lading.DocumentInput(document),
		Rules: true,
	})
	require.NoError(t, err)
	require.Nil(t, result.Error)
	require.NotNil(t, result.Output)
	assert.False(t, result.Output.Blocked(), "%+v", result.Output.Findings)
	assert.Contains(t, result.Output.Text, "B3**INV-1*L-1001*CC**20260112*125050")
	assert.Contains(t, result.Output.Text, "L3*****125050~")
}

func TestATenderPayloadWritesA204(t *testing.T) {
	t.Parallel()

	encoded, _ := project(t, lading.RoleBroker)
	locationID := pulid.ID("loc_1")
	document, err := ladingdoc.FromLoadTender(&editransfer.LoadTenderPayload{
		BOL: "BOL-1",
		Moves: []editransfer.LoadTenderMove{{Stops: []editransfer.LoadTenderStop{
			{
				LocationID:           locationID,
				LocationName:         "Dock",
				Type:                 "Pickup",
				ScheduledWindowStart: 1_768_053_600,
			},
			{Type: "Delivery", ScheduledWindowStart: 1_768_147_200},
		}}},
	}, "tof_1", ladingdoc.Zones{
		Default:   "America/New_York",
		Locations: map[pulid.ID]string{locationID: "America/Chicago"},
	}, nil)
	require.NoError(t, err)

	result, err := lading.Default().Run(context.Background(), encoded, &lading.RunRequest{
		Map:   "tenderOut",
		Input: lading.DocumentInput(document),
		Rules: true,
	})
	require.NoError(t, err)
	require.Nil(t, result.Error)
	require.NotNil(t, result.Output)
	assert.Contains(t, result.Output.Text, "B2**TRNV**tof_1**PP~")
	assert.Contains(t, result.Output.Text, "G62*37*20260110*U*0800*CT~")
}

func TestAnOutboundDocumentCarriesTheLoadsCustomFields(t *testing.T) {
	t.Parallel()

	custom := map[string]any{"po_number": "PO-559031", "declared_value": 1250.5}
	document, err := ladingdoc.FromShipmentStatus(&edipayload.ShipmentStatusPayload{
		Events: []edipayload.ShipmentStatusEvent{{StatusCode: "X1", EventAt: 1_768_055_400}},
	}, "L-1001", ladingdoc.Zones{Default: "America/Chicago"}, custom)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, sonic.Unmarshal(document, &decoded))
	assert.Equal(
		t,
		map[string]any{"po_number": "PO-559031", "declared_value": 1250.5},
		decoded["custom"],
	)
}

func TestATendersCustomFieldsReachItsPayloadByName(t *testing.T) {
	t.Parallel()

	payload, _, err := ladingdoc.ToLoadTender(
		jsonutils.RawJSON(
			`{"key":"L-1","stops":[],"custom":{"po_number":"PO-559031","declared_value":"1250.50"}}`,
		),
		ladingdoc.Unmapped{},
	)
	require.NoError(t, err)
	assert.Equal(
		t,
		map[string]any{"po_number": "PO-559031", "declared_value": "1250.50"},
		payload.CustomFields,
	)
}

func TestStatusCodesRoundTrip(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ArrivedDelivery", ladingdoc.EventKindOf("x1"))
	assert.Equal(t, "X1", ladingdoc.StatusCodeOf("ArrivedDelivery"))
	assert.Equal(t, ladingdoc.EventKindException, ladingdoc.EventKindOf("ZZ"))
}

func TestATenderDocumentCarriesPaymentEquipmentPurchaseOrderAndHazmat(t *testing.T) {
	t.Parallel()

	tender := &editransfer.LoadTenderPayload{
		BOL:       "BOL-1",
		PONumber:  "PO-7781",
		Payment:   "Collect",
		Equipment: "REEFER",
		Moves: []editransfer.LoadTenderMove{{Stops: []editransfer.LoadTenderStop{
			{Type: "Pickup", ScheduledWindowStart: 1_768_053_600},
			{Type: "Delivery", ScheduledWindowStart: 1_768_147_200},
		}}},
		Commodities: []editransfer.LoadTenderCommodity{
			{CommodityName: "Paint", Hazmat: true},
			{CommodityName: "Paper"},
		},
	}
	document, err := ladingdoc.FromLoadTender(tender, "tof_1", ladingdoc.Zones{}, nil)
	require.NoError(t, err)

	var written ladingdoc.LoadTender
	require.NoError(t, sonic.Unmarshal(document, &written))
	assert.Equal(t, "PO-7781", written.PONumber)
	assert.Equal(t, "Collect", written.Payment)
	assert.Equal(t, "REEFER", written.Equipment)
	require.Len(t, written.Commodities, 2)
	require.NotNil(t, written.Commodities[0].Hazmat)
	assert.True(t, *written.Commodities[0].Hazmat)
	assert.Nil(t, written.Commodities[1].Hazmat)

	read, key, err := ladingdoc.ToLoadTender(document, ladingdoc.Unmapped{})
	require.NoError(t, err)
	assert.Equal(t, "tof_1", key)
	assert.Equal(t, "PO-7781", read.PONumber)
	assert.Equal(t, "Collect", read.Payment)
	assert.Equal(t, "REEFER", read.Equipment)
	require.Len(t, read.Commodities, 2)
	assert.True(t, read.Commodities[0].Hazmat)
	assert.False(t, read.Commodities[1].Hazmat)

	tender.Payment = "PP"
	document, err = ladingdoc.FromLoadTender(tender, "tof_1", ladingdoc.Zones{}, nil)
	require.NoError(t, err)
	require.NoError(t, sonic.Unmarshal(document, &written))
	assert.Empty(t, written.Payment, "only a Lading payment value is written")
}

func TestABrokersTenderWritesItsPaymentAndPurchaseOrder(t *testing.T) {
	t.Parallel()

	encoded, _ := project(t, lading.RoleBroker)
	document, err := ladingdoc.FromLoadTender(&editransfer.LoadTenderPayload{
		BOL:       "BOL-1",
		PONumber:  "PO-7781",
		Payment:   "Collect",
		Equipment: "VAN",
		Moves: []editransfer.LoadTenderMove{{Stops: []editransfer.LoadTenderStop{
			{Type: "Pickup", ScheduledWindowStart: 1_768_053_600},
			{Type: "Delivery", ScheduledWindowStart: 1_768_147_200},
		}}},
	}, "tof_1", ladingdoc.Zones{Default: "America/Chicago"}, nil)
	require.NoError(t, err)

	result, err := lading.Default().Run(context.Background(), encoded, &lading.RunRequest{
		Map:   "tenderOut",
		Input: lading.DocumentInput(document),
		Rules: true,
	})
	require.NoError(t, err)
	require.Nil(t, result.Error)
	require.NotNil(t, result.Output)
	assert.Contains(t, result.Output.Text, "B2**TRNV**tof_1**CC~")
	assert.Contains(t, result.Output.Text, "L11*PO-7781*PO~")
}

func TestAReplacedTenderWithStopTemperaturesKeepsTheRangeEveryStopAllows(t *testing.T) {
	t.Parallel()

	payload, _, err := ladingdoc.ToLoadTender(
		jsonutils.RawJSON(`{"key":"L-1","purpose":"Replace","stops":[`+
			`{"sequence":1,"type":"Pickup","window":{"start":{"at":"2026-01-10T14:00:00Z"}},`+
			`"temperature":{"min":{"value":"28","unit":"°F"},"max":{"value":"38","unit":"°F"}}},`+
			`{"sequence":2,"type":"Delivery","window":{"start":{"at":"2026-01-11T14:00:00Z"}},`+
			`"temperature":{"min":{"value":"32","unit":"°F"},"max":{"value":"40","unit":"°F"}}}]}`),
		ladingdoc.Unmapped{},
	)
	require.NoError(t, err)
	assert.Equal(t, editransfer.LoadTenderPurposeChange, payload.PurposeCode)
	require.NotNil(t, payload.TemperatureMin)
	require.NotNil(t, payload.TemperatureMax)
	assert.Equal(t, int16(32), *payload.TemperatureMin)
	assert.Equal(t, int16(38), *payload.TemperatureMax)
}

func TestAnInvoiceQuotesItsPurchaseOrderAsAPORef(t *testing.T) {
	t.Parallel()

	encoded, _ := project(t, lading.RoleCarrier)
	document, err := ladingdoc.FromFreightInvoice(&edipayload.FreightInvoicePayload{
		InvoiceNumber: "INV-1",
		InvoiceDate:   1_768_219_200,
		PONumber:      " 8978300638 ",
		TotalAmount:   decimal.NewNullDecimal(decimal.RequireFromString("1250.5")),
		LineCharges: []edipayload.FreightInvoiceCharge{
			{Sequence: 1, Amount: decimal.RequireFromString("1250.5")},
		},
	}, "L-1001", "America/Chicago", nil)
	require.NoError(t, err)

	var written ladingdoc.FreightInvoice
	require.NoError(t, sonic.Unmarshal(document, &written))
	assert.Equal(t, map[string]string{edipayload.RefPurchaseOrder: "8978300638"}, written.Refs)

	result, err := lading.Default().Run(context.Background(), encoded, &lading.RunRequest{
		Map:   "invoiceOut",
		Input: lading.DocumentInput(document),
		Rules: true,
	})
	require.NoError(t, err)
	require.Nil(t, result.Error)
	require.NotNil(t, result.Output)
	assert.Contains(t, result.Output.Text, "N9*PO*8978300638~")

	read, err := ladingdoc.ToFreightInvoice(document)
	require.NoError(t, err)
	assert.Equal(t, "8978300638", read.PONumber)
}

func TestAnInvoiceWithoutAPurchaseOrderKeepsItsOtherRefs(t *testing.T) {
	t.Parallel()

	document, err := ladingdoc.FromFreightInvoice(&edipayload.FreightInvoicePayload{
		InvoiceNumber:    "INV-2",
		InvoiceDate:      1_768_219_200,
		TotalAmount:      decimal.NewNullDecimal(decimal.RequireFromString("10")),
		ReferenceNumbers: map[string]string{"BM": "BOL-2", "PO": "PO-FROM-REFS"},
	}, "L-2", "America/Chicago", nil)
	require.NoError(t, err)

	var written ladingdoc.FreightInvoice
	require.NoError(t, sonic.Unmarshal(document, &written))
	assert.Equal(t, map[string]string{"BM": "BOL-2", "PO": "PO-FROM-REFS"}, written.Refs)
}

func TestAnInvoiceSendsItsReferencesUnderTheirX12Qualifiers(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	payload := edipayloadbuild.BuildFreightInvoicePayload(&invoice.Invoice{
		ID:                pulid.MustNew("inv_"),
		Number:            "INV-7",
		InvoiceDate:       1_768_219_200,
		ShipmentID:        shipmentID,
		ShipmentBOL:       "BOL-7",
		ShipmentProNumber: "PRO-7",
		PONumber:          "PO-7",
		TotalAmount:       decimal.RequireFromString("10"),
		Lines: []*invoice.InvoiceLine{
			{LineNumber: 1, Amount: decimal.RequireFromString("10")},
		},
	})
	document, err := ladingdoc.FromFreightInvoice(
		payload.FreightInvoice,
		"L-7",
		"America/Chicago",
		nil,
	)
	require.NoError(t, err)

	var written ladingdoc.FreightInvoice
	require.NoError(t, sonic.Unmarshal(document, &written))
	assert.Equal(t, map[string]string{"BM": "BOL-7", "CN": "PRO-7", "PO": "PO-7"}, written.Refs)

	encoded, _ := project(t, lading.RoleCarrier)
	result, err := lading.Default().Run(context.Background(), encoded, &lading.RunRequest{
		Map:   "invoiceOut",
		Input: lading.DocumentInput(document),
		Rules: true,
	})
	require.NoError(t, err)
	require.Nil(t, result.Error)
	require.NotNil(t, result.Output)
	assert.Contains(t, result.Output.Text, "N9*BM*BOL-7~")
	assert.Contains(t, result.Output.Text, "N9*CN*PRO-7~")
	assert.Contains(t, result.Output.Text, "N9*PO*PO-7~")
	assert.NotContains(t, result.Output.Text, shipmentID.String(),
		"a partner never sees Trenova's record id")
	assert.NotContains(t, result.Output.Text, "N9*qualifier")
}

func TestAStatusSendsOnlyX12QualifiedReferences(t *testing.T) {
	t.Parallel()

	source := &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		BOL:       "BOL-8",
		ProNumber: "PRO-8",
	}
	event := &shipmentevent.Event{
		ID:         pulid.MustNew("sev_"),
		ShipmentID: source.ID,
		Type:       shipmentevent.TypeStopArrived,
		Summary:    "Arrived at the dock",
		OccurredAt: 1_768_055_400,
	}
	payload := edipayloadbuild.BuildShipmentEventStatusPayload(event, source).ShipmentStatus
	assert.Equal(t, event.ID, payload.EventID)

	document, err := ladingdoc.FromShipmentStatus(
		payload,
		"L-8",
		ladingdoc.Zones{Default: "America/Chicago"},
		nil,
	)
	require.NoError(t, err)

	var written ladingdoc.ShipmentStatus
	require.NoError(t, sonic.Unmarshal(document, &written))
	assert.Equal(t, map[string]string{"BM": "BOL-8", "CN": "PRO-8"}, written.Refs)
	assert.NotContains(t, string(document), source.ID.String())
	assert.NotContains(t, string(document), event.ID.String())
}

func TestAnInboundDocumentsKeyStaysOutOfItsReferences(t *testing.T) {
	t.Parallel()

	invoicePayload, err := ladingdoc.ToFreightInvoice(jsonutils.RawJSON(
		`{"number":"INV-9","date":"2026-01-12","key":"L-9","total":{"amount":"10.00"},` +
			`"refs":{"BM":"BOL-9","CN":"PRO-9"}}`,
	))
	require.NoError(t, err)
	assert.Equal(t, "L-9", invoicePayload.Key)
	assert.Equal(
		t,
		map[string]string{"BM": "BOL-9", "CN": "PRO-9"},
		invoicePayload.ReferenceNumbers,
	)

	statusPayload, err := ladingdoc.ToShipmentStatus(jsonutils.RawJSON(
		`{"key":"L-9","refs":{"BM":"BOL-9"},"events":[` +
			`{"kind":"Delivered","at":{"at":"2026-01-12T14:00:00Z"}}]}`,
	))
	require.NoError(t, err)
	assert.Equal(t, "L-9", statusPayload.Key)
	assert.Equal(t, map[string]string{"BM": "BOL-9"}, statusPayload.References)
}
