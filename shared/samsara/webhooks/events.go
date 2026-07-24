package webhooks

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

var (
	ErrEventInvalid      = errors.New("webhook event payload is invalid")
	ErrEventTypeMismatch = errors.New("webhook event type does not match requested payload")
)

type EventTypeName string

const (
	EventTypeAddressCreated             EventTypeName = "AddressCreated"
	EventTypeAddressDeleted             EventTypeName = "AddressDeleted"
	EventTypeAddressUpdated             EventTypeName = "AddressUpdated"
	EventTypeAlertIncident              EventTypeName = "AlertIncident"
	EventTypeAlertObjectEvent           EventTypeName = "AlertObjectEvent"
	EventTypeDocumentSubmitted          EventTypeName = "DocumentSubmitted"
	EventTypeDriverCreated              EventTypeName = "DriverCreated"
	EventTypeDriverUpdated              EventTypeName = "DriverUpdated"
	EventTypeDvirSubmitted              EventTypeName = "DvirSubmitted"
	EventTypeEngineFaultOff             EventTypeName = "EngineFaultOff"
	EventTypeEngineFaultOn              EventTypeName = "EngineFaultOn"
	EventTypeFormSubmitted              EventTypeName = "FormSubmitted"
	EventTypeFormUpdated                EventTypeName = "FormUpdated"
	EventTypeGatewayUnplugged           EventTypeName = "GatewayUnplugged"
	EventTypeGeofenceEntry              EventTypeName = "GeofenceEntry"
	EventTypeGeofenceExit               EventTypeName = "GeofenceExit"
	EventTypeIssueCreated               EventTypeName = "IssueCreated"
	EventTypeMissingDvirPastDue         EventTypeName = "MissingDvirPastDue"
	EventTypePredictiveMaintenanceAlert EventTypeName = "PredictiveMaintenanceAlert"
	EventTypeRouteStopArrival           EventTypeName = "RouteStopArrival"
	EventTypeRouteStopDeparture         EventTypeName = "RouteStopDeparture"
	EventTypeRouteStopEarlyLateArrival  EventTypeName = "RouteStopEarlyLateArrival"
	EventTypeRouteStopEtaUpdated        EventTypeName = "RouteStopEtaUpdated"
	EventTypeRouteStopResequence        EventTypeName = "RouteStopResequence"
	EventTypeSevereSpeedingEnded        EventTypeName = "SevereSpeedingEnded"
	EventTypeSevereSpeedingStarted      EventTypeName = "SevereSpeedingStarted"
	EventTypeSpeedingEventEnded         EventTypeName = "SpeedingEventEnded"
	EventTypeSpeedingEventStarted       EventTypeName = "SpeedingEventStarted"
	EventTypeSuddenFuelLevelDrop        EventTypeName = "SuddenFuelLevelDrop"
	EventTypeSuddenFuelLevelRise        EventTypeName = "SuddenFuelLevelRise"
	EventTypeVehicleCreated             EventTypeName = "VehicleCreated"
	EventTypeVehicleUpdated             EventTypeName = "VehicleUpdated"
)

type RawData []byte

func (r *RawData) UnmarshalJSON(data []byte) error {
	*r = append((*r)[:0], data...)
	return nil
}

func (r RawData) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return r, nil
}

type Event struct {
	EventID   string        `json:"eventId"`
	EventTime time.Time     `json:"eventTime"`
	EventType EventTypeName `json:"eventType"`
	OrgID     int64         `json:"orgId"`
	WebhookID string        `json:"webhookId"`
	Data      RawData       `json:"data"`
}

func ParseEvent(body []byte) (Event, error) {
	event := Event{}
	if err := sonic.Unmarshal(body, &event); err != nil {
		return Event{}, fmt.Errorf("parse webhook event: %w", err)
	}
	if event.EventID == "" || event.EventType == "" {
		return Event{}, ErrEventInvalid
	}
	return event, nil
}

type GeofenceEventData struct {
	Address *GeofenceEventAddress `json:"address,omitempty"`
	Vehicle *GeofenceEventVehicle `json:"vehicle,omitempty"`
	Driver  *GeofenceEventDriver  `json:"driver,omitempty"`
}

type GeofenceEventAddress struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	FormattedAddress string                 `json:"formattedAddress"`
	ExternalIDs      map[string]string      `json:"externalIds,omitempty"`
	Geofence         *GeofenceEventGeofence `json:"geofence,omitempty"`
}

type GeofenceEventGeofence struct {
	Circle  *GeofenceEventCircle  `json:"circle,omitempty"`
	Polygon *GeofenceEventPolygon `json:"polygon,omitempty"`
}

type GeofenceEventCircle struct {
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	RadiusMeters float64 `json:"radiusMeters"`
}

type GeofenceEventPolygon struct {
	Vertices []GeofenceEventVertex `json:"vertices"`
}

type GeofenceEventVertex struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type GeofenceEventVehicle struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	AssetType    string            `json:"assetType,omitempty"`
	LicensePlate string            `json:"licensePlate,omitempty"`
	Vin          string            `json:"vin,omitempty"`
	ExternalIDs  map[string]string `json:"externalIds,omitempty"`
}

type GeofenceEventDriver struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	ExternalIDs map[string]string `json:"externalIds,omitempty"`
}

type EntityEventData struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	ExternalIDs map[string]string `json:"externalIds,omitempty"`
}

type EntityEventRef = EntityEventData

func (e *Event) GeofenceData() (GeofenceEventData, error) {
	if e.EventType != EventTypeGeofenceEntry && e.EventType != EventTypeGeofenceExit {
		return GeofenceEventData{}, ErrEventTypeMismatch
	}
	if len(e.Data) == 0 {
		return GeofenceEventData{}, nil
	}

	out := GeofenceEventData{}
	if err := sonic.Unmarshal(e.Data, &out); err != nil {
		return GeofenceEventData{}, fmt.Errorf("decode geofence event data: %w", err)
	}
	return out, nil
}

func (e *Event) VehicleData() (EntityEventData, error) {
	if e.EventType != EventTypeVehicleCreated && e.EventType != EventTypeVehicleUpdated {
		return EntityEventData{}, ErrEventTypeMismatch
	}
	return decodeEntityData(e.Data, entityKeyVehicle)
}

func (e *Event) DriverData() (EntityEventData, error) {
	if e.EventType != EventTypeDriverCreated && e.EventType != EventTypeDriverUpdated {
		return EntityEventData{}, ErrEventTypeMismatch
	}
	return decodeEntityData(e.Data, entityKeyDriver)
}

type entityKey int

const (
	entityKeyVehicle entityKey = iota
	entityKeyDriver
)

type entityEnvelope struct {
	Vehicle     *EntityEventData  `json:"vehicle,omitempty"`
	Driver      *EntityEventData  `json:"driver,omitempty"`
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	ExternalIDs map[string]string `json:"externalIds,omitempty"`
}

func decodeEntityData(data RawData, key entityKey) (EntityEventData, error) {
	if len(data) == 0 {
		return EntityEventData{}, nil
	}

	env := entityEnvelope{}
	if err := sonic.Unmarshal(data, &env); err != nil {
		return EntityEventData{}, fmt.Errorf("decode entity event data: %w", err)
	}

	switch key {
	case entityKeyVehicle:
		if env.Vehicle != nil {
			return *env.Vehicle, nil
		}
	case entityKeyDriver:
		if env.Driver != nil {
			return *env.Driver, nil
		}
	}

	return EntityEventData{
		ID:          env.ID,
		Name:        env.Name,
		ExternalIDs: env.ExternalIDs,
	}, nil
}

type RouteStopEventData struct {
	Operation       string            `json:"operation"`
	Time            string            `json:"time"`
	AssignedToRoute string            `json:"assignedToRoute"`
	Driver          *EntityEventRef   `json:"driver,omitempty"`
	Vehicle         *RouteStopVehicle `json:"vehicle,omitempty"`
	Route           *RouteStopRoute   `json:"route,omitempty"`
	RouteStop       *RouteStopDetails `json:"routeStopDetails,omitempty"`
}

type RouteStopVehicle struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Vin          string            `json:"vin,omitempty"`
	LicensePlate string            `json:"licensePlate,omitempty"`
	ExternalIDs  map[string]string `json:"externalIds,omitempty"`
}

type RouteStopRoute struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	ExternalIDs map[string]string `json:"externalIds,omitempty"`
}

type RouteStopDetails struct {
	ID                  string            `json:"id"`
	State               string            `json:"state"`
	ETA                 string            `json:"eta,omitempty"`
	EnRouteTime         string            `json:"enRouteTime,omitempty"`
	ActualArrivalTime   string            `json:"actualArrivalTime,omitempty"`
	ActualDepartureTime string            `json:"actualDepartureTime,omitempty"`
	ExternalIDs         map[string]string `json:"externalIds,omitempty"`
	Orders              []RouteStopOrder  `json:"orders,omitempty"`
}

type RouteStopOrder struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	ExternalIDs map[string]string `json:"externalIds,omitempty"`
}

func (e *Event) RouteStopData() (RouteStopEventData, error) {
	//nolint:exhaustive // only route-stop event types decode route-stop data
	switch e.EventType {
	case EventTypeRouteStopArrival,
		EventTypeRouteStopDeparture,
		EventTypeRouteStopEtaUpdated,
		EventTypeRouteStopEarlyLateArrival:
	default:
		return RouteStopEventData{}, ErrEventTypeMismatch
	}

	if len(e.Data) == 0 {
		return RouteStopEventData{}, nil
	}

	out := RouteStopEventData{}
	if err := sonic.Unmarshal(e.Data, &out); err != nil {
		return RouteStopEventData{}, fmt.Errorf("decode route stop event data: %w", err)
	}
	return out, nil
}

type FormEventData struct {
	FormID                string             `json:"formId"`
	TemplateID            string             `json:"templateId"`
	TemplateRevisionID    string             `json:"templateRevisionId"`
	Status                string             `json:"status"`
	SubmittedAtTime       string             `json:"submittedAtTime"`
	AssignedToRouteStopID string             `json:"assignedToRouteStopId"`
	SubmittedBy           *FormEventActor    `json:"submittedBy,omitempty"`
	ExternalIDs           map[string]string  `json:"externalIds,omitempty"`
	Location              *FormEventLocation `json:"location,omitempty"`
	Fields                []FormEventField   `json:"fields"`
}

type FormEventActor struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type FormEventLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type FormEventField struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type formEnvelope struct {
	Form     *formEventPayload `json:"form,omitempty"`
	Document *formEventPayload `json:"document,omitempty"`
}

type formEventPayload struct {
	ID                    string             `json:"id"`
	FormTemplate          *formTemplateRef   `json:"formTemplate,omitempty"`
	TemplateID            string             `json:"templateId"`
	TemplateRevisionID    string             `json:"templateRevisionId"`
	Status                string             `json:"status"`
	SubmittedAtTime       string             `json:"submittedAtTime"`
	RouteStopID           string             `json:"routeStopId"`
	AssignedToRouteStopID string             `json:"assignedToRouteStopId"`
	SubmittedBy           *FormEventActor    `json:"submittedBy,omitempty"`
	ExternalIDs           map[string]string  `json:"externalIds,omitempty"`
	Location              *FormEventLocation `json:"location,omitempty"`
	Fields                []formFieldRaw     `json:"fields"`
}

type formTemplateRef struct {
	ID         string `json:"id"`
	RevisionID string `json:"revisionId"`
}

type formFieldRaw struct {
	ID                  string                 `json:"id"`
	Label               string                 `json:"label"`
	Type                string                 `json:"type"`
	NumberValue         *formNumberValueRaw    `json:"numberValue,omitempty"`
	TextValue           *formTextValueRaw      `json:"textValue,omitempty"`
	MultipleChoiceValue *formChoiceValueRaw    `json:"multipleChoiceValue,omitempty"`
	CheckBoxesValue     *formCheckBoxesRaw     `json:"checkBoxesValue,omitempty"`
	DateTimeValue       *formDateTimeRaw       `json:"dateTimeValue,omitempty"`
	SignatureValue      *formSignatureRaw      `json:"signatureValue,omitempty"`
	MediaValue          *formMediaRaw          `json:"mediaValue,omitempty"`
	Value               *documentFieldValueRaw `json:"value,omitempty"`
}

type formNumberValueRaw struct {
	Value float64 `json:"value"`
}

type formTextValueRaw struct {
	Value string `json:"value"`
}

type formChoiceValueRaw struct {
	Value   string `json:"value"`
	ValueID string `json:"valueId"`
}

type formCheckBoxesRaw struct {
	Value []string `json:"value"`
}

type formDateTimeRaw struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

type formSignatureRaw struct {
	Media formMediaRecordRaw `json:"media"`
}

type formMediaRaw struct {
	MediaList []formMediaRecordRaw `json:"mediaList"`
}

type formMediaRecordRaw struct {
	URL string `json:"url"`
}

type documentFieldValueRaw struct {
	StringValue string               `json:"stringValue"`
	NumberValue *float64             `json:"numberValue"`
	PhotoValue  []formMediaRecordRaw `json:"photoValue"`
}

func (e *Event) FormData() (FormEventData, error) {
	//nolint:exhaustive // only form/document event types decode form data
	switch e.EventType {
	case EventTypeFormSubmitted,
		EventTypeFormUpdated,
		EventTypeDocumentSubmitted:
	default:
		return FormEventData{}, ErrEventTypeMismatch
	}

	if len(e.Data) == 0 {
		return FormEventData{}, nil
	}

	env := formEnvelope{}
	if err := sonic.Unmarshal(e.Data, &env); err != nil {
		return FormEventData{}, fmt.Errorf("decode form event data: %w", err)
	}

	payload := env.Form
	if payload == nil {
		payload = env.Document
	}
	if payload == nil {
		payload = &formEventPayload{}
		if err := sonic.Unmarshal(e.Data, payload); err != nil {
			return FormEventData{}, fmt.Errorf("decode form event data: %w", err)
		}
	}

	return payload.normalize(), nil
}

func (p *formEventPayload) normalize() FormEventData {
	out := FormEventData{
		FormID:                p.ID,
		TemplateID:            p.TemplateID,
		TemplateRevisionID:    p.TemplateRevisionID,
		Status:                p.Status,
		SubmittedAtTime:       p.SubmittedAtTime,
		AssignedToRouteStopID: firstNonEmpty(p.AssignedToRouteStopID, p.RouteStopID),
		SubmittedBy:           p.SubmittedBy,
		ExternalIDs:           p.ExternalIDs,
		Location:              p.Location,
	}
	if p.FormTemplate != nil {
		if out.TemplateID == "" {
			out.TemplateID = p.FormTemplate.ID
		}
		if out.TemplateRevisionID == "" {
			out.TemplateRevisionID = p.FormTemplate.RevisionID
		}
	}

	out.Fields = make([]FormEventField, 0, len(p.Fields))
	for i := range p.Fields {
		field := &p.Fields[i]
		out.Fields = append(out.Fields, FormEventField{
			ID:    field.ID,
			Label: field.Label,
			Type:  field.Type,
			Value: field.flatten(),
		})
	}
	return out
}

func (f *formFieldRaw) flatten() string {
	switch {
	case f.NumberValue != nil:
		return formatFloat(f.NumberValue.Value)
	case f.TextValue != nil:
		return f.TextValue.Value
	case f.MultipleChoiceValue != nil:
		return f.MultipleChoiceValue.Value
	case f.CheckBoxesValue != nil:
		return strings.Join(f.CheckBoxesValue.Value, ", ")
	case f.DateTimeValue != nil:
		return f.DateTimeValue.Value
	case f.SignatureValue != nil:
		return f.SignatureValue.Media.URL
	case f.MediaValue != nil:
		return firstMediaURL(f.MediaValue.MediaList)
	case f.Value != nil:
		return f.Value.flatten()
	default:
		return ""
	}
}

func (v *documentFieldValueRaw) flatten() string {
	if url := firstMediaURL(v.PhotoValue); url != "" {
		return url
	}
	if v.NumberValue != nil {
		return formatFloat(*v.NumberValue)
	}
	return v.StringValue
}

func firstMediaURL(records []formMediaRecordRaw) string {
	for i := range records {
		if records[i].URL != "" {
			return records[i].URL
		}
	}
	return ""
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
