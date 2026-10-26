package ladingdoc_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/app/ladingdoc"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/lading"
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
	assert.Equal(t, edi.LoadTenderPurposeOriginal, payload.PurposeCode)
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
	assert.Contains(t, payload.RequiredMappingEntityIDs[edi.MappingEntityTypeLocation], pickup.LocationID)
	require.True(t, payload.TotalChargeAmount.Valid)
	assert.Equal(t, "1850", payload.TotalChargeAmount.Decimal.String())
	require.NotNil(t, payload.Weight)
	assert.EqualValues(t, 42000, *payload.Weight)
}

func TestAStatusPayloadWritesA214(t *testing.T) {
	t.Parallel()

	encoded, _ := project(t, lading.RoleCarrier)
	stop := int64(2)
	document, err := ladingdoc.FromShipmentStatus(&edi.ShipmentStatusPayload{
		ShipperReference: "L-1001",
		ProNumber:        "PRO-9",
		Events: []edi.ShipmentStatusEvent{{
			StatusCode:   "X1",
			EventAt:      1_768_055_400,
			StopSequence: &stop,
			City:         "Chicago",
			StateCode:    "IL",
		}},
	}, "L-1001", ladingdoc.Zones{Default: "America/Chicago"})
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
	document, err := ladingdoc.FromFreightInvoice(&edi.FreightInvoicePayload{
		InvoiceNumber: "INV-1",
		InvoiceDate:   1_768_219_200,
		TotalAmount:   decimal.NewNullDecimal(decimal.RequireFromString("1250.5")),
		LineCharges: []edi.FreightInvoiceCharge{
			{Sequence: 1, Amount: decimal.RequireFromString("1250.5")},
		},
	}, "L-1001", "America/Chicago")
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
	assert.Contains(t, result.Output.Text, "B3**INV-1*L-1001*PP**20260112*125050")
	assert.Contains(t, result.Output.Text, "L3*****125050~")
}

func TestATenderPayloadWritesA204(t *testing.T) {
	t.Parallel()

	encoded, _ := project(t, lading.RoleBroker)
	locationID := pulid.ID("loc_1")
	document, err := ladingdoc.FromLoadTender(&edi.LoadTenderPayload{
		BOL: "BOL-1",
		Moves: []edi.LoadTenderMove{{Stops: []edi.LoadTenderStop{
			{LocationID: locationID, LocationName: "Dock", Type: "Pickup", ScheduledWindowStart: 1_768_053_600},
			{Type: "Delivery", ScheduledWindowStart: 1_768_147_200},
		}}},
	}, "tof_1", ladingdoc.Zones{
		Default:   "America/New_York",
		Locations: map[pulid.ID]string{locationID: "America/Chicago"},
	})
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

func TestStatusCodesRoundTrip(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ArrivedDelivery", ladingdoc.EventKindOf("x1"))
	assert.Equal(t, "X1", ladingdoc.StatusCodeOf("ArrivedDelivery"))
	assert.Equal(t, ladingdoc.EventKindException, ladingdoc.EventKindOf("ZZ"))
}
