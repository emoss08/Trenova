package webhooks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const geofenceEntryEnvelope = `{
	"eventId": "evt-123",
	"eventTime": "2026-01-20T06:39:05.683Z",
	"eventType": "GeofenceEntry",
	"orgId": 20936,
	"webhookId": "1411751028848270",
	"data": {
		"address": {
			"id": "addr-1",
			"name": "Main Warehouse",
			"formattedAddress": "350 Rhode Island St, San Francisco, CA 94103",
			"externalIds": {"siteId": "site-9"},
			"geofence": {
				"circle": {
					"latitude": 37.765363,
					"longitude": -122.403098,
					"radiusMeters": 25
				}
			}
		},
		"vehicle": {
			"id": "veh-1",
			"name": "Truck 42",
			"assetType": "vehicle",
			"licensePlate": "ABC1234",
			"vin": "1FUJGLDR2LLLVXXXX",
			"externalIds": {"fleetId": "f-1"}
		},
		"driver": {
			"id": "drv-1",
			"name": "Alex Driver",
			"externalIds": {"payrollId": "p-7"}
		}
	}
}`

func TestParseEvent(t *testing.T) {
	t.Parallel()

	event, err := ParseEvent([]byte(geofenceEntryEnvelope))
	require.NoError(t, err)

	assert.Equal(t, "evt-123", event.EventID)
	assert.Equal(t, EventTypeGeofenceEntry, event.EventType)
	assert.Equal(t, int64(20936), event.OrgID)
	assert.Equal(t, "1411751028848270", event.WebhookID)
	assert.Equal(
		t,
		time.Date(2026, 1, 20, 6, 39, 5, 683000000, time.UTC),
		event.EventTime.UTC(),
	)
	assert.NotEmpty(t, event.Data)
}

func TestParseEventInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "missing event id", body: `{"eventType":"GeofenceEntry"}`},
		{name: "missing event type", body: `{"eventId":"evt-1"}`},
		{name: "empty object", body: `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseEvent([]byte(tt.body))
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrEventInvalid)
		})
	}
}

func TestParseEventMalformedJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseEvent([]byte(`{"eventId":`))
	require.Error(t, err)
}

func TestGeofenceData(t *testing.T) {
	t.Parallel()

	event, err := ParseEvent([]byte(geofenceEntryEnvelope))
	require.NoError(t, err)

	data, err := event.GeofenceData()
	require.NoError(t, err)

	require.NotNil(t, data.Address)
	assert.Equal(t, "addr-1", data.Address.ID)
	assert.Equal(t, "Main Warehouse", data.Address.Name)
	assert.Equal(
		t,
		"350 Rhode Island St, San Francisco, CA 94103",
		data.Address.FormattedAddress,
	)
	assert.Equal(t, map[string]string{"siteId": "site-9"}, data.Address.ExternalIDs)
	require.NotNil(t, data.Address.Geofence)
	require.NotNil(t, data.Address.Geofence.Circle)
	assert.InDelta(t, 37.765363, data.Address.Geofence.Circle.Latitude, 1e-9)
	assert.InDelta(t, -122.403098, data.Address.Geofence.Circle.Longitude, 1e-9)
	assert.InDelta(t, 25.0, data.Address.Geofence.Circle.RadiusMeters, 1e-9)

	require.NotNil(t, data.Vehicle)
	assert.Equal(t, "veh-1", data.Vehicle.ID)
	assert.Equal(t, "Truck 42", data.Vehicle.Name)
	assert.Equal(t, "ABC1234", data.Vehicle.LicensePlate)

	require.NotNil(t, data.Driver)
	assert.Equal(t, "drv-1", data.Driver.ID)
	assert.Equal(t, "Alex Driver", data.Driver.Name)
}

func TestGeofenceDataPolygon(t *testing.T) {
	t.Parallel()

	event := Event{
		EventID:   "evt-1",
		EventType: EventTypeGeofenceExit,
		Data: RawData(`{
			"address": {
				"id": "addr-2",
				"name": "Yard",
				"formattedAddress": "1 Yard Way",
				"geofence": {
					"polygon": {
						"vertices": [
							{"latitude": 1.5, "longitude": 2.5},
							{"latitude": 3.5, "longitude": 4.5}
						]
					}
				}
			}
		}`),
	}

	data, err := event.GeofenceData()
	require.NoError(t, err)
	require.NotNil(t, data.Address)
	require.NotNil(t, data.Address.Geofence)
	require.NotNil(t, data.Address.Geofence.Polygon)
	require.Len(t, data.Address.Geofence.Polygon.Vertices, 2)
	assert.InDelta(t, 3.5, data.Address.Geofence.Polygon.Vertices[1].Latitude, 1e-9)
	assert.InDelta(t, 4.5, data.Address.Geofence.Polygon.Vertices[1].Longitude, 1e-9)
}

func TestGeofenceDataTypeMismatch(t *testing.T) {
	t.Parallel()

	event := Event{EventID: "evt-1", EventType: EventTypeVehicleCreated}

	_, err := event.GeofenceData()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventTypeMismatch)
}

func TestVehicleDataNested(t *testing.T) {
	t.Parallel()

	event := Event{
		EventID:   "evt-1",
		EventType: EventTypeVehicleCreated,
		Data: RawData(
			`{"vehicle":{"id":"veh-1","name":"Truck 42","externalIds":{"fleetId":"f-1"}}}`,
		),
	}

	data, err := event.VehicleData()
	require.NoError(t, err)
	assert.Equal(t, "veh-1", data.ID)
	assert.Equal(t, "Truck 42", data.Name)
	assert.Equal(t, map[string]string{"fleetId": "f-1"}, data.ExternalIDs)
}

func TestVehicleDataFlat(t *testing.T) {
	t.Parallel()

	event := Event{
		EventID:   "evt-1",
		EventType: EventTypeVehicleUpdated,
		Data:      RawData(`{"id":"veh-2","name":"Truck 7"}`),
	}

	data, err := event.VehicleData()
	require.NoError(t, err)
	assert.Equal(t, "veh-2", data.ID)
	assert.Equal(t, "Truck 7", data.Name)
}

func TestVehicleDataTypeMismatch(t *testing.T) {
	t.Parallel()

	event := Event{EventID: "evt-1", EventType: EventTypeDriverCreated}

	_, err := event.VehicleData()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventTypeMismatch)
}

func TestDriverDataNested(t *testing.T) {
	t.Parallel()

	event := Event{
		EventID:   "evt-1",
		EventType: EventTypeDriverCreated,
		Data: RawData(
			`{"driver":{"id":"drv-1","name":"Alex Driver","externalIds":{"payrollId":"p-7"}}}`,
		),
	}

	data, err := event.DriverData()
	require.NoError(t, err)
	assert.Equal(t, "drv-1", data.ID)
	assert.Equal(t, "Alex Driver", data.Name)
	assert.Equal(t, map[string]string{"payrollId": "p-7"}, data.ExternalIDs)
}

func TestDriverDataFlat(t *testing.T) {
	t.Parallel()

	event := Event{
		EventID:   "evt-1",
		EventType: EventTypeDriverUpdated,
		Data:      RawData(`{"id":"drv-2","name":"Sam Driver"}`),
	}

	data, err := event.DriverData()
	require.NoError(t, err)
	assert.Equal(t, "drv-2", data.ID)
	assert.Equal(t, "Sam Driver", data.Name)
}

func TestDriverDataTypeMismatch(t *testing.T) {
	t.Parallel()

	event := Event{EventID: "evt-1", EventType: EventTypeGeofenceEntry}

	_, err := event.DriverData()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventTypeMismatch)
}

func TestEmptyDataDecodes(t *testing.T) {
	t.Parallel()

	geofence := Event{EventID: "evt-1", EventType: EventTypeGeofenceEntry}
	geoData, err := geofence.GeofenceData()
	require.NoError(t, err)
	assert.Nil(t, geoData.Address)

	vehicle := Event{EventID: "evt-2", EventType: EventTypeVehicleCreated}
	vehData, err := vehicle.VehicleData()
	require.NoError(t, err)
	assert.Empty(t, vehData.ID)
}

const routeStopArrivalEnvelope = `{
	"eventId": "evt-rs-1",
	"eventTime": "2026-02-01T10:00:00Z",
	"eventType": "RouteStopArrival",
	"orgId": 20936,
	"webhookId": "1411751028848270",
	"data": {
		"operation": "arrived",
		"time": "2026-02-01T09:59:30Z",
		"assignedToRoute": "vehicle",
		"driver": {
			"id": "drv-1",
			"name": "Alex Driver",
			"externalIds": {"payrollId": "p-7"}
		},
		"vehicle": {
			"id": "veh-1",
			"name": "Truck 42",
			"vin": "1FUJGLDR2LLLVXXXX",
			"licensePlate": "ABC1234",
			"externalIds": {"fleetId": "f-1"}
		},
		"route": {
			"id": "route-1",
			"name": "Morning Run",
			"externalIds": {"routeRef": "r-9"}
		},
		"routeStopDetails": {
			"id": "stop-1",
			"state": "arrived",
			"eta": "2026-02-01T10:00:00Z",
			"enRouteTime": "2026-02-01T09:30:00Z",
			"actualArrivalTime": "2026-02-01T09:59:30Z",
			"externalIds": {"shipmentId": "ship-9", "stopSequence": "1"},
			"orders": [
				{"id": "order-1", "name": "PO-1001", "externalIds": {"orderRef": "o-1"}}
			]
		}
	}
}`

func TestRouteStopDataArrival(t *testing.T) {
	t.Parallel()

	event, err := ParseEvent([]byte(routeStopArrivalEnvelope))
	require.NoError(t, err)

	data, err := event.RouteStopData()
	require.NoError(t, err)

	assert.Equal(t, "arrived", data.Operation)
	assert.Equal(t, "2026-02-01T09:59:30Z", data.Time)
	assert.Equal(t, "vehicle", data.AssignedToRoute)

	require.NotNil(t, data.Driver)
	assert.Equal(t, "drv-1", data.Driver.ID)
	assert.Equal(t, map[string]string{"payrollId": "p-7"}, data.Driver.ExternalIDs)

	require.NotNil(t, data.Vehicle)
	assert.Equal(t, "veh-1", data.Vehicle.ID)
	assert.Equal(t, "1FUJGLDR2LLLVXXXX", data.Vehicle.Vin)
	assert.Equal(t, "ABC1234", data.Vehicle.LicensePlate)

	require.NotNil(t, data.Route)
	assert.Equal(t, "route-1", data.Route.ID)
	assert.Equal(t, "Morning Run", data.Route.Name)

	require.NotNil(t, data.RouteStop)
	assert.Equal(t, "stop-1", data.RouteStop.ID)
	assert.Equal(t, "arrived", data.RouteStop.State)
	assert.Equal(t, "2026-02-01T09:59:30Z", data.RouteStop.ActualArrivalTime)
	assert.Equal(t, "2026-02-01T09:30:00Z", data.RouteStop.EnRouteTime)
	assert.Equal(t, "ship-9", data.RouteStop.ExternalIDs["shipmentId"])
	require.Len(t, data.RouteStop.Orders, 1)
	assert.Equal(t, "order-1", data.RouteStop.Orders[0].ID)
	assert.Equal(t, "PO-1001", data.RouteStop.Orders[0].Name)
}

func TestRouteStopDataDeparture(t *testing.T) {
	t.Parallel()

	event := Event{
		EventID:   "evt-rs-2",
		EventType: EventTypeRouteStopDeparture,
		Data: RawData(`{
			"operation": "departed",
			"time": "2026-02-01T10:20:00Z",
			"assignedToRoute": "driver",
			"routeStopDetails": {
				"id": "stop-1",
				"state": "departed",
				"actualArrivalTime": "2026-02-01T09:59:30Z",
				"actualDepartureTime": "2026-02-01T10:20:00Z"
			}
		}`),
	}

	data, err := event.RouteStopData()
	require.NoError(t, err)
	assert.Equal(t, "departed", data.Operation)
	assert.Equal(t, "driver", data.AssignedToRoute)
	require.NotNil(t, data.RouteStop)
	assert.Equal(t, "departed", data.RouteStop.State)
	assert.Equal(t, "2026-02-01T10:20:00Z", data.RouteStop.ActualDepartureTime)
	assert.Nil(t, data.Driver)
	assert.Nil(t, data.Vehicle)
}

func TestRouteStopDataTypeMismatch(t *testing.T) {
	t.Parallel()

	event := Event{EventID: "evt-1", EventType: EventTypeFormSubmitted}

	_, err := event.RouteStopData()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventTypeMismatch)
}

func TestRouteStopDataEmpty(t *testing.T) {
	t.Parallel()

	event := Event{EventID: "evt-1", EventType: EventTypeRouteStopEtaUpdated}
	data, err := event.RouteStopData()
	require.NoError(t, err)
	assert.Nil(t, data.RouteStop)
}

const formSubmittedEnvelope = `{
	"eventId": "evt-form-1",
	"eventTime": "2026-02-02T08:00:00Z",
	"eventType": "FormSubmitted",
	"orgId": 20936,
	"webhookId": "1411751028848270",
	"data": {
		"form": {
			"id": "form-sub-1",
			"status": "completed",
			"submittedAtTime": "2026-02-02T07:59:00Z",
			"routeStopId": "stop-1",
			"submittedBy": {"id": "drv-1", "type": "driver"},
			"externalIds": {"shipmentId": "ship-9"},
			"location": {"latitude": 37.7, "longitude": -122.4},
			"formTemplate": {"id": "tmpl-1", "revisionId": "rev-1"},
			"fields": [
				{"id": "f1", "label": "Gallons", "type": "number", "numberValue": {"value": 42.5}},
				{"id": "f2", "label": "Notes", "type": "text", "textValue": {"value": "all good"}},
				{"id": "f3", "label": "Condition", "type": "multiple_choice", "multipleChoiceValue": {"value": "Good", "valueId": "opt-1"}},
				{"id": "f4", "label": "Checks", "type": "check_boxes", "checkBoxesValue": {"value": ["Front", "Rear"]}},
				{"id": "f5", "label": "Sign", "type": "signature", "signatureValue": {"media": {"url": "https://media/sig.png"}}}
			]
		}
	}
}`

func TestFormDataSubmitted(t *testing.T) {
	t.Parallel()

	event, err := ParseEvent([]byte(formSubmittedEnvelope))
	require.NoError(t, err)

	data, err := event.FormData()
	require.NoError(t, err)

	assert.Equal(t, "form-sub-1", data.FormID)
	assert.Equal(t, "tmpl-1", data.TemplateID)
	assert.Equal(t, "rev-1", data.TemplateRevisionID)
	assert.Equal(t, "completed", data.Status)
	assert.Equal(t, "2026-02-02T07:59:00Z", data.SubmittedAtTime)
	assert.Equal(t, "stop-1", data.AssignedToRouteStopID)
	require.NotNil(t, data.SubmittedBy)
	assert.Equal(t, "drv-1", data.SubmittedBy.ID)
	assert.Equal(t, "driver", data.SubmittedBy.Type)
	assert.Equal(t, map[string]string{"shipmentId": "ship-9"}, data.ExternalIDs)
	require.NotNil(t, data.Location)
	assert.InDelta(t, 37.7, data.Location.Latitude, 1e-9)

	require.Len(t, data.Fields, 5)
	assert.Equal(t, "number", data.Fields[0].Type)
	assert.Equal(t, "42.5", data.Fields[0].Value)
	assert.Equal(t, "all good", data.Fields[1].Value)
	assert.Equal(t, "Good", data.Fields[2].Value)
	assert.Equal(t, "Front, Rear", data.Fields[3].Value)
	assert.Equal(t, "https://media/sig.png", data.Fields[4].Value)
}

func TestFormDataDocumentSubmitted(t *testing.T) {
	t.Parallel()

	event := Event{
		EventID:   "evt-doc-1",
		EventType: EventTypeDocumentSubmitted,
		Data: RawData(`{
			"document": {
				"id": "doc-1",
				"templateId": "dtmpl-1",
				"submittedAtTime": "2026-02-03T08:00:00Z",
				"fields": [
					{"id": "d1", "label": "BOL", "type": "text", "value": {"stringValue": "BOL-123"}},
					{"id": "d2", "label": "Weight", "type": "number", "value": {"numberValue": 1500}},
					{"id": "d3", "label": "Photo", "type": "photo", "value": {"photoValue": [{"url": "https://media/p.jpg"}]}}
				]
			}
		}`),
	}

	data, err := event.FormData()
	require.NoError(t, err)
	assert.Equal(t, "doc-1", data.FormID)
	assert.Equal(t, "dtmpl-1", data.TemplateID)
	require.Len(t, data.Fields, 3)
	assert.Equal(t, "BOL-123", data.Fields[0].Value)
	assert.Equal(t, "1500", data.Fields[1].Value)
	assert.Equal(t, "https://media/p.jpg", data.Fields[2].Value)
}

func TestFormDataUnwrapped(t *testing.T) {
	t.Parallel()

	event := Event{
		EventID:   "evt-form-3",
		EventType: EventTypeFormUpdated,
		Data: RawData(`{
			"id": "form-sub-9",
			"assignedToRouteStopId": "stop-2",
			"fields": [
				{"id": "f1", "label": "Notes", "type": "text", "textValue": {"value": "unwrapped"}}
			]
		}`),
	}

	data, err := event.FormData()
	require.NoError(t, err)
	assert.Equal(t, "form-sub-9", data.FormID)
	assert.Equal(t, "stop-2", data.AssignedToRouteStopID)
	require.Len(t, data.Fields, 1)
	assert.Equal(t, "unwrapped", data.Fields[0].Value)
}

func TestFormDataTypeMismatch(t *testing.T) {
	t.Parallel()

	event := Event{EventID: "evt-1", EventType: EventTypeRouteStopArrival}

	_, err := event.FormData()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventTypeMismatch)
}

func TestFormDataEmpty(t *testing.T) {
	t.Parallel()

	event := Event{EventID: "evt-1", EventType: EventTypeFormSubmitted}
	data, err := event.FormData()
	require.NoError(t, err)
	assert.Empty(t, data.FormID)
	assert.Empty(t, data.Fields)
}
