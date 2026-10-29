package triageshipmentservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageworkports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/triagelookupports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var testTenant = pagination.TenantInfo{
	OrgID:  pulid.MustNew("org_"),
	BuID:   pulid.MustNew("bu_"),
	UserID: pulid.MustNew("usr_"),
}

type fakeLookup struct {
	triagelookupports.TriageLookupRepository

	customers    []triagelookupports.TriageRecordMatch
	locations    map[string][]triagelookupports.TriageLocationMatch
	trailers     []triagelookupports.TriageRecordMatch
	commodities  []triagelookupports.TriageRecordMatch
	defaults     *triagelookupports.TriageShipmentDefaults
	serviceTypes []pulid.ID
}

func (f *fakeLookup) FindCustomers(
	context.Context,
	*triagelookupports.TriageLookupQuery,
) ([]triagelookupports.TriageRecordMatch, error) {
	return f.customers, nil
}

func (f *fakeLookup) FindLocations(
	_ context.Context,
	req *triagelookupports.TriageLocationQuery,
) ([]triagelookupports.TriageLocationMatch, error) {
	return f.locations[req.City], nil
}

func (f *fakeLookup) FindTrailerTypes(
	context.Context,
	*triagelookupports.TriageLookupQuery,
) ([]triagelookupports.TriageRecordMatch, error) {
	return f.trailers, nil
}

func (f *fakeLookup) FindCommodities(
	context.Context,
	*triagelookupports.TriageLookupQuery,
) ([]triagelookupports.TriageRecordMatch, error) {
	return f.commodities, nil
}

func (f *fakeLookup) CustomerDefaults(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*triagelookupports.TriageShipmentDefaults, error) {
	return f.defaults, nil
}

func (f *fakeLookup) ActiveServiceTypes(
	context.Context,
	pagination.TenantInfo,
	int,
) ([]pulid.ID, error) {
	return f.serviceTypes, nil
}

type fakeCreator struct {
	created []*shipment.Shipment
}

func (f *fakeCreator) Create(
	_ context.Context,
	entity *shipment.Shipment,
	_ *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	entity.ID = pulid.MustNew("shp_")
	entity.ProNumber = "S-300001"
	f.created = append(f.created, entity)

	return entity, nil
}

type fakeComments struct {
	comments []*shipmentcommentports.CreateSystemShipmentCommentRequest
	err      error
}

func (f *fakeComments) CreateSystem(
	_ context.Context,
	req *shipmentcommentports.CreateSystemShipmentCommentRequest,
) (*shipment.ShipmentComment, error) {
	f.comments = append(f.comments, req)

	return &shipment.ShipmentComment{}, f.err
}

func newLoadService() (*Service, *fakeLookup, *fakeCreator, *fakeComments) {
	customerID := pulid.MustNew("cus_")
	lookup := &fakeLookup{
		customers: []triagelookupports.TriageRecordMatch{
			{ID: customerID, Label: "Crestline Retail", Exact: true},
		},
		locations: map[string][]triagelookupports.TriageLocationMatch{
			"Columbus": {
				{
					ID:       pulid.MustNew("loc_"),
					Label:    "Columbus DC",
					Timezone: "America/New_York",
					Exact:    true,
				},
			},
			"Louisville": {
				{
					ID:       pulid.MustNew("loc_"),
					Label:    "Louisville Store",
					Timezone: "America/New_York",
				},
			},
		},
		defaults: &triagelookupports.TriageShipmentDefaults{
			ServiceTypeID:  pulid.MustNew("st_"),
			ShipmentTypeID: pulid.MustNew("sht_"),
		},
	}
	creator := &fakeCreator{}
	comments := &fakeComments{}

	return &Service{
		l:        zap.NewNop(),
		lookup:   lookup,
		creator:  creator,
		comments: comments,
		now:      func() int64 { return 1_791_000_000 },
	}, lookup, creator, comments
}

func loadRequest(
	fields ...triageworkports.TriageLoadFieldValue,
) *triageworkports.CreateTriageLoadRequest {
	for i := range fields {
		fields[i].Index = i
	}

	return &triageworkports.CreateTriageLoadRequest{
		TenantInfo:     testTenant,
		Actor:          permissionports.UserActor(testTenant),
		Fields:         fields,
		IdempotencyKey: "load-key",
	}
}

func field(key inboundmessage.LoadFieldKey, value string) triageworkports.TriageLoadFieldValue {
	return triageworkports.TriageLoadFieldValue{Key: key, Value: value}
}

func multiFields(err error) []string {
	multi, ok := errors.AsType[*errortypes.MultiError](err)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(multi.Errors))
	for _, item := range multi.Errors {
		out = append(out, item.Field)
	}

	return out
}

func TestCreateLoadBuildsShipment(t *testing.T) {
	t.Parallel()

	service, lookup, creator, comments := newLoadService()

	created, err := service.CreateLoad(t.Context(), loadRequest(
		field(inboundmessage.LoadFieldCustomer, "Crestline"),
		field(inboundmessage.LoadFieldOrigin, "Columbus, OH"),
		field(inboundmessage.LoadFieldDestination, "Louisville Store · Louisville, KY"),
		field(inboundmessage.LoadFieldPickup, "Oct 7, 08:00-14:00"),
		field(inboundmessage.LoadFieldDelivery, "Oct 8, 09:00"),
		field(inboundmessage.LoadFieldWeight, "42,000 lb"),
		field(inboundmessage.LoadFieldRate, "$1,240.00 all in"),
		field(inboundmessage.LoadFieldReference, "PO 88240"),
		field(inboundmessage.LoadFieldInstructions, "Call ahead"),
	))
	require.NoError(t, err)

	assert.Equal(t, "S-300001", created.ProNumber)
	require.Len(t, creator.created, 1)
	entity := creator.created[0]
	assert.Equal(t, lookup.customers[0].ID, entity.CustomerID)
	assert.Equal(t, lookup.defaults.ServiceTypeID, entity.ServiceTypeID)
	assert.Equal(t, "load-key", entity.IdempotencyKey)
	assert.Equal(t, "PO 88240", entity.ExternalReference)
	require.NotNil(t, entity.Weight)
	assert.Equal(t, int64(42000), *entity.Weight)
	assert.Equal(t, "1240", entity.FreightChargeAmount.Decimal.String())
	assert.True(t, entity.FormulaTemplateID.IsNil(), "a tendered rate is a manual rate")
	require.Len(t, entity.Moves, 1)
	stops := entity.Moves[0].Stops
	require.Len(t, stops, 2)
	assert.Equal(t, shipment.StopTypePickup, stops[0].Type)
	require.NotNil(t, stops[0].ScheduledWindowEnd)
	assert.Equal(t, int64(6*60*60), *stops[0].ScheduledWindowEnd-stops[0].ScheduledWindowStart)
	assert.Nil(t, stops[1].ScheduledWindowEnd)
	require.Len(t, comments.comments, 1)
	assert.Equal(t, "Call ahead", comments.comments[0].Comment)
}

func TestCreateLoadReportsEachField(t *testing.T) {
	t.Parallel()

	service, lookup, creator, _ := newLoadService()
	lookup.customers = nil
	lookup.trailers = []triagelookupports.TriageRecordMatch{
		{ID: pulid.MustNew("tt_"), Label: "53' Dry Van"},
		{ID: pulid.MustNew("tt_"), Label: "53' Reefer"},
	}

	_, err := service.CreateLoad(t.Context(), loadRequest(
		field(inboundmessage.LoadFieldCustomer, "Nobody Inc"),
		field(inboundmessage.LoadFieldOrigin, "Atlantis, ZZ"),
		field(inboundmessage.LoadFieldPickup, "whenever"),
		field(inboundmessage.LoadFieldEquipment, "53'"),
		field(inboundmessage.LoadFieldWeight, "heavy"),
		field(inboundmessage.LoadFieldRate, "call me"),
		field(inboundmessage.LoadFieldReference, strings.Repeat("r", maxReferenceRunes+1)),
	))
	require.Error(t, err)

	got := multiFields(err)
	for _, want := range []string{
		"fields[0].value",
		"fields[1].value",
		"fields[2].value",
		"fields[3].value",
		"fields[4].value",
		"fields[5].value",
		"fields[6].value",
		"fields",
	} {
		assert.Contains(t, got, want)
	}
	assert.Empty(t, creator.created)
}

func TestCreateLoadUsesMatchedCustomerAndSoleServiceType(t *testing.T) {
	t.Parallel()

	service, lookup, creator, _ := newLoadService()
	lookup.defaults = nil
	lookup.serviceTypes = []pulid.ID{pulid.MustNew("st_")}
	req := loadRequest(
		field(inboundmessage.LoadFieldOrigin, "Columbus, OH"),
		field(inboundmessage.LoadFieldDestination, "Louisville, KY"),
		field(inboundmessage.LoadFieldPickup, "Oct 7, 08:00"),
		field(inboundmessage.LoadFieldDelivery, "Oct 8, 09:00"),
	)
	req.CustomerID = pulid.MustNew("cus_")

	_, err := service.CreateLoad(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, req.CustomerID, creator.created[0].CustomerID)
	assert.Equal(t, lookup.serviceTypes[0], creator.created[0].ServiceTypeID)

	lookup.serviceTypes = append(lookup.serviceTypes, pulid.MustNew("st_"))
	_, err = service.CreateLoad(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, multiFields(err), "fields")
}

func TestPickRecord(t *testing.T) {
	t.Parallel()

	a := triagelookupports.TriageRecordMatch{ID: pulid.MustNew("cus_"), Label: "A", Exact: true}
	b := triagelookupports.TriageRecordMatch{ID: pulid.MustNew("cus_"), Label: "B"}
	c := triagelookupports.TriageRecordMatch{ID: pulid.MustNew("cus_"), Label: "C", Exact: true}

	_, ok := pickRecord(nil)
	assert.False(t, ok)
	got, ok := pickRecord([]triagelookupports.TriageRecordMatch{a, b})
	assert.True(t, ok)
	assert.Equal(t, a.ID, got.ID)
	got, ok = pickRecord([]triagelookupports.TriageRecordMatch{b})
	assert.True(t, ok)
	assert.Equal(t, b.ID, got.ID)
	_, ok = pickRecord([]triagelookupports.TriageRecordMatch{a, c})
	assert.False(t, ok)
	_, ok = pickRecord([]triagelookupports.TriageRecordMatch{b, b})
	assert.False(t, ok)
}

func TestSplitPlace(t *testing.T) {
	t.Parallel()

	cases := []struct {
		text, name, city, state string
	}{
		{text: "Columbus, OH", city: "Columbus", state: "OH"},
		{text: "Columbus DC · Columbus, OH", name: "Columbus DC", city: "Columbus", state: "OH"},
		{text: "Columbus DC, Columbus, OH", name: "Columbus DC", city: "Columbus", state: "OH"},
		{text: "Columbus DC", name: "Columbus DC"},
	}
	for _, tc := range cases {
		name, city, state := splitPlace(tc.text)
		assert.Equal(t, tc.name, name, tc.text)
		assert.Equal(t, tc.city, city, tc.text)
		assert.Equal(t, tc.state, state, tc.text)
	}
}

func TestParsePounds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		text string
		want int64
		ok   bool
	}{
		{text: "42,000 lb", want: 42000, ok: true},
		{text: "42000", want: 42000, ok: true},
		{text: "1000 kg", want: 2205, ok: true},
		{text: "0 lb", ok: false},
		{text: "heavy", ok: false},
		{text: "99,000,000 lb", ok: false},
	}
	for _, tc := range cases {
		got, ok := parsePounds(tc.text)
		assert.Equal(t, tc.ok, ok, tc.text)
		assert.Equal(t, tc.want, got, tc.text)
	}
}
